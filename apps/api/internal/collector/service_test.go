package collector_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/config"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/database"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

type mockCollector struct {
	name string
	jobs []collector.RawJob
	err  error
}

func (m *mockCollector) Name() string {
	return m.name
}

func (m *mockCollector) Collect(ctx context.Context, query collector.SearchQuery) ([]collector.RawJob, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.jobs, nil
}

func setupTestDB(t *testing.T) (*database.DB, *database.JobRepository) {
	t.Helper()

	if testing.Short() {
		t.Skip("pulando teste de integracao com banco real no modo -short")
	}

	if os.Getenv("DB_PORT") == "" {
		_ = os.Setenv("DB_PORT", "5433")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("falha ao carregar config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dbInstance, err := database.New(ctx, cfg)
	if err != nil {
		t.Skipf("PostgreSQL indisponível: %v", err)
		return nil, nil
	}

	// Limpa o banco antes de iniciar o teste
	_, _ = dbInstance.Pool.Exec(ctx, "TRUNCATE TABLE jobs CASCADE;")

	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cleanCancel()
		_, _ = dbInstance.Pool.Exec(cleanCtx, "TRUNCATE TABLE jobs CASCADE;")
		dbInstance.Close()
	})

	repo := database.NewJobRepository(dbInstance.Pool, nil)
	return dbInstance, repo
}

func TestCollectService_Orchestration(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	svc := collector.NewService(repo, nil, nil, nil)

	mockData := []collector.RawJob{
		{
			ExternalID:     "sc-001",
			Title:          "Técnico em Enfermagem - CTI",
			Company:        "Hospital Santa Casa",
			City:           "Porto Alegre",
			State:          "RS",
			Source:         "santacasa",
			SourceURL:      "https://santacasa.gupy.io/jobs/sc-001",
			WorkMode:       "on-site",
			EmploymentType: "clt",
		},
		{
			ExternalID:     "sc-002",
			Title:          "Técnico em Enfermagem - Bloco Cirúrgico",
			Company:        "Hospital Santa Casa",
			City:           "Porto Alegre",
			State:          "RS",
			Source:         "santacasa",
			SourceURL:      "https://santacasa.gupy.io/jobs/sc-002",
			WorkMode:       "on-site",
			EmploymentType: "clt",
		},
	}

	collectorA := &mockCollector{
		name: "santacasa",
		jobs: mockData,
	}

	t.Run("primeira execucao insere todas as novas vagas com sucesso", func(t *testing.T) {
		res, err := svc.CollectFrom(ctx, collectorA, collector.SearchQuery{})
		if err != nil {
			t.Fatalf("CollectFrom() erro inesperado: %v", err)
		}

		if res.TotalFound != 2 {
			t.Errorf("TotalFound = %d, esperado 2", res.TotalFound)
		}
		if res.Inserted != 2 {
			t.Errorf("Inserted = %d, esperado 2", res.Inserted)
		}
		if res.Updated != 0 {
			t.Errorf("Updated = %d, esperado 0", res.Updated)
		}
		if res.Duplicates != 0 {
			t.Errorf("Duplicates = %d, esperado 0", res.Duplicates)
		}
		if res.Failed != 0 {
			t.Errorf("Failed = %d, esperado 0", res.Failed)
		}

		job1, err := repo.FindBySourceAndExternalID(ctx, "santacasa", "sc-001")
		if err != nil {
			t.Fatalf("vaga sc-001 não encontrada no banco: %v", err)
		}
		if job1.Title != "Técnico em Enfermagem - CTI" {
			t.Errorf("título no banco incorreto: %s", job1.Title)
		}
	})

	t.Run("segunda execucao com mesmos dados e idempotente e atualiza last_seen_at", func(t *testing.T) {
		job1Before, err := repo.FindBySourceAndExternalID(ctx, "santacasa", "sc-001")
		if err != nil {
			t.Fatalf("erro ao buscar vaga antes: %v", err)
		}

		time.Sleep(10 * time.Millisecond) // Garante diferença no timestamp de last_seen_at

		res, err := svc.CollectFrom(ctx, collectorA, collector.SearchQuery{})
		if err != nil {
			t.Fatalf("CollectFrom() segunda execução erro: %v", err)
		}

		if res.Inserted != 0 {
			t.Errorf("esperava 0 inserções na repetição, obteve %d", res.Inserted)
		}
		if res.Updated != 2 {
			t.Errorf("esperava 2 atualizações na repetição, obteve %d", res.Updated)
		}

		job1After, err := repo.FindBySourceAndExternalID(ctx, "santacasa", "sc-001")
		if err != nil {
			t.Fatalf("erro ao buscar vaga após atualização: %v", err)
		}
		if !job1After.LastSeenAt.After(job1Before.LastSeenAt) {
			t.Errorf("LastSeenAt deveria ter sido atualizado para data posterior: before=%v, after=%v",
				job1Before.LastSeenAt, job1After.LastSeenAt)
		}
	})

	t.Run("detecta duplicata logica por fingerprint de outra fonte", func(t *testing.T) {
		// Vaga com id e fonte diferentes, mas mesma empresa, título e cidade (mesmo fingerprint)
		collectorB := &mockCollector{
			name: "agregador-externo",
			jobs: []collector.RawJob{
				{
					ExternalID:     "ext-999",
					Title:          "Técnico em Enfermagem - CTI",
					Company:        "Hospital Santa Casa",
					City:           "Porto Alegre",
					State:          "RS",
					Source:         "agregador-externo",
					SourceURL:      "https://outro-site.com/jobs/999",
					WorkMode:       "on-site",
					EmploymentType: "clt",
				},
			},
		}

		res, err := svc.CollectFrom(ctx, collectorB, collector.SearchQuery{})
		if err != nil {
			t.Fatalf("CollectFrom() erro: %v", err)
		}

		if res.Duplicates != 1 {
			t.Errorf("Duplicates = %d, esperado 1", res.Duplicates)
		}
		if res.Inserted != 0 {
			t.Errorf("Inserted = %d, esperado 0 para duplicata lógica", res.Inserted)
		}
	})

	t.Run("trata falha de validacao em item individual sem interromper os demais", func(t *testing.T) {
		collectorMixed := &mockCollector{
			name: "santacasa",
			jobs: []collector.RawJob{
				{
					ExternalID: "invalid-01",
					Title:      "", // Título vazio é inválido
					Company:    "Hospital Santa Casa",
					Source:     "santacasa",
					SourceURL:  "https://santacasa.gupy.io/jobs/invalid-01",
				},
				{
					ExternalID:     "sc-003",
					Title:          "Técnico em Enfermagem - Pediatria",
					Company:        "Hospital Santa Casa",
					City:           "Porto Alegre",
					State:          "RS",
					Source:         "santacasa",
					SourceURL:      "https://santacasa.gupy.io/jobs/sc-003",
					WorkMode:       "on-site",
					EmploymentType: "clt",
				},
			},
		}

		res, err := svc.CollectFrom(ctx, collectorMixed, collector.SearchQuery{})
		if err != nil {
			t.Fatalf("CollectFrom() erro: %v", err)
		}

		if res.TotalFound != 2 {
			t.Errorf("TotalFound = %d, esperado 2", res.TotalFound)
		}
		if res.Failed != 1 {
			t.Errorf("Failed = %d, esperado 1", res.Failed)
		}
		if res.Inserted != 1 {
			t.Errorf("Inserted = %d, esperado 1", res.Inserted)
		}
		if len(res.Errors) != 1 {
			t.Errorf("len(Errors) = %d, esperado 1", len(res.Errors))
		}
	})
}

func TestCollectService_EndToEndWithSantaCasaFixture(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	fixture, err := os.ReadFile("testdata/santacasa_fixture.html")
	if err != nil {
		t.Fatalf("falha ao carregar fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(fixture)
	}))
	defer server.Close()

	santaCasaCollector := collector.NewSantaCasaCollectorWithURL(server.URL, server.Client(), 5*time.Second)
	svc := collector.NewService(repo, nil, nil, nil)

	// 1. Coleta e persistência com filtro para Técnico
	res, err := svc.CollectFrom(ctx, santaCasaCollector, collector.SearchQuery{Query: "técnico"})
	if err != nil {
		t.Fatalf("CollectFrom() com Santa Casa erro inesperado: %v", err)
	}

	if res.Inserted != 1 {
		t.Errorf("esperava 1 inserção de vaga filtrada, obteve %d", res.Inserted)
	}

	saved, err := repo.FindBySourceAndExternalID(ctx, "santacasa", "11889779")
	if err != nil {
		t.Fatalf("vaga 11889779 não foi encontrada no banco: %v", err)
	}
	if saved.Title != "Técnico em Enfermagem - Centro Cirúrgico" {
		t.Errorf("título no banco = %q", saved.Title)
	}
	if saved.Company != "Santa Casa de Porto Alegre" {
		t.Errorf("empresa no banco = %q", saved.Company)
	}
	if saved.WorkMode != job.WorkModeOnSite {
		t.Errorf("work mode = %q, esperado ON_SITE", saved.WorkMode)
	}

	// 2. Coleta de todas as vagas (adiciona as demais e é idempotente na primeira)
	resAll, err := svc.CollectFrom(ctx, santaCasaCollector, collector.SearchQuery{})
	if err != nil {
		t.Fatalf("segunda coleta erro: %v", err)
	}

	if resAll.TotalFound != 3 {
		t.Errorf("TotalFound = %d, esperado 3", resAll.TotalFound)
	}
	if resAll.Inserted != 2 { // 2 novas (Enfermeiro e Advogado)
		t.Errorf("esperava 2 novas inserções, obteve %d", resAll.Inserted)
	}
	if resAll.Updated != 1 { // 1 atualizada (Técnico previamente inserida)
		t.Errorf("esperava 1 atualização, obteve %d", resAll.Updated)
	}
}

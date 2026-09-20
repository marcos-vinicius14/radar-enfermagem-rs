package database_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/config"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/database"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

// setupIntegrationTest inicializa a conexão com o banco de dados PostgreSQL real
// e registra o cleanup para truncar a tabela após a execução dos testes.
func setupIntegrationTest(t *testing.T) (*database.DB, *database.JobRepository) {
	t.Helper()

	if testing.Short() {
		t.Skip("pulando teste de integração em modo -short")
	}

	if os.Getenv("DB_PORT") == "" {
		_ = os.Setenv("DB_PORT", "5433")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("falha ao carregar configuração de teste: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dbInstance, err := database.New(ctx, cfg)
	if err != nil {
		t.Skipf("pulando teste de integracao: PostgreSQL indisponivel: %v", err)
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

func newValidTestJob(source, externalID string) job.Job {
	minSalary := int64(3200)
	maxSalary := int64(4500)
	pubDate := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Microsecond)
	now := time.Now().UTC().Truncate(time.Microsecond)

	return job.Job{
		ExternalID:     externalID,
		Title:          "Técnico de Enfermagem - CTI Adulto",
		Company:        "Hospital Moinhos de Vento",
		Description:    "Atividades assistenciais e cuidados intensivos",
		City:           "Porto Alegre",
		State:          "RS",
		Source:         source,
		SourceURL:      "https://hospitalmoinhos.gupy.io/job/1",
		Fingerprint:    job.Fingerprint("Hospital Moinhos de Vento", "Técnico de Enfermagem - CTI Adulto", "Porto Alegre"),
		WorkMode:       job.WorkModeOnSite,
		EmploymentType: job.EmploymentTypeFullTime,
		SalaryMin:      &minSalary,
		SalaryMax:      &maxSalary,
		PublishedAt:    &pubDate,
		CollectedAt:    now,
		LastSeenAt:     now,
		Status:         job.StatusActive,
	}
}

func TestJobRepository_Insert_And_FindByID(t *testing.T) {
	_, repo := setupIntegrationTest(t)
	ctx := context.Background()

	t.Run("insere vaga com id nil e postgres 18 gera uuidv7 nativo", func(t *testing.T) {
		input := newValidTestJob("portal_moinhos", "vaga-001")
		input.ID = uuid.Nil

		created, err := repo.Insert(ctx, input)
		if err != nil {
			t.Fatalf("Insert() erro inesperado: %v", err)
		}

		if created.ID == uuid.Nil {
			t.Fatalf("esperava que PostgreSQL gerasse UUIDv7 não nulo, obteve uuid.Nil")
		}
		if created.Title != input.Title {
			t.Errorf("Title = %q, esperado %q", created.Title, input.Title)
		}
		if created.Status != job.StatusActive {
			t.Errorf("Status = %q, esperado %q", created.Status, job.StatusActive)
		}

		found, err := repo.FindByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("FindByID() erro: %v", err)
		}
		if found.ID != created.ID {
			t.Errorf("FindByID ID = %s, esperado %s", found.ID, created.ID)
		}
		if found.Fingerprint != input.Fingerprint {
			t.Errorf("Fingerprint = %s, esperado %s", found.Fingerprint, input.Fingerprint)
		}
	})

	t.Run("insere vaga com id pre-gerado em Go", func(t *testing.T) {
		customID := uuid.New()
		input := newValidTestJob("portal_saolucas", "vaga-002")
		input.ID = customID

		created, err := repo.Insert(ctx, input)
		if err != nil {
			t.Fatalf("Insert() erro: %v", err)
		}
		if created.ID != customID {
			t.Errorf("Insert() ID = %s, esperado %s", created.ID, customID)
		}

		found, err := repo.FindByID(ctx, customID)
		if err != nil {
			t.Fatalf("FindByID() erro: %v", err)
		}
		if found.ID != customID {
			t.Errorf("FindByID() ID = %s, esperado %s", found.ID, customID)
		}
	})
}

func TestJobRepository_FindBySourceAndExternalID(t *testing.T) {
	_, repo := setupIntegrationTest(t)
	ctx := context.Background()

	source := "santacasa"
	extID := "sc-9988"

	input := newValidTestJob(source, extID)
	created, err := repo.Insert(ctx, input)
	if err != nil {
		t.Fatalf("Insert() erro: %v", err)
	}

	t.Run("retorna vaga correspondente quando existe", func(t *testing.T) {
		found, err := repo.FindBySourceAndExternalID(ctx, source, extID)
		if err != nil {
			t.Fatalf("FindBySourceAndExternalID() erro: %v", err)
		}
		if found.ID != created.ID {
			t.Errorf("ID = %s, esperado %s", found.ID, created.ID)
		}
		if found.Source != source || found.ExternalID != extID {
			t.Errorf("Source/ExternalID inconsistentes: %s / %s", found.Source, found.ExternalID)
		}
	})

	t.Run("retorna ErrNotFound quando vaga nao existe", func(t *testing.T) {
		_, err := repo.FindBySourceAndExternalID(ctx, "fonte_fantasma", "nao-existe")
		if err == nil {
			t.Fatalf("esperava erro ErrNotFound, obteve nil")
		}
		if !errors.Is(err, job.ErrNotFound) {
			t.Fatalf("esperava errors.Is(err, job.ErrNotFound), obteve: %v", err)
		}
	})
}

func TestJobRepository_Update(t *testing.T) {
	_, repo := setupIntegrationTest(t)
	ctx := context.Background()

	original := newValidTestJob("unimed", "uni-450")
	created, err := repo.Insert(ctx, original)
	if err != nil {
		t.Fatalf("Insert() erro: %v", err)
	}

	t.Run("atualiza dados de vaga existente e incrementa updated_at", func(t *testing.T) {
		toUpdate := created
		toUpdate.Title = "Técnico de Enfermagem - Pronto Atendimento"
		toUpdate.Status = job.StatusUnknown
		newMax := int64(5200)
		toUpdate.SalaryMax = &newMax

		updated, err := repo.Update(ctx, toUpdate)
		if err != nil {
			t.Fatalf("Update() erro inesperado: %v", err)
		}

		if updated.Title != toUpdate.Title {
			t.Errorf("Title = %q, esperado %q", updated.Title, toUpdate.Title)
		}
		if updated.Status != job.StatusUnknown {
			t.Errorf("Status = %q, esperado %q", updated.Status, job.StatusUnknown)
		}
		if *updated.SalaryMax != newMax {
			t.Errorf("SalaryMax = %d, esperado %d", *updated.SalaryMax, newMax)
		}
	})

	t.Run("retorna ErrNotFound ao tentar atualizar vaga inexistente", func(t *testing.T) {
		ghostJob := created
		ghostJob.ID = uuid.New()

		_, err := repo.Update(ctx, ghostJob)
		if err == nil {
			t.Fatalf("esperava erro para vaga inexistente, obteve nil")
		}
		if !errors.Is(err, job.ErrNotFound) {
			t.Fatalf("esperava errors.Is(err, job.ErrNotFound), obteve: %v", err)
		}
	})
}

func TestJobRepository_UpdateLastSeen(t *testing.T) {
	_, repo := setupIntegrationTest(t)
	ctx := context.Background()

	jobItem, err := repo.Insert(ctx, newValidTestJob("doctorclin", "dc-111"))
	if err != nil {
		t.Fatalf("Insert() erro: %v", err)
	}

	t.Run("atualiza last_seen_at com sucesso", func(t *testing.T) {
		newLastSeen := time.Now().Add(1 * time.Hour).UTC().Truncate(time.Microsecond)
		err := repo.UpdateLastSeen(ctx, jobItem.ID, newLastSeen)
		if err != nil {
			t.Fatalf("UpdateLastSeen() erro: %v", err)
		}

		found, err := repo.FindByID(ctx, jobItem.ID)
		if err != nil {
			t.Fatalf("FindByID() erro: %v", err)
		}

		if found.LastSeenAt.Sub(newLastSeen) > time.Second || newLastSeen.Sub(found.LastSeenAt) > time.Second {
			t.Errorf("LastSeenAt = %v, esperado próximo de %v", found.LastSeenAt, newLastSeen)
		}
	})

	t.Run("retorna ErrNotFound quando vaga nao existe", func(t *testing.T) {
		err := repo.UpdateLastSeen(ctx, uuid.New(), time.Now())
		if err == nil {
			t.Fatalf("esperava erro, obteve nil")
		}
		if !errors.Is(err, job.ErrNotFound) {
			t.Fatalf("esperava errors.Is(err, job.ErrNotFound), obteve: %v", err)
		}
	})
}

func TestJobRepository_List(t *testing.T) {
	_, repo := setupIntegrationTest(t)
	ctx := context.Background()

	// Insere 4 vagas com timestamps distintos de publicação
	t1 := time.Now().Add(-4 * time.Hour).UTC().Truncate(time.Microsecond)
	t2 := time.Now().Add(-3 * time.Hour).UTC().Truncate(time.Microsecond)
	t3 := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Microsecond)
	t4 := time.Now().Add(-1 * time.Hour).UTC().Truncate(time.Microsecond)

	j1 := newValidTestJob("hcpa", "hcpa-1")
	j1.PublishedAt = &t1
	j2 := newValidTestJob("hcpa", "hcpa-2")
	j2.PublishedAt = &t2
	j3 := newValidTestJob("hcpa", "hcpa-3")
	j3.PublishedAt = &t3
	j4 := newValidTestJob("hcpa", "hcpa-4")
	j4.PublishedAt = &t4

	for _, j := range []job.Job{j1, j2, j3, j4} {
		if _, err := repo.Insert(ctx, j); err != nil {
			t.Fatalf("Insert() erro: %v", err)
		}
	}

	t.Run("lista com ordenacao deterministica decrescente por published_at", func(t *testing.T) {
		jobs, err := repo.List(ctx, job.ListParams{Limit: 10, Offset: 0})
		if err != nil {
			t.Fatalf("List() erro: %v", err)
		}
		if len(jobs) != 4 {
			t.Fatalf("len(jobs) = %d, esperado 4", len(jobs))
		}

		// Ordem esperada: hcpa-4 (mais recente), hcpa-3, hcpa-2, hcpa-1
		expectedOrder := []string{"hcpa-4", "hcpa-3", "hcpa-2", "hcpa-1"}
		for i, exp := range expectedOrder {
			if jobs[i].ExternalID != exp {
				t.Errorf("posicao %d: ExternalID = %s, esperado %s", i, jobs[i].ExternalID, exp)
			}
		}
	})

	t.Run("respeita limit e offset na paginacao", func(t *testing.T) {
		page1, err := repo.List(ctx, job.ListParams{Limit: 2, Offset: 0})
		if err != nil {
			t.Fatalf("List() page 1 erro: %v", err)
		}
		if len(page1) != 2 {
			t.Fatalf("len(page1) = %d, esperado 2", len(page1))
		}
		if page1[0].ExternalID != "hcpa-4" || page1[1].ExternalID != "hcpa-3" {
			t.Errorf("page 1 resultados inesperados: %s, %s", page1[0].ExternalID, page1[1].ExternalID)
		}

		page2, err := repo.List(ctx, job.ListParams{Limit: 2, Offset: 2})
		if err != nil {
			t.Fatalf("List() page 2 erro: %v", err)
		}
		if len(page2) != 2 {
			t.Fatalf("len(page2) = %d, esperado 2", len(page2))
		}
		if page2[0].ExternalID != "hcpa-2" || page2[1].ExternalID != "hcpa-1" {
			t.Errorf("page 2 resultados inesperados: %s, %s", page2[0].ExternalID, page2[1].ExternalID)
		}
	})

	t.Run("aplica limite default quando limit <= 0", func(t *testing.T) {
		jobs, err := repo.List(ctx, job.ListParams{Limit: 0, Offset: 0})
		if err != nil {
			t.Fatalf("List() erro: %v", err)
		}
		if len(jobs) != 4 {
			t.Fatalf("esperava 4 itens com limite padrão, obteve %d", len(jobs))
		}
	})
}

func TestJobRepository_UniqueConstraintViolation(t *testing.T) {
	_, repo := setupIntegrationTest(t)
	ctx := context.Background()

	source := "fleury"
	extID := "fl-990"

	job1 := newValidTestJob(source, extID)
	if _, err := repo.Insert(ctx, job1); err != nil {
		t.Fatalf("Insert() primeira inserção falhou: %v", err)
	}

	// Tentativa de duplicar a mesma fonte e ID externo
	job2 := newValidTestJob(source, extID)
	job2.Title = "Outro Título Qualquer"

	_, err := repo.Insert(ctx, job2)
	if err == nil {
		t.Fatalf("esperava erro de violação de constraint UNIQUE(source, external_id), obteve nil")
	}
}

func TestJobRepository_EdgeCases(t *testing.T) {
	_, repo := setupIntegrationTest(t)
	ctx := context.Background()

	t.Run("vaga com campos nulos opcionais (salarios nulos, published_at nulo)", func(t *testing.T) {
		input := newValidTestJob("divina", "div-001")
		input.SalaryMin = nil
		input.SalaryMax = nil
		input.PublishedAt = nil

		created, err := repo.Insert(ctx, input)
		if err != nil {
			t.Fatalf("Insert() erro com campos opcionais nulos: %v", err)
		}
		if created.SalaryMin != nil {
			t.Errorf("SalaryMin = %v, esperado nil", created.SalaryMin)
		}
		if created.SalaryMax != nil {
			t.Errorf("SalaryMax = %v, esperado nil", created.SalaryMax)
		}
		if created.PublishedAt != nil {
			t.Errorf("PublishedAt = %v, esperado nil", created.PublishedAt)
		}

		found, err := repo.FindByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("FindByID() erro: %v", err)
		}
		if found.SalaryMin != nil || found.SalaryMax != nil || found.PublishedAt != nil {
			t.Errorf("campos opcionais nulos persistidos incorretamente")
		}
	})

	t.Run("FindByID com id aleatorio inexistente retorna ErrNotFound", func(t *testing.T) {
		_, err := repo.FindByID(ctx, uuid.New())
		if err == nil {
			t.Fatalf("esperava erro, obteve nil")
		}
		if !errors.Is(err, job.ErrNotFound) {
			t.Fatalf("esperava errors.Is(err, job.ErrNotFound), obteve: %v", err)
		}
	})

	t.Run("Insert com dados invalidos falha na validacao de dominio antes do banco", func(t *testing.T) {
		invalid := newValidTestJob("maededeus", "mae-01")
		invalid.Title = ""

		_, err := repo.Insert(ctx, invalid)
		if err == nil {
			t.Fatalf("esperava erro de validação de título vazio, obteve nil")
		}
	})
}

func TestJobRepository_FindByFingerprint(t *testing.T) {
	_, repo := setupIntegrationTest(t)
	ctx := context.Background()

	testJob := newValidTestJob("santacasa", "sc-fingerprint-test")
	created, err := repo.Insert(ctx, testJob)
	if err != nil {
		t.Fatalf("Insert() erro: %v", err)
	}

	t.Run("retorna vaga quando fingerprint existe", func(t *testing.T) {
		found, err := repo.FindByFingerprint(ctx, created.Fingerprint)
		if err != nil {
			t.Fatalf("FindByFingerprint() erro inesperado: %v", err)
		}
		if found.ID != created.ID {
			t.Errorf("ID = %v, esperado %v", found.ID, created.ID)
		}
		if found.Fingerprint != created.Fingerprint {
			t.Errorf("Fingerprint = %q, esperado %q", found.Fingerprint, created.Fingerprint)
		}
	})

	t.Run("retorna ErrNotFound quando fingerprint nao existe", func(t *testing.T) {
		unknownFingerprint := "0000000000000000000000000000000000000000000000000000000000000000"
		_, err := repo.FindByFingerprint(ctx, unknownFingerprint)
		if err == nil {
			t.Fatalf("esperava erro para fingerprint inexistente, obteve nil")
		}
		if !errors.Is(err, job.ErrNotFound) {
			t.Fatalf("esperava errors.Is(err, job.ErrNotFound), obteve: %v", err)
		}
	})
}

func TestJobRepository_ReconcileStatuses(t *testing.T) {
	_, repo := setupIntegrationTest(t)
	ctx := context.Background()

	now := time.Now().UTC()

	// Vaga 1: ACTIVE vista há 30 horas (deve virar UNKNOWN)
	j1 := newValidTestJob("santacasa", "job-reconcile-1")
	j1.Status = job.StatusActive
	j1.LastSeenAt = now.Add(-30 * time.Hour)
	saved1, err := repo.Insert(ctx, j1)
	if err != nil {
		t.Fatalf("Insert(j1) falhou: %v", err)
	}

	// Vaga 2: UNKNOWN vista há 8 dias (deve virar EXPIRED)
	j2 := newValidTestJob("moinhos", "job-reconcile-2")
	j2.Status = job.StatusUnknown
	j2.LastSeenAt = now.Add(-8 * 24 * time.Hour)
	saved2, err := repo.Insert(ctx, j2)
	if err != nil {
		t.Fatalf("Insert(j2) falhou: %v", err)
	}

	// Vaga 3: ACTIVE vista há 1 hora (deve continuar ACTIVE)
	j3 := newValidTestJob("hcpa", "job-reconcile-3")
	j3.Status = job.StatusActive
	j3.LastSeenAt = now.Add(-1 * time.Hour)
	saved3, err := repo.Insert(ctx, j3)
	if err != nil {
		t.Fatalf("Insert(j3) falhou: %v", err)
	}

	unknownBefore := now.Add(-24 * time.Hour)
	expiredBefore := now.Add(-7 * 24 * time.Hour)

	result, err := repo.ReconcileStatuses(ctx, unknownBefore, expiredBefore)
	if err != nil {
		t.Fatalf("ReconcileStatuses() erro: %v", err)
	}

	if result.MarkedUnknown != 1 {
		t.Errorf("MarkedUnknown = %d, esperado 1", result.MarkedUnknown)
	}
	if result.MarkedExpired != 1 {
		t.Errorf("MarkedExpired = %d, esperado 1", result.MarkedExpired)
	}

	// Verifica se a vaga 1 agora é UNKNOWN
	check1, err := repo.FindByID(ctx, saved1.ID)
	if err != nil {
		t.Fatalf("FindByID(saved1.ID) erro: %v", err)
	}
	if check1.Status != job.StatusUnknown {
		t.Errorf("status de saved1 = %s, esperado %s", check1.Status, job.StatusUnknown)
	}

	// Verifica se a vaga 2 agora é EXPIRED
	check2, err := repo.FindByID(ctx, saved2.ID)
	if err != nil {
		t.Fatalf("FindByID(saved2.ID) erro: %v", err)
	}
	if check2.Status != job.StatusExpired {
		t.Errorf("status de saved2 = %s, esperado %s", check2.Status, job.StatusExpired)
	}

	// Verifica se a vaga 3 permaneceu ACTIVE
	check3, err := repo.FindByID(ctx, saved3.ID)
	if err != nil {
		t.Fatalf("FindByID(saved3.ID) erro: %v", err)
	}
	if check3.Status != job.StatusActive {
		t.Errorf("status de saved3 = %s, esperado %s", check3.Status, job.StatusActive)
	}
}

func TestJobRepository_Search(t *testing.T) {
	_, repo := setupIntegrationTest(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Microsecond)
	t1 := now.Add(-10 * time.Hour)
	t2 := now.Add(-24 * time.Hour)
	t3 := now.Add(-72 * time.Hour)
	t4 := now.Add(-120 * time.Hour)

	jobsToInsert := []job.Job{
		{
			ExternalID:     "search-001",
			Title:          "Técnico de Enfermagem - CTI Adulto",
			Company:        "Hospital Moinhos de Vento",
			Description:    "Plantão 12x36 diurno em terapia intensiva",
			City:           "Porto Alegre",
			State:          "RS",
			Source:         "moinhos",
			SourceURL:      "https://moinhos.com/jobs/001",
			Fingerprint:    "fp-search-001",
			WorkMode:       job.WorkModeOnSite,
			EmploymentType: job.EmploymentTypeFullTime,
			PublishedAt:    &t1,
			Status:         job.StatusActive,
		},
		{
			ExternalID:     "search-002",
			Title:          "Técnico de Enfermagem - Pediatria",
			Company:        "Hospital Santa Casa",
			Description:    "Atendimento infantil e berçário",
			City:           "Porto Alegre",
			State:          "RS",
			Source:         "santacasa",
			SourceURL:      "https://santacasa.com/jobs/002",
			Fingerprint:    "fp-search-002",
			WorkMode:       job.WorkModeOnSite,
			EmploymentType: job.EmploymentTypeFullTime,
			PublishedAt:    &t2,
			Status:         job.StatusActive,
		},
		{
			ExternalID:     "search-003",
			Title:          "Enfermeiro Auditor Clínico",
			Company:        "Unimed Porto Alegre",
			Description:    "Auditoria técnica hospitalar",
			City:           "Canoas",
			State:          "RS",
			Source:         "unimed",
			SourceURL:      "https://unimed.com/jobs/003",
			Fingerprint:    "fp-search-003",
			WorkMode:       job.WorkModeHybrid,
			EmploymentType: job.EmploymentTypeFullTime,
			PublishedAt:    &t3,
			Status:         job.StatusActive,
		},
		{
			ExternalID:     "search-004",
			Title:          "Técnico em Enfermagem - Bloco Cirúrgico",
			Company:        "Hospital Divina Providência",
			Description:    "Instrumentação cirúrgica e recuperação pós-anestésica",
			City:           "Porto Alegre",
			State:          "RS",
			Source:         "divina",
			SourceURL:      "https://divina.com/jobs/004",
			Fingerprint:    "fp-search-004",
			WorkMode:       job.WorkModeOnSite,
			EmploymentType: job.EmploymentTypeFullTime,
			PublishedAt:    &t4,
			Status:         job.StatusUnknown,
		},
	}

	for _, j := range jobsToInsert {
		if _, err := repo.Insert(ctx, j); err != nil {
			t.Fatalf("falha ao inserir vaga de teste: %v", err)
		}
	}

	t.Run("busca textual por query em titulo e descricao", func(t *testing.T) {
		res, err := repo.Search(ctx, job.FilterParams{
			Query: "Pediatria",
			Page:  1,
			Size:  10,
		})
		if err != nil {
			t.Fatalf("Search() erro: %v", err)
		}
		if res.Total != 1 {
			t.Errorf("Total = %d, esperado 1", res.Total)
		}
		if len(res.Items) != 1 || res.Items[0].ExternalID != "search-002" {
			t.Errorf("vaga inesperada retornada: %+v", res.Items)
		}
	})

	t.Run("filtro por cidade e empresa", func(t *testing.T) {
		res, err := repo.Search(ctx, job.FilterParams{
			City:    "Canoas",
			Company: "Unimed",
			Page:    1,
			Size:    10,
		})
		if err != nil {
			t.Fatalf("Search() erro: %v", err)
		}
		if res.Total != 1 {
			t.Errorf("Total = %d, esperado 1", res.Total)
		}
		if len(res.Items) != 1 || res.Items[0].Company != "Unimed Porto Alegre" {
			t.Errorf("vaga inesperada: %+v", res.Items)
		}
	})

	t.Run("filtro por status explicito UNKNOWN", func(t *testing.T) {
		res, err := repo.Search(ctx, job.FilterParams{
			Status: "UNKNOWN",
			Page:   1,
			Size:   10,
		})
		if err != nil {
			t.Fatalf("Search() erro: %v", err)
		}
		if res.Total != 1 {
			t.Errorf("Total = %d, esperado 1", res.Total)
		}
		if len(res.Items) != 1 || res.Items[0].ExternalID != "search-004" {
			t.Errorf("esperado search-004, obteve: %+v", res.Items)
		}
	})

	t.Run("filtro por data de publicacao published_since", func(t *testing.T) {
		since := now.Add(-30 * time.Hour) // deve incluir search-001 (10h) e search-002 (24h)
		res, err := repo.Search(ctx, job.FilterParams{
			PublishedSince: &since,
			Page:           1,
			Size:           10,
		})
		if err != nil {
			t.Fatalf("Search() erro: %v", err)
		}
		if res.Total != 2 {
			t.Errorf("Total = %d, esperado 2", res.Total)
		}
	})

	t.Run("paginacao e ordenacao deterministica", func(t *testing.T) {
		// Sem filtros específicos, deve retornar todas ordenadas por published_at DESC
		page1, err := repo.Search(ctx, job.FilterParams{
			Page: 1,
			Size: 2,
		})
		if err != nil {
			t.Fatalf("Search() page 1 erro: %v", err)
		}
		if page1.Total != 4 {
			t.Errorf("Total = %d, esperado 4", page1.Total)
		}
		if page1.TotalPages != 2 {
			t.Errorf("TotalPages = %d, esperado 2", page1.TotalPages)
		}
		if len(page1.Items) != 2 {
			t.Fatalf("esperado 2 itens na page 1, obteve %d", len(page1.Items))
		}
		if page1.Items[0].ExternalID != "search-001" || page1.Items[1].ExternalID != "search-002" {
			t.Errorf("ordenação inesperada na page 1: [%s, %s]", page1.Items[0].ExternalID, page1.Items[1].ExternalID)
		}

		page2, err := repo.Search(ctx, job.FilterParams{
			Page: 2,
			Size: 2,
		})
		if err != nil {
			t.Fatalf("Search() page 2 erro: %v", err)
		}
		if len(page2.Items) != 2 {
			t.Fatalf("esperado 2 itens na page 2, obteve %d", len(page2.Items))
		}
		if page2.Items[0].ExternalID != "search-003" || page2.Items[1].ExternalID != "search-004" {
			t.Errorf("ordenação inesperada na page 2: [%s, %s]", page2.Items[0].ExternalID, page2.Items[1].ExternalID)
		}
	})
}

func TestJobRepository_Aggregations(t *testing.T) {
	_, repo := setupIntegrationTest(t)
	ctx := context.Background()

	jobs := []job.Job{
		{
			ExternalID:     "agg-01",
			Title:          "Vaga 1",
			Company:        "Hospital Santa Casa",
			City:           "Porto Alegre",
			State:          "RS",
			Source:         "santacasa",
			SourceURL:      "https://sc.com/1",
			Fingerprint:    "fp-agg-01",
			WorkMode:       job.WorkModeOnSite,
			EmploymentType: job.EmploymentTypeFullTime,
			Status:         job.StatusActive,
		},
		{
			ExternalID:     "agg-02",
			Title:          "Vaga 2",
			Company:        "Hospital Santa Casa",
			City:           "Porto Alegre",
			State:          "RS",
			Source:         "santacasa",
			SourceURL:      "https://sc.com/2",
			Fingerprint:    "fp-agg-02",
			WorkMode:       job.WorkModeOnSite,
			EmploymentType: job.EmploymentTypeFullTime,
			Status:         job.StatusActive,
		},
		{
			ExternalID:     "agg-03",
			Title:          "Vaga 3",
			Company:        "Hospital Moinhos de Vento",
			City:           "Canoas",
			State:          "RS",
			Source:         "moinhos",
			SourceURL:      "https://hmv.com/3",
			Fingerprint:    "fp-agg-03",
			WorkMode:       job.WorkModeOnSite,
			EmploymentType: job.EmploymentTypeFullTime,
			Status:         job.StatusActive,
		},
		{
			ExternalID:     "agg-04",
			Title:          "Vaga 4",
			Company:        "Hospital Moinhos de Vento",
			City:           "Canoas",
			State:          "RS",
			Source:         "moinhos",
			SourceURL:      "https://hmv.com/4",
			Fingerprint:    "fp-agg-04",
			WorkMode:       job.WorkModeOnSite,
			EmploymentType: job.EmploymentTypeFullTime,
			Status:         job.StatusExpired, // Não-ativa
		},
	}

	for _, j := range jobs {
		if _, err := repo.Insert(ctx, j); err != nil {
			t.Fatalf("falha ao inserir vaga para agregação: %v", err)
		}
	}

	t.Run("ListCompanies com status ACTIVE", func(t *testing.T) {
		companies, err := repo.ListCompanies(ctx, "ACTIVE")
		if err != nil {
			t.Fatalf("ListCompanies erro: %v", err)
		}
		if len(companies) != 2 {
			t.Fatalf("esperado 2 empresas ativas, obteve %d", len(companies))
		}
		// Ordenação alfabética: Hospital Moinhos de Vento (1 ativa), Hospital Santa Casa (2 ativas)
		if companies[0].Name != "Hospital Moinhos de Vento" || companies[0].TotalJobs != 1 {
			t.Errorf("empresa 0 inesperada: %+v", companies[0])
		}
		if companies[1].Name != "Hospital Santa Casa" || companies[1].TotalJobs != 2 {
			t.Errorf("empresa 1 inesperada: %+v", companies[1])
		}
	})

	t.Run("ListCities com status ACTIVE", func(t *testing.T) {
		cities, err := repo.ListCities(ctx, "ACTIVE")
		if err != nil {
			t.Fatalf("ListCities erro: %v", err)
		}
		if len(cities) != 2 {
			t.Fatalf("esperado 2 cidades ativas, obteve %d", len(cities))
		}
		if cities[0].City != "Canoas" || cities[0].TotalJobs != 1 {
			t.Errorf("cidade 0 inesperada: %+v", cities[0])
		}
		if cities[1].City != "Porto Alegre" || cities[1].TotalJobs != 2 {
			t.Errorf("cidade 1 inesperada: %+v", cities[1])
		}
	})

	t.Run("ListSources com todas as vagas", func(t *testing.T) {
		sources, err := repo.ListSources(ctx, "")
		if err != nil {
			t.Fatalf("ListSources erro: %v", err)
		}
		if len(sources) != 2 {
			t.Fatalf("esperado 2 fontes, obteve %d", len(sources))
		}
		if sources[0].Source != "moinhos" || sources[0].TotalJobs != 2 {
			t.Errorf("fonte 0 inesperada: %+v", sources[0])
		}
		if sources[1].Source != "santacasa" || sources[1].TotalJobs != 2 {
			t.Errorf("fonte 1 inesperada: %+v", sources[1])
		}
	})
}

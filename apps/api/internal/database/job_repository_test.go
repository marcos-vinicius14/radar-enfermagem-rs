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

package job_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

type mockRepository struct {
	job.Repository
	jobs       []job.Job
	deletedIDs []uuid.UUID
}

func (m *mockRepository) ListActiveForPruning(ctx context.Context, limit, offset int32) ([]job.Job, error) {
	if int(offset) >= len(m.jobs) {
		return []job.Job{}, nil
	}
	end := int(offset + limit)
	if end > len(m.jobs) {
		end = len(m.jobs)
	}
	return m.jobs[offset:end], nil
}

func (m *mockRepository) DeleteByIDs(ctx context.Context, ids []uuid.UUID) (int64, error) {
	m.deletedIDs = append(m.deletedIDs, ids...)

	// Filtra os jobs restantes
	remaining := make([]job.Job, 0, len(m.jobs))
	deleteMap := make(map[uuid.UUID]bool)
	for _, id := range ids {
		deleteMap[id] = true
	}
	for _, j := range m.jobs {
		if !deleteMap[j.ID] {
			remaining = append(remaining, j)
		}
	}
	m.jobs = remaining
	return int64(len(ids)), nil
}

func TestPruner_Prune(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()
	id4 := uuid.New()

	mockRepo := &mockRepository{
		jobs: []job.Job{
			{
				ID:          id1,
				Title:       "Técnico de Enfermagem - UTI",
				City:        "Porto Alegre",
				State:       "RS",
				Company:     "Hospital Santa Casa",
				Status:      job.StatusActive,
				CollectedAt: time.Now(),
			},
			{
				ID:          id2,
				Title:       "BANCO DE TALENTOS - Arquiteto/a - Projetos Executivos",
				City:        "Porto Alegre",
				State:       "RS",
				Company:     "PUCRS",
				Status:      job.StatusActive,
				CollectedAt: time.Now(),
			},
			{
				ID:          id3,
				Title:       "Jovem Aprendiz - Colégio São Carlos",
				City:        "Santa Vitória do Palmar",
				State:       "RS",
				Company:     "Hospital Mãe de Deus",
				Status:      job.StatusActive,
				CollectedAt: time.Now(),
			},
			{
				ID:          id4,
				Title:       "Enfermeiro Obstetra",
				City:        "Canoas",
				State:       "RS",
				Company:     "Hospital Moinhos",
				Status:      job.StatusActive,
				CollectedAt: time.Now(),
			},
		},
	}

	pruner := job.NewPruner(mockRepo, nil)
	res, err := pruner.Prune(context.Background())
	if err != nil {
		t.Fatalf("Prune() erro inesperado: %v", err)
	}

	if res.TotalChecked != 4 {
		t.Errorf("TotalChecked esperado 4, obteve %d", res.TotalChecked)
	}

	if res.TotalPruned != 2 {
		t.Errorf("TotalPruned esperado 2, obteve %d", res.TotalPruned)
	}

	if len(mockRepo.jobs) != 2 {
		t.Errorf("Vagas restantes no repositório esperadas 2, obteve %d", len(mockRepo.jobs))
	}

	// Verifica se as vagas restantes são estritamente as de enfermagem
	for _, remainingJob := range mockRepo.jobs {
		if remainingJob.ID == id2 || remainingJob.ID == id3 {
			t.Errorf("Vaga fora de domínio não foi removida: ID=%v Título=%s", remainingJob.ID, remainingJob.Title)
		}
	}
}

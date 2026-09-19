package job_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

func TestJob_Validate(t *testing.T) {
	now := time.Now()
	validJob := func() job.Job {
		return job.Job{
			ID:             uuid.New(),
			ExternalID:     "12345",
			Title:          "Técnico de Enfermagem - UTI Adulto",
			Company:        "Hospital Moinhos de Vento",
			Description:    "Atendimento assistencial na UTI",
			City:           "Porto Alegre",
			State:          "RS",
			Source:         "moinhos",
			SourceURL:      "https://carreiras.hospitalmoinhos.org.br/vaga/12345",
			Fingerprint:    job.Fingerprint("Hospital Moinhos de Vento", "Técnico de Enfermagem - UTI Adulto", "Porto Alegre"),
			WorkMode:       job.WorkModeOnSite,
			EmploymentType: job.EmploymentTypeFullTime,
			Status:         job.StatusActive,
			CollectedAt:    now,
			LastSeenAt:     now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}

	t.Run("vaga valida com campos obrigatorios preenchidos", func(t *testing.T) {
		j := validJob()
		if err := j.Validate(); err != nil {
			t.Fatalf("esperava vaga valida, obteve erro: %v", err)
		}
	})

	t.Run("erro quando titulo esta vazio", func(t *testing.T) {
		j := validJob()
		j.Title = "   "
		err := j.Validate()
		if err == nil || !strings.Contains(err.Error(), "título da vaga é obrigatório") {
			t.Fatalf("esperava erro de título obrigatório, obteve: %v", err)
		}
	})

	t.Run("erro quando empresa esta vazia", func(t *testing.T) {
		j := validJob()
		j.Company = ""
		err := j.Validate()
		if err == nil || !strings.Contains(err.Error(), "instituição/empresa da vaga é obrigatória") {
			t.Fatalf("esperava erro de empresa obrigatória, obteve: %v", err)
		}
	})

	t.Run("erro quando fonte esta vazia", func(t *testing.T) {
		j := validJob()
		j.Source = ""
		err := j.Validate()
		if err == nil || !strings.Contains(err.Error(), "fonte da vaga é obrigatória") {
			t.Fatalf("esperava erro de fonte obrigatória, obteve: %v", err)
		}
	})

	t.Run("erro quando source URL esta vazia", func(t *testing.T) {
		j := validJob()
		j.SourceURL = ""
		err := j.Validate()
		if err == nil || !strings.Contains(err.Error(), "URL de origem da vaga é obrigatória") {
			t.Fatalf("esperava erro de URL obrigatória, obteve: %v", err)
		}
	})

	t.Run("erro quando fingerprint esta vazio", func(t *testing.T) {
		j := validJob()
		j.Fingerprint = ""
		err := j.Validate()
		if err == nil || !strings.Contains(err.Error(), "fingerprint da vaga é obrigatório") {
			t.Fatalf("esperava erro de fingerprint obrigatório, obteve: %v", err)
		}
	})

	t.Run("erro quando salario minimo e maior que o maximo", func(t *testing.T) {
		j := validJob()
		min := int64(5000)
		max := int64(3000)
		j.SalaryMin = &min
		j.SalaryMax = &max

		err := j.Validate()
		if err == nil || !strings.Contains(err.Error(), "salário mínimo não pode ser maior") {
			t.Fatalf("esperava erro de salário mínimo maior que máximo, obteve: %v", err)
		}
	})

	t.Run("salarios validos quando min menor ou igual ao maximo", func(t *testing.T) {
		j := validJob()
		min := int64(3000)
		max := int64(5000)
		j.SalaryMin = &min
		j.SalaryMax = &max

		if err := j.Validate(); err != nil {
			t.Fatalf("esperava vaga valida com salarios coerentes, obteve: %v", err)
		}
	})
}

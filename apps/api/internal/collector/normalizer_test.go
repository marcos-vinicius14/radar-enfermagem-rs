package collector_test

import (
	"strings"
	"testing"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

func TestNormalizer_Normalize(t *testing.T) {
	normalizer := collector.NewNormalizer()

	t.Run("converte RawJob valido com normalizacao de espacos, estado e modalidade", func(t *testing.T) {
		pubDate := time.Now().Add(-1 * time.Hour).Truncate(time.Second)
		minSal := int64(3000)
		maxSal := int64(4500)

		raw := collector.RawJob{
			ExternalID:     " 12345 ",
			Title:          "  Técnico em   Enfermagem - UTI  Adulto  ",
			Company:        "  Hospital Santa   Casa  ",
			Description:    "  Descrição com múltiplos espaços   ",
			City:           "  Porto   Alegre  ",
			State:          " Rio Grande do Sul ",
			Source:         " santacasa ",
			SourceURL:      " https://santacasa.gupy.io/jobs/12345 ",
			WorkMode:       "on-site",
			EmploymentType: "clt",
			SalaryMin:      &minSal,
			SalaryMax:      &maxSal,
			PublishedAt:    &pubDate,
		}

		j, err := normalizer.Normalize(raw)
		if err != nil {
			t.Fatalf("Normalize() erro inesperado: %v", err)
		}

		if j.ExternalID != "12345" {
			t.Errorf("ExternalID = %q, esperado %q", j.ExternalID, "12345")
		}
		if j.Title != "Técnico em Enfermagem - UTI Adulto" {
			t.Errorf("Title = %q, esperado %q", j.Title, "Técnico em Enfermagem - UTI Adulto")
		}
		if j.Company != "Hospital Santa Casa" {
			t.Errorf("Company = %q, esperado %q", j.Company, "Hospital Santa Casa")
		}
		if j.City != "Porto Alegre" {
			t.Errorf("City = %q, esperado %q", j.City, "Porto Alegre")
		}
		if j.State != "RS" {
			t.Errorf("State = %q, esperado %q", j.State, "RS")
		}
		if j.Source != "santacasa" {
			t.Errorf("Source = %q, esperado %q", j.Source, "santacasa")
		}
		if j.SourceURL != "https://santacasa.gupy.io/jobs/12345" {
			t.Errorf("SourceURL = %q, esperado %q", j.SourceURL, "https://santacasa.gupy.io/jobs/12345")
		}
		if j.WorkMode != job.WorkModeOnSite {
			t.Errorf("WorkMode = %q, esperado %q", j.WorkMode, job.WorkModeOnSite)
		}
		if j.EmploymentType != job.EmploymentTypeFullTime {
			t.Errorf("EmploymentType = %q, esperado %q", j.EmploymentType, job.EmploymentTypeFullTime)
		}
		if j.Status != job.StatusActive {
			t.Errorf("Status = %q, esperado %q", j.Status, job.StatusActive)
		}
		if len(j.Fingerprint) != 64 {
			t.Errorf("Fingerprint length = %d, esperado 64", len(j.Fingerprint))
		}
		expectedFingerprint := job.Fingerprint("Hospital Santa Casa", "Técnico em Enfermagem - UTI Adulto", "Porto Alegre")
		if j.Fingerprint != expectedFingerprint {
			t.Errorf("Fingerprint = %q, esperado %q", j.Fingerprint, expectedFingerprint)
		}
		if j.CollectedAt.IsZero() {
			t.Errorf("CollectedAt não foi preenchido")
		}
		if j.LastSeenAt.IsZero() {
			t.Errorf("LastSeenAt não foi preenchido")
		}
	})

	t.Run("mapeamento de modalidades de trabalho e vinculos empregaticios", func(t *testing.T) {
		cases := []struct {
			rawMode      string
			rawType      string
			expectedMode job.WorkMode
			expectedType job.EmploymentType
		}{
			{"presencial", "efetivo", job.WorkModeOnSite, job.EmploymentTypeFullTime},
			{"on-site", "vacancy_type_effective", job.WorkModeOnSite, job.EmploymentTypeFullTime},
			{"remoto", "estagio", job.WorkModeRemote, job.EmploymentTypeInternship},
			{"remote", "estágio", job.WorkModeRemote, job.EmploymentTypeInternship},
			{"hibrido", "temporario", job.WorkModeHybrid, job.EmploymentTypeTemporary},
			{"hybrid", "temporary", job.WorkModeHybrid, job.EmploymentTypeTemporary},
			{"outro", "meio_periodo", "", job.EmploymentTypePartTime},
		}

		for _, tc := range cases {
			raw := collector.RawJob{
				ExternalID:     "100",
				Title:          "Enfermeiro",
				Company:        "Hospital Teste",
				City:           "Porto Alegre",
				State:          "RS",
				Source:         "teste",
				SourceURL:      "https://example.com/job/100",
				WorkMode:       tc.rawMode,
				EmploymentType: tc.rawType,
			}

			j, err := normalizer.Normalize(raw)
			if err != nil {
				t.Fatalf("Normalize() erro inesperado: %v", err)
			}
			if j.WorkMode != tc.expectedMode {
				t.Errorf("mode %q: obteve %q, esperado %q", tc.rawMode, j.WorkMode, tc.expectedMode)
			}
			if j.EmploymentType != tc.expectedType {
				t.Errorf("type %q: obteve %q, esperado %q", tc.rawType, j.EmploymentType, tc.expectedType)
			}
		}
	})

	t.Run("infere estado RS quando vazio e cidade for da regiao metropolitana de Porto Alegre", func(t *testing.T) {
		raw := collector.RawJob{
			ExternalID: "200",
			Title:      "Técnico de Enfermagem",
			Company:    "Hospital Metropolitano",
			City:       "Canoas",
			State:      "",
			Source:     "portal",
			SourceURL:  "https://example.com/200",
		}

		j, err := normalizer.Normalize(raw)
		if err != nil {
			t.Fatalf("Normalize() erro: %v", err)
		}
		if j.State != "RS" {
			t.Errorf("State = %q, esperado RS", j.State)
		}
	})

	t.Run("falha na validacao quando campos obrigatorios estao ausentes", func(t *testing.T) {
		base := collector.RawJob{
			ExternalID: "300",
			Title:      "Técnico de Enfermagem",
			Company:    "Hospital Geral",
			City:       "Porto Alegre",
			State:      "RS",
			Source:     "portal",
			SourceURL:  "https://example.com/300",
		}

		// Título vazio
		badTitle := base
		badTitle.Title = "   "
		_, err := normalizer.Normalize(badTitle)
		if err == nil || !strings.Contains(err.Error(), "título") {
			t.Errorf("esperava erro de título obrigatório, obteve: %v", err)
		}

		// Empresa vazia
		badCompany := base
		badCompany.Company = ""
		_, err = normalizer.Normalize(badCompany)
		if err == nil || !strings.Contains(err.Error(), "instituição/empresa") {
			t.Errorf("esperava erro de empresa obrigatória, obteve: %v", err)
		}

		// Fonte vazia
		badSource := base
		badSource.Source = ""
		_, err = normalizer.Normalize(badSource)
		if err == nil || !strings.Contains(err.Error(), "fonte") {
			t.Errorf("esperava erro de fonte obrigatória, obteve: %v", err)
		}

		// URL de origem vazia
		badURL := base
		badURL.SourceURL = ""
		_, err = normalizer.Normalize(badURL)
		if err == nil || !strings.Contains(err.Error(), "URL") {
			t.Errorf("esperava erro de URL obrigatória, obteve: %v", err)
		}

		// Salário mínimo maior que máximo
		minSal := int64(5000)
		maxSal := int64(3000)
		badSal := base
		badSal.SalaryMin = &minSal
		badSal.SalaryMax = &maxSal
		_, err = normalizer.Normalize(badSal)
		if err == nil || !strings.Contains(err.Error(), "salário") {
			t.Errorf("esperava erro de salário inválido, obteve: %v", err)
		}
	})
}

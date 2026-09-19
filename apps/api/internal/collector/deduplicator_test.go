package collector_test

import (
	"testing"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

func TestDeduplicator_DeduplicateBatch(t *testing.T) {
	deduplicator := collector.NewDeduplicator()

	t.Run("remove duplicatas exatas por source + external_id mantendo a primeira", func(t *testing.T) {
		jobs := []job.Job{
			{
				Source:      "santacasa",
				ExternalID:  "101",
				Title:       "Técnico em Enfermagem I",
				Company:     "Santa Casa",
				City:        "Porto Alegre",
				Fingerprint: "fp-1",
			},
			{
				Source:      "santacasa",
				ExternalID:  "101", // Duplicata exata
				Title:       "Técnico em Enfermagem I (Atualizado)",
				Company:     "Santa Casa",
				City:        "Porto Alegre",
				Fingerprint: "fp-1-updated",
			},
			{
				Source:      "santacasa",
				ExternalID:  "102",
				Title:       "Técnico em Enfermagem II",
				Company:     "Santa Casa",
				City:        "Porto Alegre",
				Fingerprint: "fp-2",
			},
		}

		deduped, discarded := deduplicator.DeduplicateBatch(jobs)
		if len(deduped) != 2 {
			t.Fatalf("esperava 2 vagas, obteve %d", len(deduped))
		}
		if discarded != 1 {
			t.Errorf("discarded = %d, esperado 1", discarded)
		}
		if deduped[0].ExternalID != "101" || deduped[0].Title != "Técnico em Enfermagem I" {
			t.Errorf("primeiro elemento preservado incorretamente: %+v", deduped[0])
		}
		if deduped[1].ExternalID != "102" {
			t.Errorf("segundo elemento incorreto: %+v", deduped[1])
		}
	})

	t.Run("remove duplicatas logicas por fingerprint dentro do lote", func(t *testing.T) {
		jobs := []job.Job{
			{
				Source:      "santacasa",
				ExternalID:  "201",
				Title:       "Técnico em Enfermagem",
				Company:     "Santa Casa",
				City:        "Porto Alegre",
				Fingerprint: "mesmo-fingerprint-123",
			},
			{
				Source:      "santacasa",
				ExternalID:  "202", // ID diferente, mas mesmo fingerprint
				Title:       "Técnico em Enfermagem",
				Company:     "Santa Casa",
				City:        "Porto Alegre",
				Fingerprint: "mesmo-fingerprint-123",
			},
		}

		deduped, discarded := deduplicator.DeduplicateBatch(jobs)
		if len(deduped) != 1 {
			t.Fatalf("esperava 1 vaga desduplicada, obteve %d", len(deduped))
		}
		if discarded != 1 {
			t.Errorf("discarded = %d, esperado 1", discarded)
		}
		if deduped[0].ExternalID != "201" {
			t.Errorf("deveria ter preservado o primeiro elemento (201), obteve: %s", deduped[0].ExternalID)
		}
	})

	t.Run("lote sem duplicatas preserva todos os elementos intactos", func(t *testing.T) {
		jobs := []job.Job{
			{
				Source:      "santacasa",
				ExternalID:  "301",
				Fingerprint: "fp-301",
			},
			{
				Source:      "santacasa",
				ExternalID:  "302",
				Fingerprint: "fp-302",
			},
			{
				Source:      "moinhos",
				ExternalID:  "303",
				Fingerprint: "fp-303",
			},
		}

		deduped, discarded := deduplicator.DeduplicateBatch(jobs)
		if len(deduped) != 3 {
			t.Errorf("esperava 3 vagas, obteve %d", len(deduped))
		}
		if discarded != 0 {
			t.Errorf("discarded = %d, esperado 0", discarded)
		}
	})

	t.Run("lote vazio retorna slice vazio sem panics", func(t *testing.T) {
		deduped, discarded := deduplicator.DeduplicateBatch(nil)
		if len(deduped) != 0 || discarded != 0 {
			t.Errorf("esperava slice vazio e 0 descartes, obteve len=%d discarded=%d", len(deduped), discarded)
		}
	})
}

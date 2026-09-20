//go:build e2e

package collector_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
)

func TestHCPACollector_LiveE2E(t *testing.T) {
	c := collector.NewHCPACollector(nil, 15*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	jobs, err := c.Collect(ctx, collector.SearchQuery{})
	if err != nil {
		t.Fatalf("falha ao coletar dados reais do HCPA (portal ou layout alterado): %v", err)
	}

	if len(jobs) == 0 {
		t.Fatalf("nenhum edital/processo seletivo foi extraído do portal real do HCPA (possível mudança de estrutura)")
	}

	first := jobs[0]
	if first.ExternalID == "" {
		t.Errorf("ExternalID vazio no primeiro edital extraído")
	}
	if first.Title == "" {
		t.Errorf("Title vazio no primeiro edital extraído")
	}
	if first.SourceURL == "" {
		t.Errorf("SourceURL vazio no primeiro edital extraído")
	}

	t.Logf("Sucesso E2E Live: %d editais/processos seletivos extraídos do HCPA. Primeiro: [%s] %s (%s)",
		len(jobs), first.ExternalID, first.Title, first.City)
}

func TestHCPACollector_LiveE2E_Enfermagem(t *testing.T) {
	c := collector.NewHCPACollector(nil, 15*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	jobs, err := c.Collect(ctx, collector.SearchQuery{Query: "edital"})
	if err != nil {
		t.Fatalf("falha ao coletar: %v", err)
	}

	t.Logf("=== EDITAIS/PROCESSOS SELETIVOS DO HCPA (%d) ===", len(jobs))
	for i, j := range jobs {
		t.Logf("[%d] ID: %s | Título: %s | Local: %s/%s | URL: %s",
			i+1, j.ExternalID, j.Title, j.City, j.State, j.SourceURL)
	}

	if len(jobs) > 0 {
		rawJSON, _ := json.MarshalIndent(jobs[0], "", "  ")
		t.Logf("\nExemplo de RawJob extraído em formato JSON:\n%s", string(rawJSON))
	}
}

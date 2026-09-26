package senior_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector/sources/senior"
)

func TestMaeDeDeusCollector_Collect(t *testing.T) {
	fixtureContent, err := os.ReadFile("testdata/senior_fixture.json")
	if err != nil {
		t.Fatalf("falha ao carregar fixture de teste: %v", err)
	}

	t.Run("extrai com sucesso vagas da fixture JSON do Mãe de Deus", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := senior.NewMaeDeDeusCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		if c.Name() != "maededeus" {
			t.Errorf("Name() = %q, esperado maededeus", c.Name())
		}

		ctx := context.Background()
		jobs, err := c.Collect(ctx, collector.SearchQuery{})
		if err != nil {
			t.Fatalf("Collect() erro inesperado: %v", err)
		}

		if len(jobs) != 3 {
			t.Fatalf("esperava 3 vagas extraídas, obteve %d", len(jobs))
		}

		first := jobs[0]
		if first.ExternalID != "d8610d98-373e-4fea-b2a3-956dee488e83" {
			t.Errorf("ExternalID = %q, incorreto", first.ExternalID)
		}
		if first.Title != "Técnico de Enfermagem - Centro Cirúrgico - Hospital Mãe de Deus" {
			t.Errorf("Title = %q, incorreto", first.Title)
		}
		if first.Company != "Hospital Mãe de Deus" {
			t.Errorf("Company = %q, esperado Hospital Mãe de Deus", first.Company)
		}
		if first.City != "Porto Alegre" {
			t.Errorf("City = %q, esperado Porto Alegre", first.City)
		}
		if first.Source != "maededeus" {
			t.Errorf("Source = %q, esperado maededeus", first.Source)
		}
		expectedURL := "https://somosaesc.portaldetalentos.senior.com.br/vacancy/d8610d98-373e-4fea-b2a3-956dee488e83"
		if first.SourceURL != expectedURL {
			t.Errorf("SourceURL = %q, esperado %q", first.SourceURL, expectedURL)
		}
	})

	t.Run("filtra vagas pelo termo da busca", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := senior.NewMaeDeDeusCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		ctx := context.Background()
		jobs, err := c.Collect(ctx, collector.SearchQuery{Query: "técnico"})
		if err != nil {
			t.Fatalf("Collect() erro: %v", err)
		}

		if len(jobs) != 1 {
			t.Fatalf("esperava 1 vaga filtrada por 'técnico', obteve %d", len(jobs))
		}
		if jobs[0].ExternalID != "d8610d98-373e-4fea-b2a3-956dee488e83" {
			t.Errorf("vaga filtrada incorreta: %s", jobs[0].Title)
		}
	})
}

package vagascom_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector/sources/vagascom"
)

func TestFleuryCollector_Collect(t *testing.T) {
	fixtureContent, err := os.ReadFile("testdata/fleury_fixture.html")
	if err != nil {
		t.Fatalf("falha ao carregar fixture de teste: %v", err)
	}

	t.Run("extrai com sucesso vagas da fixture HTML do Fleury", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := vagascom.NewFleuryCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		if c.Name() != "fleury" {
			t.Errorf("Name() = %q, esperado fleury", c.Name())
		}

		ctx := context.Background()
		jobs, err := c.Collect(ctx, collector.SearchQuery{})
		if err != nil {
			t.Fatalf("Collect() erro inesperado: %v", err)
		}

		if len(jobs) != 2 {
			t.Fatalf("esperava 2 vagas extraídas, obteve %d", len(jobs))
		}

		first := jobs[0]
		if first.ExternalID != "2816306" {
			t.Errorf("ExternalID = %q, esperado 2816306", first.ExternalID)
		}
		if !strings.Contains(first.Title, "Técnico(a) de Enfermagem") {
			t.Errorf("Title = %q, incorreto", first.Title)
		}
		if first.Company != "Grupo Fleury / Weinmann" {
			t.Errorf("Company = %q, incorreto", first.Company)
		}
		if first.City != "Porto Alegre" {
			t.Errorf("City = %q, esperado Porto Alegre", first.City)
		}
		if first.Source != "fleury" {
			t.Errorf("Source = %q, esperado fleury", first.Source)
		}
		expectedURL := "https://trabalheconosco.vagas.com.br/grupo-fleury/oportunidade/tecnico-a-de-enfermagem-weinmann-porto-alegre/2816306"
		if first.SourceURL != expectedURL {
			t.Errorf("SourceURL = %q, esperado %q", first.SourceURL, expectedURL)
		}
	})

	t.Run("filtra vagas por query", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := vagascom.NewFleuryCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		ctx := context.Background()
		jobs, err := c.Collect(ctx, collector.SearchQuery{Query: "colhedor"})
		if err != nil {
			t.Fatalf("Collect() erro: %v", err)
		}

		if len(jobs) != 1 {
			t.Fatalf("esperava 1 vaga filtrada por 'colhedor', obteve %d", len(jobs))
		}
		if jobs[0].ExternalID != "2816307" {
			t.Errorf("vaga filtrada incorreta: %s", jobs[0].Title)
		}
	})
}

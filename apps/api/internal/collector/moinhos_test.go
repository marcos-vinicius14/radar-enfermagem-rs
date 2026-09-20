package collector_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
)

func TestMoinhosCollector_Collect(t *testing.T) {
	fixtureContent, err := os.ReadFile("testdata/moinhos_fixture.html")
	if err != nil {
		t.Fatalf("falha ao carregar fixture de teste: %v", err)
	}

	t.Run("extrai com sucesso vagas da fixture HTML do Moinhos de Vento", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := collector.NewMoinhosCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		if c.Name() != "moinhos" {
			t.Errorf("Name() = %q, esperado moinhos", c.Name())
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
		if first.ExternalID != "11973172" {
			t.Errorf("ExternalID = %q, esperado 11973172", first.ExternalID)
		}
		if first.Title != "Técnico de Enfermagem - CTI Adulto" {
			t.Errorf("Title = %q, incorreto", first.Title)
		}
		if first.Company != "Hospital Moinhos de Vento" {
			t.Errorf("Company = %q, esperado Hospital Moinhos de Vento", first.Company)
		}
		if first.City != "Porto Alegre" {
			t.Errorf("City = %q, esperado Porto Alegre", first.City)
		}
		if first.State != "RS" {
			t.Errorf("State = %q, esperado RS", first.State)
		}
		if first.Source != "moinhos" {
			t.Errorf("Source = %q, esperado moinhos", first.Source)
		}
		expectedURL := "https://hospitalmoinhos.gupy.io/jobs/11973172"
		if first.SourceURL != expectedURL {
			t.Errorf("SourceURL = %q, esperado %q", first.SourceURL, expectedURL)
		}
	})

	t.Run("filtra vagas pelo termo da busca", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := collector.NewMoinhosCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		ctx := context.Background()
		jobs, err := c.Collect(ctx, collector.SearchQuery{Query: "técnico"})
		if err != nil {
			t.Fatalf("Collect() erro: %v", err)
		}

		if len(jobs) != 1 {
			t.Fatalf("esperava 1 vaga filtrada por 'técnico', obteve %d", len(jobs))
		}
		if jobs[0].ExternalID != "11973172" {
			t.Errorf("vaga filtrada incorreta: %s", jobs[0].Title)
		}
	})

	t.Run("retorna erro descritivo quando layout não possui script de dados", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("<html><body>Sem dados</body></html>"))
		}))
		defer server.Close()

		c := collector.NewMoinhosCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		ctx := context.Background()
		_, err := c.Collect(ctx, collector.SearchQuery{})
		if err == nil {
			t.Fatalf("esperava erro por ausência de dados, obteve nil")
		}
		if !strings.Contains(err.Error(), "não foi possível localizar os dados de vagas") {
			t.Errorf("mensagem de erro inesperada: %v", err)
		}
	})
}

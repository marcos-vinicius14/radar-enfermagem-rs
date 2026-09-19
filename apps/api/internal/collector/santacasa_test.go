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

func TestSantaCasaCollector_Collect(t *testing.T) {
	fixtureContent, err := os.ReadFile("testdata/santacasa_fixture.html")
	if err != nil {
		t.Fatalf("falha ao carregar fixture de teste: %v", err)
	}

	t.Run("extrai com sucesso todas as vagas da fixture HTML", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := collector.NewSantaCasaCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		if c.Name() != "santacasa" {
			t.Errorf("Name() = %q, esperado santacasa", c.Name())
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
		if first.ExternalID != "11889779" {
			t.Errorf("ExternalID = %q, esperado 11889779", first.ExternalID)
		}
		if first.Title != "Técnico em Enfermagem - Centro Cirúrgico" {
			t.Errorf("Title = %q, esperado Técnico em Enfermagem - Centro Cirúrgico", first.Title)
		}
		if first.Company != "Santa Casa de Porto Alegre" {
			t.Errorf("Company = %q, esperado Santa Casa de Porto Alegre", first.Company)
		}
		if first.City != "Porto Alegre" {
			t.Errorf("City = %q, esperado Porto Alegre", first.City)
		}
		if first.State != "RS" {
			t.Errorf("State = %q, esperado RS", first.State)
		}
		if first.Source != "santacasa" {
			t.Errorf("Source = %q, esperado santacasa", first.Source)
		}
		if first.WorkMode != "on-site" {
			t.Errorf("WorkMode = %q, esperado on-site", first.WorkMode)
		}
		if first.EmploymentType != "vacancy_type_effective" {
			t.Errorf("EmploymentType = %q, esperado vacancy_type_effective", first.EmploymentType)
		}
		expectedURL := "https://santacasa.gupy.io/jobs/11889779"
		if first.SourceURL != expectedURL {
			t.Errorf("SourceURL = %q, esperado %q", first.SourceURL, expectedURL)
		}
	})

	t.Run("filtra vagas pelo termo da busca", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := collector.NewSantaCasaCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		ctx := context.Background()
		jobs, err := c.Collect(ctx, collector.SearchQuery{Query: "técnico"})
		if err != nil {
			t.Fatalf("Collect() erro: %v", err)
		}

		if len(jobs) != 1 {
			t.Fatalf("esperava apenas 1 vaga filtrada por 'técnico', obteve %d", len(jobs))
		}
		if jobs[0].ExternalID != "11889779" {
			t.Errorf("vaga filtrada incorreta: %s", jobs[0].Title)
		}
	})

	t.Run("retorna erro descritivo em pt-BR quando o layout nao possui a tag de dados", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("<html><body>Layout alterado sem script de dados</body></html>"))
		}))
		defer server.Close()

		c := collector.NewSantaCasaCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		ctx := context.Background()
		_, err := c.Collect(ctx, collector.SearchQuery{})
		if err == nil {
			t.Fatalf("esperava erro por ausência de dados, obteve nil")
		}
		if !strings.Contains(err.Error(), "não foi possível localizar os dados de vagas") {
			t.Errorf("mensagem de erro inesperada: %v", err)
		}
	})

	t.Run("retorna erro quando servidor remoto retorna status 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c := collector.NewSantaCasaCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		ctx := context.Background()
		_, err := c.Collect(ctx, collector.SearchQuery{})
		if err == nil {
			t.Fatalf("esperava erro para HTTP 500, obteve nil")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("esperava código 500 no erro, obteve: %v", err)
		}
	})
}

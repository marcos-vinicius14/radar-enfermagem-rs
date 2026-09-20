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

func TestHCPACollector_Collect(t *testing.T) {
	fixtureContent, err := os.ReadFile("testdata/hcpa_fixture.html")
	if err != nil {
		t.Fatalf("falha ao carregar fixture de teste: %v", err)
	}

	t.Run("extrai com sucesso editais da fixture HTML do HCPA", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := collector.NewHCPACollectorWithURL(server.URL, server.Client(), 5*time.Second)

		if c.Name() != "hcpa" {
			t.Errorf("Name() = %q, esperado hcpa", c.Name())
		}

		ctx := context.Background()
		jobs, err := c.Collect(ctx, collector.SearchQuery{})
		if err != nil {
			t.Fatalf("Collect() erro inesperado: %v", err)
		}

		if len(jobs) != 3 {
			t.Fatalf("esperava 3 editais extraídos, obteve %d", len(jobs))
		}

		first := jobs[0]
		if first.ExternalID == "" {
			t.Errorf("ExternalID não pode ser vazio")
		}
		if !strings.Contains(first.Title, "Edital nº 02/2026") {
			t.Errorf("Title = %q, incorreto", first.Title)
		}
		if first.Company != "Hospital de Clínicas de Porto Alegre" {
			t.Errorf("Company = %q, incorreto", first.Company)
		}
		if first.City != "Porto Alegre" {
			t.Errorf("City = %q, esperado Porto Alegre", first.City)
		}
		if first.Source != "hcpa" {
			t.Errorf("Source = %q, esperado hcpa", first.Source)
		}
	})

	t.Run("filtra editais por busca textual", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := collector.NewHCPACollectorWithURL(server.URL, server.Client(), 5*time.Second)

		ctx := context.Background()
		jobs, err := c.Collect(ctx, collector.SearchQuery{Query: "Superior"})
		if err != nil {
			t.Fatalf("Collect() erro: %v", err)
		}

		if len(jobs) != 1 {
			t.Fatalf("esperava 1 edital filtrado por 'Superior', obteve %d", len(jobs))
		}
		if !strings.Contains(jobs[0].Title, "Edital nº 01/2026") {
			t.Errorf("edital filtrado incorreto: %s", jobs[0].Title)
		}
	})

	t.Run("retorna erro quando servidor remoto retorna 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c := collector.NewHCPACollectorWithURL(server.URL, server.Client(), 5*time.Second)

		_, err := c.Collect(context.Background(), collector.SearchQuery{})
		if err == nil {
			t.Fatalf("esperava erro, obteve nil")
		}
		if !strings.Contains(err.Error(), "código HTTP 500") {
			t.Errorf("mensagem de erro inesperada: %v", err)
		}
	})
}

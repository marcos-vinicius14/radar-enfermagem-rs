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

func TestGupyCollector_Collect(t *testing.T) {
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

		c := collector.NewGupyCollector(collector.GupyConfig{
			Name:         "gupy_test",
			CompanyName:  "Empresa Teste Gupy",
			BaseURL:      server.URL,
			DefaultCity:  "Porto Alegre",
			DefaultState: "RS",
		}, server.Client(), 5*time.Second)

		if c.Name() != "gupy_test" {
			t.Errorf("Name() = %q, esperado gupy_test", c.Name())
		}
		if c.BaseURL() != server.URL {
			t.Errorf("BaseURL() = %q, esperado %q", c.BaseURL(), server.URL)
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
			t.Errorf("Title = %q, incorreto", first.Title)
		}
		if first.Source != "gupy_test" {
			t.Errorf("Source = %q, esperado gupy_test", first.Source)
		}
	})

	t.Run("filtra vagas por query e cidade", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := collector.NewGupyCollector(collector.GupyConfig{
			Name:         "gupy_test",
			CompanyName:  "Empresa Teste Gupy",
			BaseURL:      server.URL,
			DefaultCity:  "Porto Alegre",
			DefaultState: "RS",
		}, server.Client(), 5*time.Second)

		ctx := context.Background()
		jobs, err := c.Collect(ctx, collector.SearchQuery{Query: "técnico", City: "Porto Alegre"})
		if err != nil {
			t.Fatalf("Collect() erro: %v", err)
		}

		if len(jobs) != 1 {
			t.Fatalf("esperava 1 vaga filtrada, obteve %d", len(jobs))
		}
		if jobs[0].ExternalID != "11889779" {
			t.Errorf("vaga filtrada incorreta: %s", jobs[0].Title)
		}
	})

	t.Run("retorna erro em pt-BR quando script __NEXT_DATA__ esta ausente", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("<html><body>Sem dados</body></html>"))
		}))
		defer server.Close()

		c := collector.NewGupyCollector(collector.GupyConfig{
			Name:    "gupy_test",
			BaseURL: server.URL,
		}, server.Client(), 5*time.Second)

		_, err := c.Collect(context.Background(), collector.SearchQuery{})
		if err == nil {
			t.Fatalf("esperava erro, obteve nil")
		}
		if !strings.Contains(err.Error(), "não foi possível localizar os dados de vagas") {
			t.Errorf("mensagem de erro inesperada: %v", err)
		}
	})

	t.Run("retorna erro quando status HTTP for 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c := collector.NewGupyCollector(collector.GupyConfig{
			Name:    "gupy_test",
			BaseURL: server.URL,
		}, server.Client(), 5*time.Second)

		_, err := c.Collect(context.Background(), collector.SearchQuery{})
		if err == nil {
			t.Fatalf("esperava erro HTTP 500, obteve nil")
		}
		if !strings.Contains(err.Error(), "código HTTP 500") {
			t.Errorf("mensagem de erro inesperada: %v", err)
		}
	})
}

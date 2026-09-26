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

func TestVagasComCollector_Collect(t *testing.T) {
	fixtureContent, err := os.ReadFile("testdata/vagascom_fixture.html")
	if err != nil {
		t.Fatalf("falha ao carregar fixture de teste: %v", err)
	}

	t.Run("extrai com sucesso vagas da fixture HTML do Vagas.com", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := vagascom.NewVagasComCollector(vagascom.VagasComConfig{
			Name:          "fleury_test",
			CompanyName:   "Grupo Fleury / Weinmann",
			BaseURL:       server.URL,
			PublicBaseURL: "https://trabalheconosco.vagas.com.br",
			DefaultCity:   "Porto Alegre",
			DefaultState:  "RS",
		}, server.Client(), 5*time.Second)

		if c.Name() != "fleury_test" {
			t.Errorf("Name() = %q, esperado fleury_test", c.Name())
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
		if first.State != "RS" {
			t.Errorf("State = %q, esperado RS", first.State)
		}
		expectedURL := "https://trabalheconosco.vagas.com.br/grupo-fleury/oportunidade/tecnico-a-de-enfermagem-weinmann-porto-alegre/2816306"
		if first.SourceURL != expectedURL {
			t.Errorf("SourceURL = %q, esperado %q", first.SourceURL, expectedURL)
		}
	})

	t.Run("filtra vagas por query e cidade", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := vagascom.NewVagasComCollector(vagascom.VagasComConfig{
			Name:        "fleury_test",
			BaseURL:     server.URL,
			DefaultCity: "Porto Alegre",
		}, server.Client(), 5*time.Second)

		ctx := context.Background()
		jobs, err := c.Collect(ctx, collector.SearchQuery{Query: "enfermagem", City: "Porto Alegre"})
		if err != nil {
			t.Fatalf("Collect() erro: %v", err)
		}

		if len(jobs) != 1 {
			t.Fatalf("esperava 1 vaga filtrada, obteve %d", len(jobs))
		}
		if jobs[0].ExternalID != "2816306" {
			t.Errorf("vaga filtrada incorreta: %s", jobs[0].Title)
		}
	})

	t.Run("retorna erro em pt-BR quando layout for completamente inválido", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("<html><body>Layout irreconhecível sem termos</body></html>"))
		}))
		defer server.Close()

		c := vagascom.NewVagasComCollector(vagascom.VagasComConfig{
			Name:    "fleury_test",
			BaseURL: server.URL,
		}, server.Client(), 5*time.Second)

		_, err := c.Collect(context.Background(), collector.SearchQuery{})
		if err == nil {
			t.Fatalf("esperava erro, obteve nil")
		}
		if !strings.Contains(err.Error(), "não foi possível localizar as vagas") {
			t.Errorf("mensagem de erro inesperada: %v", err)
		}
	})

	t.Run("retorna erro quando status HTTP for 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c := vagascom.NewVagasComCollector(vagascom.VagasComConfig{
			Name:    "fleury_test",
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

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

func TestSeniorCollector_Collect(t *testing.T) {
	fixtureContent, err := os.ReadFile("testdata/senior_fixture.json")
	if err != nil {
		t.Fatalf("falha ao carregar fixture de teste: %v", err)
	}

	t.Run("extrai com sucesso vagas da resposta JSON da Senior", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := collector.NewSeniorCollector(collector.SeniorConfig{
			Name:          "maededeus_test",
			CompanyName:   "Hospital Mãe de Deus",
			CompanyID:     "5ddfb2e5-3a03-4dd9-8d35-ae8d5538185b",
			EndpointURL:   server.URL,
			PortalBaseURL: "https://somosaesc.portaldetalentos.senior.com.br",
			DefaultCity:   "Porto Alegre",
			DefaultState:  "RS",
		}, server.Client(), 5*time.Second)

		if c.Name() != "maededeus_test" {
			t.Errorf("Name() = %q, esperado maededeus_test", c.Name())
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
			t.Errorf("Company = %q, incorreto", first.Company)
		}
		if first.City != "Porto Alegre" {
			t.Errorf("City = %q, esperado Porto Alegre", first.City)
		}
		if first.State != "RS" {
			t.Errorf("State = %q, esperado RS", first.State)
		}
		if first.WorkMode != "presencial" {
			t.Errorf("WorkMode = %q, esperado presencial", first.WorkMode)
		}
		expectedURL := "https://somosaesc.portaldetalentos.senior.com.br/vacancy/d8610d98-373e-4fea-b2a3-956dee488e83"
		if first.SourceURL != expectedURL {
			t.Errorf("SourceURL = %q, esperado %q", first.SourceURL, expectedURL)
		}
		if first.PublishedAt == nil {
			t.Errorf("PublishedAt não deveria ser nil")
		}
	})

	t.Run("filtra vagas por query", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := collector.NewSeniorCollector(collector.SeniorConfig{
			Name:        "senior_test",
			EndpointURL: server.URL,
		}, server.Client(), 5*time.Second)

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

	t.Run("retorna erro em pt-BR quando resposta não for JSON válido", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("não é um json válido"))
		}))
		defer server.Close()

		c := collector.NewSeniorCollector(collector.SeniorConfig{
			Name:        "senior_test",
			EndpointURL: server.URL,
		}, server.Client(), 5*time.Second)

		_, err := c.Collect(context.Background(), collector.SearchQuery{})
		if err == nil {
			t.Fatalf("esperava erro, obteve nil")
		}
		if !strings.Contains(err.Error(), "deserializar dados estruturados") {
			t.Errorf("mensagem de erro inesperada: %v", err)
		}
	})

	t.Run("retorna erro quando status HTTP for 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c := collector.NewSeniorCollector(collector.SeniorConfig{
			Name:        "senior_test",
			EndpointURL: server.URL,
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

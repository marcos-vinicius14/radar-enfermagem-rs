package custom_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector/sources/custom"
)

func TestDivinaCollector_Collect(t *testing.T) {
	fixtureContent, err := os.ReadFile("testdata/divina_fixture.html")
	if err != nil {
		t.Fatalf("falha ao carregar fixture de teste: %v", err)
	}

	t.Run("extrai com sucesso vagas da fixture HTML da Divina Providência", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := custom.NewDivinaCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		if c.Name() != "divina" {
			t.Errorf("Name() = %q, esperado divina", c.Name())
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
		if first.ExternalID == "" {
			t.Errorf("ExternalID não pode ser vazio")
		}
		if !strings.Contains(first.Title, "Técn. de Enfermagem") {
			t.Errorf("Title = %q, incorreto", first.Title)
		}
		if first.Company != "Rede de Saúde Divina Providência" {
			t.Errorf("Company = %q, incorreto", first.Company)
		}
		if first.City != "Porto Alegre" {
			t.Errorf("City = %q, esperado Porto Alegre", first.City)
		}
		if first.State != "RS" {
			t.Errorf("State = %q, esperado RS", first.State)
		}
		if first.Source != "divina" {
			t.Errorf("Source = %q, esperado divina", first.Source)
		}
	})

	t.Run("filtra vagas por busca textual", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := custom.NewDivinaCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		ctx := context.Background()
		jobs, err := c.Collect(ctx, collector.SearchQuery{Query: "obstétrico"})
		if err != nil {
			t.Fatalf("Collect() erro: %v", err)
		}

		if len(jobs) != 1 {
			t.Fatalf("esperava 1 vaga filtrada por 'obstétrico', obteve %d", len(jobs))
		}
		if !strings.Contains(jobs[0].Title, "Obstétrico") {
			t.Errorf("vaga filtrada incorreta: %s", jobs[0].Title)
		}
	})

	t.Run("retorna erro quando servidor remoto retorna 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c := custom.NewDivinaCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		_, err := c.Collect(context.Background(), collector.SearchQuery{})
		if err == nil {
			t.Fatalf("esperava erro, obteve nil")
		}
		if !strings.Contains(err.Error(), "código HTTP 500") {
			t.Errorf("mensagem de erro inesperada: %v", err)
		}
	})
}

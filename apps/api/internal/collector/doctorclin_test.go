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

func TestDoctorClinCollector_Collect(t *testing.T) {
	fixtureContent, err := os.ReadFile("testdata/doctorclin_fixture.html")
	if err != nil {
		t.Fatalf("falha ao carregar fixture de teste: %v", err)
	}

	t.Run("extrai com sucesso vagas da fixture HTML da Doctor Clin", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := collector.NewDoctorClinCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		if c.Name() != "doctorclin" {
			t.Errorf("Name() = %q, esperado doctorclin", c.Name())
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
		if first.ExternalID != "12438350" {
			t.Errorf("ExternalID = %q, esperado 12438350", first.ExternalID)
		}
		if first.Title != "Técnico de Enfermagem - Atendimento Clínico" {
			t.Errorf("Title = %q, incorreto", first.Title)
		}
		if first.Company != "DC GROUP" && first.Company != "Doctor Clin" {
			t.Errorf("Company = %q, incorreto", first.Company)
		}
		if first.City != "Novo Hamburgo" {
			t.Errorf("City = %q, esperado Novo Hamburgo", first.City)
		}
		if first.Source != "doctorclin" {
			t.Errorf("Source = %q, esperado doctorclin", first.Source)
		}
		expectedURL := "https://dcgroup.gupy.io/jobs/12438350"
		if first.SourceURL != expectedURL {
			t.Errorf("SourceURL = %q, esperado %q", first.SourceURL, expectedURL)
		}
	})

	t.Run("filtra vagas pelo termo da busca", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(fixtureContent)
		}))
		defer server.Close()

		c := collector.NewDoctorClinCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		ctx := context.Background()
		jobs, err := c.Collect(ctx, collector.SearchQuery{Query: "auditoria"})
		if err != nil {
			t.Fatalf("Collect() erro: %v", err)
		}

		if len(jobs) != 1 {
			t.Fatalf("esperava 1 vaga filtrada, obteve %d", len(jobs))
		}
		if jobs[0].ExternalID != "12438351" {
			t.Errorf("vaga filtrada incorreta: %s", jobs[0].Title)
		}
	})

	t.Run("retorna erro quando servidor remoto retorna 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c := collector.NewDoctorClinCollectorWithURL(server.URL, server.Client(), 5*time.Second)

		_, err := c.Collect(context.Background(), collector.SearchQuery{})
		if err == nil {
			t.Fatalf("esperava erro, obteve nil")
		}
		if !strings.Contains(err.Error(), "código HTTP 500") {
			t.Errorf("mensagem de erro inesperada: %v", err)
		}
	})
}

package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	internalhttp "github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/http"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

type mockJobRepository struct {
	searchFn        func(ctx context.Context, params job.FilterParams) (job.PaginatedJobs, error)
	findByIDFn      func(ctx context.Context, id uuid.UUID) (job.Job, error)
	listCompaniesFn func(ctx context.Context, status string) ([]job.CompanyStat, error)
	listCitiesFn    func(ctx context.Context, status string) ([]job.CityStat, error)
	listSourcesFn   func(ctx context.Context, status string) ([]job.SourceStat, error)
}

func (m *mockJobRepository) Insert(ctx context.Context, j job.Job) (job.Job, error) {
	return j, nil
}

func (m *mockJobRepository) Update(ctx context.Context, j job.Job) (job.Job, error) {
	return j, nil
}

func (m *mockJobRepository) FindByID(ctx context.Context, id uuid.UUID) (job.Job, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return job.Job{}, job.ErrNotFound
}

func (m *mockJobRepository) FindBySourceAndExternalID(ctx context.Context, source, externalID string) (job.Job, error) {
	return job.Job{}, job.ErrNotFound
}

func (m *mockJobRepository) FindByFingerprint(ctx context.Context, fingerprint string) (job.Job, error) {
	return job.Job{}, job.ErrNotFound
}

func (m *mockJobRepository) List(ctx context.Context, params job.ListParams) ([]job.Job, error) {
	return nil, nil
}

func (m *mockJobRepository) Search(ctx context.Context, params job.FilterParams) (job.PaginatedJobs, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, params)
	}
	return job.PaginatedJobs{}, nil
}

func (m *mockJobRepository) ListCompanies(ctx context.Context, status string) ([]job.CompanyStat, error) {
	if m.listCompaniesFn != nil {
		return m.listCompaniesFn(ctx, status)
	}
	return nil, nil
}

func (m *mockJobRepository) ListCities(ctx context.Context, status string) ([]job.CityStat, error) {
	if m.listCitiesFn != nil {
		return m.listCitiesFn(ctx, status)
	}
	return nil, nil
}

func (m *mockJobRepository) ListSources(ctx context.Context, status string) ([]job.SourceStat, error) {
	if m.listSourcesFn != nil {
		return m.listSourcesFn(ctx, status)
	}
	return nil, nil
}

func (m *mockJobRepository) UpdateLastSeen(ctx context.Context, id uuid.UUID, lastSeenAt time.Time) error {
	return nil
}

func (m *mockJobRepository) ReconcileStatuses(ctx context.Context, unknownBefore, expiredBefore time.Time) (job.StatusReconciliationResult, error) {
	return job.StatusReconciliationResult{}, nil
}

func (m *mockJobRepository) DeleteByIDs(ctx context.Context, ids []uuid.UUID) (int64, error) {
	return int64(len(ids)), nil
}

func (m *mockJobRepository) ListActiveForPruning(ctx context.Context, limit, offset int32) ([]job.Job, error) {
	return nil, nil
}

func sampleJob() job.Job {
	id := uuid.MustParse("01923b7e-8c34-7123-9000-123456789abc")
	now := time.Now().UTC()
	return job.Job{
		ID:             id,
		ExternalID:     "vaga-101",
		Title:          "Técnico de Enfermagem - CTI",
		Company:        "Hospital Moinhos de Vento",
		Description:    "Cuidados intensivos",
		City:           "Porto Alegre",
		State:          "RS",
		Source:         "moinhos",
		SourceURL:      "https://moinhos.com/jobs/101",
		WorkMode:       job.WorkModeOnSite,
		EmploymentType: job.EmploymentTypeFullTime,
		PublishedAt:    &now,
		Status:         job.StatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func TestListJobsEndpoint(t *testing.T) {
	t.Run("retorna lista paginada com sucesso e status 200", func(t *testing.T) {
		repo := &mockJobRepository{
			searchFn: func(ctx context.Context, params job.FilterParams) (job.PaginatedJobs, error) {
				if params.Page != 1 || params.Size != 20 {
					t.Errorf("parâmetros de paginação default incorretos: page=%d, size=%d", params.Page, params.Size)
				}
				if params.Status != "ACTIVE" {
					t.Errorf("status padrão esperado 'ACTIVE', obteve: %s", params.Status)
				}
				return job.PaginatedJobs{
					Items:      []job.Job{sampleJob()},
					Page:       1,
					Size:       20,
					Total:      1,
					TotalPages: 1,
				}, nil
			},
		}

		router := internalhttp.NewRouter(testLogger(), &mockDB{}, repo)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("esperado status 200, obteve %d", rec.Code)
		}

		var res job.PaginatedJobs
		if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
			t.Fatalf("falha ao decodificar resposta JSON: %v", err)
		}

		if res.Total != 1 || len(res.Items) != 1 {
			t.Errorf("resposta inesperada: %+v", res)
		}
		if res.Items[0].Title != "Técnico de Enfermagem - CTI" {
			t.Errorf("título da vaga incorreto: %s", res.Items[0].Title)
		}
	})

	t.Run("repassa filtros customizados query, city, company, status e date", func(t *testing.T) {
		var receivedParams job.FilterParams
		repo := &mockJobRepository{
			searchFn: func(ctx context.Context, params job.FilterParams) (job.PaginatedJobs, error) {
				receivedParams = params
				return job.PaginatedJobs{
					Items:      []job.Job{},
					Page:       params.Page,
					Size:       params.Size,
					Total:      0,
					TotalPages: 0,
				}, nil
			},
		}

		router := internalhttp.NewRouter(testLogger(), &mockDB{}, repo)
		url := "/api/v1/jobs?query=enfermagem&city=Canoas&state=RS&company=Unimed&source=unimed&status=all&date=7d&page=2&size=50"
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("esperado status 200, obteve %d", rec.Code)
		}

		if receivedParams.Query != "enfermagem" {
			t.Errorf("Query = %s, esperado 'enfermagem'", receivedParams.Query)
		}
		if receivedParams.City != "Canoas" {
			t.Errorf("City = %s, esperado 'Canoas'", receivedParams.City)
		}
		if receivedParams.State != "RS" {
			t.Errorf("State = %s, esperado 'RS'", receivedParams.State)
		}
		if receivedParams.Company != "Unimed" {
			t.Errorf("Company = %s, esperado 'Unimed'", receivedParams.Company)
		}
		if receivedParams.Source != "unimed" {
			t.Errorf("Source = %s, esperado 'unimed'", receivedParams.Source)
		}
		if receivedParams.Status != "" {
			t.Errorf("Status com 'all' deveria ser vazio, obteve '%s'", receivedParams.Status)
		}
		if receivedParams.PublishedSince == nil {
			t.Errorf("PublishedSince não deveria ser nulo para date=7d")
		}
		if receivedParams.Page != 2 || receivedParams.Size != 50 {
			t.Errorf("paginação incorreta: page=%d, size=%d", receivedParams.Page, receivedParams.Size)
		}
	})

	t.Run("retorna 400 Bad Request quando page não é numérico", func(t *testing.T) {
		router := internalhttp.NewRouter(testLogger(), &mockDB{}, &mockJobRepository{})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?page=abc", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("esperado status 400, obteve %d", rec.Code)
		}

		var body map[string]string
		_ = json.NewDecoder(rec.Body).Decode(&body)
		if body["error"] != "parâmetro de página inválido" {
			t.Errorf("mensagem de erro inesperada: %s", body["error"])
		}
	})

	t.Run("retorna 400 Bad Request quando size não é numérico", func(t *testing.T) {
		router := internalhttp.NewRouter(testLogger(), &mockDB{}, &mockJobRepository{})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?size=xyz", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("esperado status 400, obteve %d", rec.Code)
		}

		var body map[string]string
		_ = json.NewDecoder(rec.Body).Decode(&body)
		if body["error"] != "parâmetro de tamanho de página inválido" {
			t.Errorf("mensagem de erro inesperada: %s", body["error"])
		}
	})

	t.Run("retorna 400 Bad Request quando date possui formato desconhecido", func(t *testing.T) {
		router := internalhttp.NewRouter(testLogger(), &mockDB{}, &mockJobRepository{})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?date=invalido", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("esperado status 400, obteve %d", rec.Code)
		}

		var body map[string]string
		_ = json.NewDecoder(rec.Body).Decode(&body)
		if body["error"] != "formato de data inválido, use 'today', '3d', '7d', '30d' ou 'AAAA-MM-DD'" {
			t.Errorf("mensagem de erro inesperada: %s", body["error"])
		}
	})

	t.Run("retorna 503 Service Unavailable quando o repositório é nulo", func(t *testing.T) {
		router := internalhttp.NewRouter(testLogger(), &mockDB{}, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("esperado status 503, obteve %d", rec.Code)
		}
	})
}

func TestGetJobByIDEndpoint(t *testing.T) {
	validID := uuid.MustParse("01923b7e-8c34-7123-9000-123456789abc")

	t.Run("retorna detalhes da vaga com status 200 quando existe", func(t *testing.T) {
		repo := &mockJobRepository{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (job.Job, error) {
				if id == validID {
					return sampleJob(), nil
				}
				return job.Job{}, job.ErrNotFound
			},
		}

		router := internalhttp.NewRouter(testLogger(), &mockDB{}, repo)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+validID.String(), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("esperado status 200, obteve %d", rec.Code)
		}

		var j job.Job
		if err := json.NewDecoder(rec.Body).Decode(&j); err != nil {
			t.Fatalf("falha ao decodificar JSON: %v", err)
		}
		if j.ID != validID || j.Title != "Técnico de Enfermagem - CTI" {
			t.Errorf("vaga retornada incorreta: %+v", j)
		}
	})

	t.Run("retorna 404 Not Found quando a vaga não existe", func(t *testing.T) {
		repo := &mockJobRepository{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (job.Job, error) {
				return job.Job{}, job.ErrNotFound
			},
		}

		router := internalhttp.NewRouter(testLogger(), &mockDB{}, repo)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+uuid.New().String(), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("esperado status 404, obteve %d", rec.Code)
		}

		var body map[string]string
		_ = json.NewDecoder(rec.Body).Decode(&body)
		if body["error"] != "vaga não encontrada" {
			t.Errorf("mensagem de erro inesperada: %s", body["error"])
		}
	})

	t.Run("retorna 400 Bad Request quando ID é inválido", func(t *testing.T) {
		router := internalhttp.NewRouter(testLogger(), &mockDB{}, &mockJobRepository{})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/id-invalido-123", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("esperado status 400, obteve %d", rec.Code)
		}

		var body map[string]string
		_ = json.NewDecoder(rec.Body).Decode(&body)
		if body["error"] != "identificador de vaga inválido" {
			t.Errorf("mensagem de erro inesperada: %s", body["error"])
		}
	})

	t.Run("retorna 500 quando repositório falha inesperadamente", func(t *testing.T) {
		repo := &mockJobRepository{
			findByIDFn: func(ctx context.Context, id uuid.UUID) (job.Job, error) {
				return job.Job{}, errors.New("banco indisponível")
			},
		}

		router := internalhttp.NewRouter(testLogger(), &mockDB{}, repo)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+validID.String(), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("esperado status 500, obteve %d", rec.Code)
		}
	})
}

func TestMetadataEndpoints(t *testing.T) {
	t.Run("GET /api/v1/companies retorna empresas com contagem", func(t *testing.T) {
		repo := &mockJobRepository{
			listCompaniesFn: func(ctx context.Context, status string) ([]job.CompanyStat, error) {
				if status != "ACTIVE" {
					t.Errorf("status esperado 'ACTIVE', obteve '%s'", status)
				}
				return []job.CompanyStat{
					{Name: "Hospital Moinhos de Vento", TotalJobs: 10},
					{Name: "Hospital Santa Casa", TotalJobs: 25},
				}, nil
			},
		}

		router := internalhttp.NewRouter(testLogger(), &mockDB{}, repo)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/companies", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("esperado status 200, obteve %d", rec.Code)
		}

		var companies []job.CompanyStat
		if err := json.NewDecoder(rec.Body).Decode(&companies); err != nil {
			t.Fatalf("falha ao decodificar JSON: %v", err)
		}

		if len(companies) != 2 || companies[0].TotalJobs != 10 {
			t.Errorf("resposta inesperada: %+v", companies)
		}
	})

	t.Run("GET /api/v1/cities retorna cidades com contagem", func(t *testing.T) {
		repo := &mockJobRepository{
			listCitiesFn: func(ctx context.Context, status string) ([]job.CityStat, error) {
				return []job.CityStat{
					{City: "Canoas", State: "RS", TotalJobs: 5},
					{City: "Porto Alegre", State: "RS", TotalJobs: 40},
				}, nil
			},
		}

		router := internalhttp.NewRouter(testLogger(), &mockDB{}, repo)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cities", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("esperado status 200, obteve %d", rec.Code)
		}

		var cities []job.CityStat
		if err := json.NewDecoder(rec.Body).Decode(&cities); err != nil {
			t.Fatalf("falha ao decodificar JSON: %v", err)
		}

		if len(cities) != 2 || cities[1].City != "Porto Alegre" {
			t.Errorf("resposta inesperada: %+v", cities)
		}
	})

	t.Run("GET /api/v1/sources retorna fontes com contagem", func(t *testing.T) {
		repo := &mockJobRepository{
			listSourcesFn: func(ctx context.Context, status string) ([]job.SourceStat, error) {
				return []job.SourceStat{
					{Source: "hcpa", TotalJobs: 8},
					{Source: "santacasa", TotalJobs: 15},
				}, nil
			},
		}

		router := internalhttp.NewRouter(testLogger(), &mockDB{}, repo)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sources", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("esperado status 200, obteve %d", rec.Code)
		}

		var sources []job.SourceStat
		if err := json.NewDecoder(rec.Body).Decode(&sources); err != nil {
			t.Fatalf("falha ao decodificar JSON: %v", err)
		}

		if len(sources) != 2 || sources[0].Source != "hcpa" {
			t.Errorf("resposta inesperada: %+v", sources)
		}
	})
}

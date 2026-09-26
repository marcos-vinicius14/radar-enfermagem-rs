package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	internalhttp "github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/http"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

type mockWebJobRepo struct {
	searchResult  job.PaginatedJobs
	searchErr     error
	citiesResult  []job.CityStat
	citiesErr     error
	compResult    []job.CompanyStat
	compErr       error
	capturedParam job.FilterParams
}

func (m *mockWebJobRepo) Search(ctx context.Context, params job.FilterParams) (job.PaginatedJobs, error) {
	m.capturedParam = params
	if m.searchErr != nil {
		return job.PaginatedJobs{}, m.searchErr
	}
	return m.searchResult, nil
}

func (m *mockWebJobRepo) ListCities(ctx context.Context, status string) ([]job.CityStat, error) {
	if m.citiesErr != nil {
		return nil, m.citiesErr
	}
	return m.citiesResult, nil
}

func (m *mockWebJobRepo) ListCompanies(ctx context.Context, status string) ([]job.CompanyStat, error) {
	if m.compErr != nil {
		return nil, m.compErr
	}
	return m.compResult, nil
}

// Stubs não utilizados pelo WebHandler
func (m *mockWebJobRepo) Insert(ctx context.Context, j job.Job) (job.Job, error) {
	return j, nil
}
func (m *mockWebJobRepo) Update(ctx context.Context, j job.Job) (job.Job, error) {
	return j, nil
}
func (m *mockWebJobRepo) FindByID(ctx context.Context, id uuid.UUID) (job.Job, error) {
	return job.Job{}, nil
}
func (m *mockWebJobRepo) FindBySourceAndExternalID(ctx context.Context, s, e string) (job.Job, error) {
	return job.Job{}, nil
}
func (m *mockWebJobRepo) FindByFingerprint(ctx context.Context, f string) (job.Job, error) {
	return job.Job{}, nil
}
func (m *mockWebJobRepo) List(ctx context.Context, params job.ListParams) ([]job.Job, error) {
	return nil, nil
}
func (m *mockWebJobRepo) ListSources(ctx context.Context, status string) ([]job.SourceStat, error) {
	return nil, nil
}
func (m *mockWebJobRepo) UpdateLastSeen(ctx context.Context, id uuid.UUID, t time.Time) error {
	return nil
}
func (m *mockWebJobRepo) ReconcileStatuses(ctx context.Context, u, e time.Time) (job.StatusReconciliationResult, error) {
	return job.StatusReconciliationResult{}, nil
}
func (m *mockWebJobRepo) DeleteByIDs(ctx context.Context, ids []uuid.UUID) (int64, error) {
	return int64(len(ids)), nil
}
func (m *mockWebJobRepo) ListActiveForPruning(ctx context.Context, limit, offset int32) ([]job.Job, error) {
	return nil, nil
}

func createSampleJob() job.Job {
	pub := time.Now().Add(-3 * time.Hour)
	salMin := int64(3200)
	salMax := int64(4200)

	return job.Job{
		ID:             uuid.New(),
		Title:          "Técnico de Enfermagem - CTI Adulto",
		Company:        "Hospital Moinhos de Vento",
		City:           "Porto Alegre",
		State:          "RS",
		Source:         "moinhos",
		SourceURL:      "https://trabalheconosco.vagas.com.br/moinhos/vaga/123",
		WorkMode:       job.WorkModeOnSite,
		EmploymentType: job.EmploymentTypeFullTime,
		SalaryMin:      &salMin,
		SalaryMax:      &salMax,
		PublishedAt:    &pub,
		Description:    "Assistência em terapia intensiva adulta.",
		Status:         job.StatusActive,
		UpdatedAt:      time.Now(),
	}
}

func TestWebHandler_Home_Success(t *testing.T) {
	sampleJob := createSampleJob()
	mockRepo := &mockWebJobRepo{
		searchResult: job.PaginatedJobs{
			Items:      []job.Job{sampleJob},
			Page:       1,
			Size:       20,
			Total:      1,
			TotalPages: 1,
		},
		citiesResult: []job.CityStat{
			{City: "Porto Alegre", State: "RS", TotalJobs: 1},
		},
		compResult: []job.CompanyStat{
			{Name: "Hospital Moinhos de Vento", TotalJobs: 1},
		},
	}

	router := internalhttp.NewRouter(testLogger(), &mockDB{}, mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava status 200 para GET /, obteve: %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("esperava Content-Type text/html, obteve: %s", contentType)
	}

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Error("esperava cabeçalho ETag presente na resposta")
	}

	body := rec.Body.String()
	expectedContents := []string{
		"<!DOCTYPE html>",
		"Radar Enfermagem RS",
		"Hospital Moinhos de Vento",
		"Técnico de Enfermagem - CTI Adulto",
		"Porto Alegre / RS",
		"UTI / CTI",
		"Candidatar-se no Portal Oficial",
		"id=\"search-form\"",
		"hx-sync=\"this:replace\"",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(body, expected) {
			t.Errorf("resposta HTML não contém elemento esperado: %q", expected)
		}
	}
}

func TestWebHandler_Jobs_HTMX_Fragment(t *testing.T) {
	sampleJob := createSampleJob()
	mockRepo := &mockWebJobRepo{
		searchResult: job.PaginatedJobs{
			Items:      []job.Job{sampleJob},
			Page:       1,
			Size:       20,
			Total:      1,
			TotalPages: 1,
		},
	}

	router := internalhttp.NewRouter(testLogger(), &mockDB{}, mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava status 200 para fragmento HTMX, obteve: %d", rec.Code)
	}

	body := rec.Body.String()

	// O fragmento NÃO deve conter o layout completo
	if strings.Contains(body, "<!DOCTYPE html>") || strings.Contains(body, "<html") {
		t.Error("fragmento HTMX não deve conter tags de documento completo (<!DOCTYPE ou <html)")
	}

	// O fragmento deve conter o alvo de substituição e os cards
	if !strings.Contains(body, "id=\"jobs-container\"") {
		t.Error("fragmento HTMX deve conter id=\"jobs-container\"")
	}
	if !strings.Contains(body, "Técnico de Enfermagem - CTI Adulto") {
		t.Error("fragmento HTMX deve conter o card renderizado")
	}
}

func TestWebHandler_Jobs_Filters_CityAndQuery(t *testing.T) {
	mockRepo := &mockWebJobRepo{
		searchResult: job.PaginatedJobs{
			Items:      []job.Job{},
			Page:       1,
			Size:       20,
			Total:      0,
			TotalPages: 0,
		},
	}

	router := internalhttp.NewRouter(testLogger(), &mockDB{}, mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/jobs?query=UTI&city=Porto+Alegre&company=Santa+Casa&date=today", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava status 200, obteve: %d", rec.Code)
	}

	// Valida parâmetros capturados pelo repositório
	if mockRepo.capturedParam.Query != "UTI" {
		t.Errorf("esperava Query='UTI', obteve: %q", mockRepo.capturedParam.Query)
	}
	if mockRepo.capturedParam.City != "Porto Alegre" {
		t.Errorf("esperava City='Porto Alegre', obteve: %q", mockRepo.capturedParam.City)
	}
	if mockRepo.capturedParam.Company != "Santa Casa" {
		t.Errorf("esperava Company='Santa Casa', obteve: %q", mockRepo.capturedParam.Company)
	}
	if mockRepo.capturedParam.PublishedSince == nil {
		t.Error("esperava PublishedSince preenchido para filtro date='today'")
	}
}

func TestWebHandler_Jobs_EmptyState(t *testing.T) {
	mockRepo := &mockWebJobRepo{
		searchResult: job.PaginatedJobs{
			Items:      []job.Job{},
			Page:       1,
			Size:       20,
			Total:      0,
			TotalPages: 0,
		},
	}

	router := internalhttp.NewRouter(testLogger(), &mockDB{}, mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/jobs?query=inexistente_123", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava status 200, obteve: %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Nenhuma vaga encontrada") {
		t.Errorf("esperava estado vazio amigável em pt-BR, obteve: %s", body)
	}
	if !strings.Contains(body, "Limpar todos os filtros") {
		t.Errorf("esperava botão de limpar filtros, obteve: %s", body)
	}
}

func TestWebHandler_Jobs_Pagination(t *testing.T) {
	sampleJob := createSampleJob()
	mockRepo := &mockWebJobRepo{
		searchResult: job.PaginatedJobs{
			Items:      []job.Job{sampleJob},
			Page:       2,
			Size:       10,
			Total:      50,
			TotalPages: 5,
		},
	}

	router := internalhttp.NewRouter(testLogger(), &mockDB{}, mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/jobs?page=2&size=10", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava status 200, obteve: %d", rec.Code)
	}

	if mockRepo.capturedParam.Page != 2 || mockRepo.capturedParam.Size != 10 {
		t.Errorf("esperava page=2 e size=10, obteve page=%d e size=%d", mockRepo.capturedParam.Page, mockRepo.capturedParam.Size)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Página <strong>2</strong> de <strong>5</strong>") {
		t.Errorf("esperava informações de paginação 'Página 2 de 5', obteve: %s", body)
	}
}

func TestWebHandler_ETag_304NotModified(t *testing.T) {
	sampleJob := createSampleJob()
	mockRepo := &mockWebJobRepo{
		searchResult: job.PaginatedJobs{
			Items:      []job.Job{sampleJob},
			Page:       1,
			Size:       20,
			Total:      1,
			TotalPages: 1,
		},
	}

	router := internalhttp.NewRouter(testLogger(), &mockDB{}, mockRepo)

	// 1ª requisição obtém o ETag
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	etag := rec1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("esperava ETag presente na primeira requisição")
	}

	// 2ª requisição consecutiva simulando F5 com o mesmo ETag
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("If-None-Match", etag)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusNotModified {
		t.Fatalf("esperava HTTP 304 Not Modified no F5 com ETag idêntico, obteve: %d", rec2.Code)
	}

	if rec2.Body.Len() > 0 {
		t.Errorf("esperava corpo vazio para resposta 304, obteve %d bytes", rec2.Body.Len())
	}
}

func TestWebHandler_StaticFiles(t *testing.T) {
	router := internalhttp.NewRouter(testLogger(), &mockDB{}, nil)

	// 1. CSS
	reqCSS := httptest.NewRequest(http.MethodGet, "/static/css/styles.css", nil)
	recCSS := httptest.NewRecorder()
	router.ServeHTTP(recCSS, reqCSS)

	if recCSS.Code != http.StatusOK {
		t.Fatalf("esperava 200 para /static/css/styles.css, obteve: %d", recCSS.Code)
	}
	if cache := recCSS.Header().Get("Cache-Control"); !strings.Contains(cache, "immutable") {
		t.Errorf("esperava Cache-Control com immutable para assets estáticos, obteve: %s", cache)
	}

	// 2. JS HTMX
	reqJS := httptest.NewRequest(http.MethodGet, "/static/js/htmx.min.js", nil)
	recJS := httptest.NewRecorder()
	router.ServeHTTP(recJS, reqJS)

	if recJS.Code != http.StatusOK {
		t.Fatalf("esperava 200 para /static/js/htmx.min.js, obteve: %d", recJS.Code)
	}

	// 3. Favicon SVG
	reqSVG := httptest.NewRequest(http.MethodGet, "/static/img/favicon.svg", nil)
	recSVG := httptest.NewRecorder()
	router.ServeHTTP(recSVG, reqSVG)

	if recSVG.Code != http.StatusOK {
		t.Fatalf("esperava 200 para /static/img/favicon.svg, obteve: %d", recSVG.Code)
	}
}

func TestWebHandler_Home_SpecialtyChipActive_SSR(t *testing.T) {
	mockRepo := &mockWebJobRepo{
		searchResult: job.PaginatedJobs{
			Items:      []job.Job{},
			Page:       1,
			Size:       20,
			Total:      0,
			TotalPages: 0,
		},
	}

	router := internalhttp.NewRouter(testLogger(), &mockDB{}, mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/?query=UTI", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava status 200 para GET /?query=UTI, obteve: %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "class=\"chip-btn active\"\n                data-query=\"UTI\"") {
		t.Error("esperava que o chip UTI tivesse a classe 'active' ao renderizar no SSR com query=UTI")
	}
	if strings.Contains(body, "class=\"chip-btn active\"\n                data-query=\"Cirurgico\"") {
		t.Error("não esperava que o chip Cirurgico tivesse a classe 'active'")
	}
}

func TestWebHandler_Home_SpecialtyChipsMultipleActive_SSR(t *testing.T) {
	mockRepo := &mockWebJobRepo{
		searchResult: job.PaginatedJobs{
			Items:      []job.Job{},
			Page:       1,
			Size:       20,
			Total:      0,
			TotalPages: 0,
		},
	}

	router := internalhttp.NewRouter(testLogger(), &mockDB{}, mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/?query=UTI,Pediatria", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava status 200 para GET /?query=UTI,Pediatria, obteve: %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "class=\"chip-btn active\"\n                data-query=\"UTI\"") {
		t.Error("esperava que o chip UTI tivesse a classe 'active' ao renderizar no SSR com query=UTI,Pediatria")
	}
	if !strings.Contains(body, "class=\"chip-btn active\"\n                data-query=\"Pediatria\"") {
		t.Error("esperava que o chip Pediatria tivesse a classe 'active' ao renderizar no SSR com query=UTI,Pediatria")
	}
	if strings.Contains(body, "class=\"chip-btn active\"\n                data-query=\"Cirurgico\"") {
		t.Error("não esperava que o chip Cirurgico tivesse a classe 'active'")
	}
}

func TestWebHandler_ReloadAfterHTMX_DoesNotReturn304ForFragmentETag(t *testing.T) {
	sampleJob := createSampleJob()
	mockRepo := &mockWebJobRepo{
		searchResult: job.PaginatedJobs{
			Items:      []job.Job{sampleJob},
			Page:       1,
			Size:       20,
			Total:      1,
			TotalPages: 1,
		},
	}

	router := internalhttp.NewRouter(testLogger(), &mockDB{}, mockRepo)

	// 1. Requisição HTMX (fragmento)
	reqHTMX := httptest.NewRequest(http.MethodGet, "/jobs?query=UTI", nil)
	reqHTMX.Header.Set("HX-Request", "true")
	recHTMX := httptest.NewRecorder()
	router.ServeHTTP(recHTMX, reqHTMX)

	if recHTMX.Code != http.StatusOK {
		t.Fatalf("esperava status 200 para fragmento HTMX, obteve: %d", recHTMX.Code)
	}

	vary := recHTMX.Header().Get("Vary")
	if !strings.Contains(vary, "HX-Request") {
		t.Errorf("esperava cabeçalho Vary com HX-Request, obteve: %q", vary)
	}

	cacheControl := recHTMX.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "no-store") {
		t.Errorf("esperava Cache-Control com no-store para fragmento HTMX, obteve: %q", cacheControl)
	}

	fragETag := recHTMX.Header().Get("ETag")
	if fragETag == "" {
		t.Fatal("esperava ETag presente na resposta do fragmento")
	}

	// 2. F5 / Reload no navegador: requisição completa com If-None-Match do fragmento
	reqReload := httptest.NewRequest(http.MethodGet, "/jobs?query=UTI", nil)
	reqReload.Header.Set("If-None-Match", fragETag)
	recReload := httptest.NewRecorder()
	router.ServeHTTP(recReload, reqReload)

	// O servidor NÃO deve responder 304 com o ETag do fragmento para uma requisição de página completa!
	if recReload.Code != http.StatusOK {
		t.Fatalf("esperava status 200 (renderização completa da página no reload), obteve: %d", recReload.Code)
	}

	body := recReload.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("esperava página completa com <!DOCTYPE html> no reload")
	}
	if !strings.Contains(body, "/static/css/styles.css") {
		t.Error("esperava link de folha de estilos CSS na página recarregada")
	}
}

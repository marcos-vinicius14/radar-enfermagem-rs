package http

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/web"
)

// WebHandler manipula requisições do frontend HTMX e renderização de páginas HTML.
type WebHandler struct {
	repo   job.Repository
	view   *web.ViewEngine
	logger *slog.Logger
}

// NewWebHandler cria uma nova instância de WebHandler.
func NewWebHandler(repo job.Repository, view *web.ViewEngine, logger *slog.Logger) *WebHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &WebHandler{
		repo:   repo,
		view:   view,
		logger: logger,
	}
}

// HandleHome manipula GET / renderizando a página inicial completa (SSR).
func (h *WebHandler) HandleHome(w http.ResponseWriter, r *http.Request) {
	h.renderJobsPageOrFragment(w, r, false)
}

// HandleJobs manipula GET /jobs.
// Se for requisição HTMX (HX-Request: true), renderiza exclusivamente o fragmento da listagem.
// Caso contrário (ex.: recarregamento ou link direto), renderiza a página completa com o estado dos filtros.
func (h *WebHandler) HandleJobs(w http.ResponseWriter, r *http.Request) {
	isHTMX := isHTMXRequest(r)
	h.renderJobsPageOrFragment(w, r, isHTMX)
}

func (h *WebHandler) renderJobsPageOrFragment(w http.ResponseWriter, r *http.Request, fragmentOnly bool) {
	if h.repo == nil || h.view == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("<!DOCTYPE html><html><body><h1>Serviço temporariamente indisponível</h1><p>Não foi possível conectar ao banco de dados.</p></body></html>"))
		return
	}

	q := r.URL.Query()

	page := 1
	if pageStr := q.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	size := 20
	if sizeStr := q.Get("size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 {
			if s > 100 {
				size = 100
			} else {
				size = s
			}
		}
	}

	status := "ACTIVE"
	if statusParam := q.Get("status"); statusParam != "" {
		if strings.EqualFold(statusParam, "all") {
			status = ""
		} else {
			status = strings.ToUpper(statusParam)
		}
	}

	dateParam := q.Get("date")
	publishedSince, publishedUntil, _ := parseDateFilter(dateParam, time.Now().UTC())

	filterParams := job.FilterParams{
		Query:          strings.TrimSpace(q.Get("query")),
		City:           strings.TrimSpace(q.Get("city")),
		State:          strings.TrimSpace(q.Get("state")),
		Company:        strings.TrimSpace(q.Get("company")),
		Source:         strings.TrimSpace(q.Get("source")),
		Status:         status,
		PublishedSince: publishedSince,
		PublishedUntil: publishedUntil,
		Page:           page,
		Size:           size,
	}

	// 1. Busca as vagas paginadas
	paginated, err := h.repo.Search(r.Context(), filterParams)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "falha ao buscar vagas para renderizacao web",
			slog.String("query", filterParams.Query),
			slog.String("erro", err.Error()),
		)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("<div class='empty-state'><h3 class='empty-title'>Erro ao carregar vagas</h3><p class='empty-description'>Ocorreu uma instabilidade momentânea. Por favor, tente novamente.</p></div>"))
		return
	}

	// 2. Mecanismo de Cache ETag e validação HTTP contra F5 excessivo
	etag := calculateETag(paginated, filterParams)
	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// 3. Busca lista de cidades e instituições ativas para os filtros (apenas se página completa)
	var citiesList []web.CityItem
	var companiesList []web.CompanyItem

	if !fragmentOnly {
		dbCities, err := h.repo.ListCities(r.Context(), "ACTIVE")
		if err == nil {
			citiesList = make([]web.CityItem, 0, len(dbCities))
			for _, c := range dbCities {
				citiesList = append(citiesList, web.CityItem{
					City:      c.City,
					State:     c.State,
					TotalJobs: c.TotalJobs,
				})
			}
		}

		dbCompanies, err := h.repo.ListCompanies(r.Context(), "ACTIVE")
		if err == nil {
			companiesList = make([]web.CompanyItem, 0, len(dbCompanies))
			for _, c := range dbCompanies {
				companiesList = append(companiesList, web.CompanyItem{
					Name:      c.Name,
					TotalJobs: c.TotalJobs,
				})
			}
		}
	}

	// 4. Mapeamento para ViewModels da camada web
	jobItems := make([]web.JobItem, 0, len(paginated.Items))
	for _, item := range paginated.Items {
		jobItems = append(jobItems, web.JobItem{
			ID:             item.ID,
			Title:          item.Title,
			Company:        item.Company,
			City:           item.City,
			State:          item.State,
			Source:         item.Source,
			SourceURL:      item.SourceURL,
			WorkMode:       string(item.WorkMode),
			EmploymentType: string(item.EmploymentType),
			SalaryMin:      item.SalaryMin,
			SalaryMax:      item.SalaryMax,
			PublishedAt:    item.PublishedAt,
			Description:    item.Description,
			Status:         string(item.Status),
		})
	}

	pageData := web.PageData{
		Items:      jobItems,
		Page:       paginated.Page,
		Size:       paginated.Size,
		Total:      paginated.Total,
		TotalPages: paginated.TotalPages,
		Params: web.FilterParams{
			Query:   filterParams.Query,
			City:    filterParams.City,
			Company: filterParams.Company,
			Status:  filterParams.Status,
		},
		DateFilter: dateParam,
		Cities:     citiesList,
		Companies:  companiesList,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("ETag", etag)

	if fragmentOnly {
		w.Header().Set("Cache-Control", "no-cache")
		if err := h.view.RenderJobListFragment(w, pageData); err != nil {
			h.logger.ErrorContext(r.Context(), "falha ao renderizar fragmento HTMX", "erro", err.Error())
		}
	} else {
		w.Header().Set("Cache-Control", "public, max-age=5, must-revalidate")
		if err := h.view.RenderIndex(w, pageData); err != nil {
			h.logger.ErrorContext(r.Context(), "falha ao renderizar pagina index", "erro", err.Error())
		}
	}
}

func calculateETag(paginated job.PaginatedJobs, params job.FilterParams) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d-%d-%d-q:%s-c:%s-co:%s", paginated.Total, paginated.Page, len(paginated.Items), params.Query, params.City, params.Company)
	if len(paginated.Items) > 0 {
		first := paginated.Items[0]
		fmt.Fprintf(&sb, "-f:%s-%s", first.ID.String(), first.UpdatedAt.Format(time.RFC3339Nano))
	}
	hash := sha256.Sum256([]byte(sb.String()))
	return `"` + hex.EncodeToString(hash[:8]) + `"`
}

func isHTMXRequest(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}

package http

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

// JobsHandler gerencia requisições HTTP para vagas de emprego e seus metadados.
type JobsHandler struct {
	repo   job.Repository
	logger *slog.Logger
}

// NewJobsHandler cria uma nova instância de JobsHandler.
func NewJobsHandler(repo job.Repository, logger *slog.Logger) *JobsHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &JobsHandler{
		repo:   repo,
		logger: logger,
	}
}

// ListJobs manipula GET /api/v1/jobs com paginação e filtros dinâmicos.
func (h *JobsHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		respondError(w, http.StatusServiceUnavailable, "serviço indisponível: banco de dados desconectado")
		return
	}

	q := r.URL.Query()

	page := 1
	if pageStr := q.Get("page"); pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "parâmetro de página inválido")
			return
		}
		if p < 1 {
			p = 1
		}
		page = p
	}

	size := 20
	if sizeStr := q.Get("size"); sizeStr != "" {
		s, err := strconv.Atoi(sizeStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "parâmetro de tamanho de página inválido")
			return
		}
		if s < 1 {
			s = 20
		} else if s > 100 {
			s = 100
		}
		size = s
	}

	// Status padrão: ACTIVE caso não especificado. Se informado "all", remove o filtro.
	status := "ACTIVE"
	if statusParam := q.Get("status"); statusParam != "" {
		if strings.EqualFold(statusParam, "all") {
			status = ""
		} else {
			status = strings.ToUpper(statusParam)
		}
	}

	publishedSince, publishedUntil, err := parseDateFilter(q.Get("date"), time.Now().UTC())
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	params := job.FilterParams{
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

	result, err := h.repo.Search(r.Context(), params)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "falha ao buscar vagas na API",
			slog.String("query", params.Query),
			slog.String("erro", err.Error()),
		)
		respondError(w, http.StatusInternalServerError, "falha interna ao listar vagas")
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// GetJobByID manipula GET /api/v1/jobs/{id}.
func (h *JobsHandler) GetJobByID(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		respondError(w, http.StatusServiceUnavailable, "serviço indisponível: banco de dados desconectado")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "identificador de vaga inválido")
		return
	}

	foundJob, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, job.ErrNotFound) {
			respondError(w, http.StatusNotFound, "vaga não encontrada")
			return
		}

		h.logger.ErrorContext(r.Context(), "falha ao buscar vaga por ID na API",
			slog.String("id", id.String()),
			slog.String("erro", err.Error()),
		)
		respondError(w, http.StatusInternalServerError, "falha ao buscar detalhes da vaga")
		return
	}

	respondJSON(w, http.StatusOK, foundJob)
}

// ListCompanies manipula GET /api/v1/companies.
func (h *JobsHandler) ListCompanies(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		respondError(w, http.StatusServiceUnavailable, "serviço indisponível: banco de dados desconectado")
		return
	}

	status := parseStatusParam(r.URL.Query().Get("status"))

	companies, err := h.repo.ListCompanies(r.Context(), status)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "falha ao listar empresas na API",
			slog.String("status", status),
			slog.String("erro", err.Error()),
		)
		respondError(w, http.StatusInternalServerError, "falha ao listar empresas")
		return
	}

	respondJSON(w, http.StatusOK, companies)
}

// ListCities manipula GET /api/v1/cities.
func (h *JobsHandler) ListCities(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		respondError(w, http.StatusServiceUnavailable, "serviço indisponível: banco de dados desconectado")
		return
	}

	status := parseStatusParam(r.URL.Query().Get("status"))

	cities, err := h.repo.ListCities(r.Context(), status)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "falha ao listar cidades na API",
			slog.String("status", status),
			slog.String("erro", err.Error()),
		)
		respondError(w, http.StatusInternalServerError, "falha ao listar cidades")
		return
	}

	respondJSON(w, http.StatusOK, cities)
}

// ListSources manipula GET /api/v1/sources.
func (h *JobsHandler) ListSources(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		respondError(w, http.StatusServiceUnavailable, "serviço indisponível: banco de dados desconectado")
		return
	}

	status := parseStatusParam(r.URL.Query().Get("status"))

	sources, err := h.repo.ListSources(r.Context(), status)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "falha ao listar fontes na API",
			slog.String("status", status),
			slog.String("erro", err.Error()),
		)
		respondError(w, http.StatusInternalServerError, "falha ao listar fontes")
		return
	}

	respondJSON(w, http.StatusOK, sources)
}

func parseStatusParam(statusParam string) string {
	if statusParam == "" {
		return "ACTIVE"
	}
	if strings.EqualFold(statusParam, "all") {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(statusParam))
}

func parseDateFilter(dateStr string, now time.Time) (*time.Time, *time.Time, error) {
	dateStr = strings.TrimSpace(strings.ToLower(dateStr))
	if dateStr == "" {
		return nil, nil, nil
	}

	switch dateStr {
	case "today", "hoje":
		since := now.Add(-24 * time.Hour)
		return &since, nil, nil
	case "3d":
		since := now.Add(-3 * 24 * time.Hour)
		return &since, nil, nil
	case "7d", "week", "semana":
		since := now.Add(-7 * 24 * time.Hour)
		return &since, nil, nil
	case "30d", "month", "mes":
		since := now.Add(-30 * 24 * time.Hour)
		return &since, nil, nil
	default:
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, nil, fmt.Errorf("formato de data inválido, use 'today', '3d', '7d', '30d' ou 'AAAA-MM-DD'")
		}
		startOfDay := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, time.UTC)
		endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)
		return &startOfDay, &endOfDay, nil
	}
}

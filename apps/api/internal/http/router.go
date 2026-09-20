package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/logger"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/web"
	"golang.org/x/time/rate"
)

// DB define o contrato mínimo para checagem de saúde da base de dados.
type DB interface {
	Ping(ctx context.Context) error
}

// NewRouter cria e configura o roteador Chi com middlewares, observabilidade, rotas web e API.
func NewRouter(l *slog.Logger, db DB, jobRepo job.Repository) http.Handler {
	if l == nil {
		l = slog.Default()
	}

	r := chi.NewRouter()

	// 1. Inicializa o ViewEngine de templates HTML embutidos
	view, err := web.NewViewEngine()
	if err != nil {
		l.Error("falha crítica ao inicializar view engine dos templates HTML", "erro", err)
	}

	// 2. Middlewares essenciais
	r.Use(middleware.RequestID)
	r.Use(logger.Middleware(l))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// 3. Middleware de Rate Limiting por IP (Token Bucket contra F5 e cliques múltiplos)
	rateLimiter := NewIPRateLimiter(rate.Limit(5), 15, view, l)
	r.Use(rateLimiter.Middleware())

	// 4. Rotas de observabilidade
	r.Get("/health", handleHealth)
	r.Get("/ready", handleReady(db))

	// 5. Arquivos estáticos (CSS, JS, Favicon com cache imutável)
	staticFileServer := http.StripPrefix("/static/", http.FileServer(web.StaticFS()))
	r.Get("/static/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
		staticFileServer.ServeHTTP(w, r)
	})

	// 6. Rotas do Frontend Web HTMX
	webHandler := NewWebHandler(jobRepo, view, l)
	r.Get("/", webHandler.HandleHome)
	r.Get("/jobs", webHandler.HandleJobs)

	// 7. Rotas da API v1 JSON
	jobsHandler := NewJobsHandler(jobRepo, l)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/jobs", jobsHandler.ListJobs)
		r.Get("/jobs/{id}", jobsHandler.GetJobByID)
		r.Get("/companies", jobsHandler.ListCompanies)
		r.Get("/cities", jobsHandler.ListCities)
		r.Get("/sources", jobsHandler.ListSources)
	})

	// 8. 404 Handler
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		respondError(w, http.StatusNotFound, "recurso não encontrado")
	})

	return r
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func handleReady(db DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			respondJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status":   "unavailable",
				"database": "down",
				"error":    err.Error(),
			})
			return
		}

		respondJSON(w, http.StatusOK, map[string]any{
			"status":   "ready",
			"database": "up",
		})
	}
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}

// WithTestPanicRoute permite anexar uma rota de teste ao router (usado exclusivamente em testes).
func WithTestPanicRoute(handler http.Handler, path string, panicHandler http.Handler) http.Handler {
	mux, ok := handler.(*chi.Mux)
	if ok {
		mux.Handle(path, panicHandler)
		return mux
	}
	r := chi.NewRouter()
	r.Mount("/", handler)
	r.Handle(path, panicHandler)
	return r
}

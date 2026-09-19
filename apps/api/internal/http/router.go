package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/logger"
)

// DB define o contrato mínimo para checagem de saúde da base de dados.
type DB interface {
	Ping(ctx context.Context) error
}

// NewRouter cria e configura o roteador Chi com middlewares e rotas padrão.
func NewRouter(l *slog.Logger, db DB) http.Handler {
	r := chi.NewRouter()

	// Middlewares essenciais
	r.Use(middleware.RequestID)
	r.Use(logger.Middleware(l))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Rotas de observabilidade
	r.Get("/health", handleHealth)
	r.Get("/ready", handleReady(db))

	// 404 Handler em JSON
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "recurso não encontrado",
		})
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

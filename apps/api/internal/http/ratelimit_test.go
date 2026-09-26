package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	internalhttp "github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/http"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/web"
	"golang.org/x/time/rate"
)

func TestRateLimiter_AllowsNormalRequests(t *testing.T) {
	view, err := web.NewViewEngine()
	if err != nil {
		t.Fatalf("erro ao criar view engine: %v", err)
	}

	rl := internalhttp.NewIPRateLimiter(rate.Limit(10), 5, view, nil)
	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperava status 200 para requisição normal, obteve: %d", rec.Code)
	}
}

func TestRateLimiter_BlocksExcessiveRequests(t *testing.T) {
	view, err := web.NewViewEngine()
	if err != nil {
		t.Fatalf("erro ao criar view engine: %v", err)
	}

	// Limiter restrito: 1 req/s, burst de 3
	rl := internalhttp.NewIPRateLimiter(rate.Limit(1), 3, view, nil)
	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ip := "203.0.113.10:4321"

	// As 3 primeiras requisições devem passar (burst = 3)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("esperava status 200 no burst %d, obteve: %d", i+1, rec.Code)
		}
	}

	// A 4ª requisição rápida consecutiva (dedos nervosos/F5 repetido) deve ser bloqueada com 429
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ip
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("esperava status 429 Too Many Requests, obteve: %d", rec.Code)
	}

	if retryAfter := rec.Header().Get("Retry-After"); retryAfter == "" {
		t.Error("esperava cabeçalho Retry-After presente na resposta 429")
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Calma aí!") {
		t.Errorf("esperava página amigável de rate limit, obteve: %s", body)
	}
}

func TestRateLimiter_JSONRequestError(t *testing.T) {
	rl := internalhttp.NewIPRateLimiter(rate.Limit(1), 1, nil, nil)
	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ip := "198.51.100.22:8000"

	// 1ª requisição consome o burst
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	req1.RemoteAddr = ip
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve: %d", rec1.Code)
	}

	// 2ª requisição imediata via JSON
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	req2.RemoteAddr = ip
	req2.Header.Set("Accept", "application/json")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("esperava 429, obteve: %d", rec2.Code)
	}

	body := rec2.Body.String()
	if !strings.Contains(body, "muitas requisições") {
		t.Errorf("esperava mensagem de erro JSON em pt-BR, obteve: %s", body)
	}
}

func TestRateLimiter_CleanupInactive(t *testing.T) {
	rl := internalhttp.NewIPRateLimiter(rate.Limit(1), 1, nil, nil)
	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Imediatamente não deve limpar se maxAge for alto
	rl.CleanupInactive(1 * time.Hour)

	// Com maxAge 0, deve limpar
	rl.CleanupInactive(0)
}

func TestRateLimiter_BypassesStaticAssets(t *testing.T) {
	rl := internalhttp.NewIPRateLimiter(rate.Limit(1), 1, nil, nil)
	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ip := "198.51.100.99:8000"

	// Faz 10 requisições seguidas para arquivos estáticos - nenhuma deve ser bloqueada
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/static/css/styles.css", nil)
		req.RemoteAddr = ip
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("esperava 200 para asset estático na tentativa %d, obteve: %d", i+1, rec.Code)
		}
	}
}

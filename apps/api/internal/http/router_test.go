package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	internalhttp "github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/http"
)

type mockDB struct {
	pingErr error
}

func (m *mockDB) Ping(ctx context.Context) error {
	return m.pingErr
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHealthEndpoint(t *testing.T) {
	router := internalhttp.NewRouter(testLogger(), &mockDB{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got: %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode json response: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got: %v", body["status"])
	}
	if _, ok := body["timestamp"]; !ok {
		t.Errorf("expected 'timestamp' field in response")
	}
}

func TestReadyEndpoint_HealthyDB(t *testing.T) {
	router := internalhttp.NewRouter(testLogger(), &mockDB{pingErr: nil}, nil)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got: %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode json response: %v", err)
	}

	if body["status"] != "ready" {
		t.Errorf("expected status 'ready', got: %v", body["status"])
	}
	if body["database"] != "up" {
		t.Errorf("expected database 'up', got: %v", body["database"])
	}
}

func TestReadyEndpoint_UnhealthyDB(t *testing.T) {
	router := internalhttp.NewRouter(testLogger(), &mockDB{pingErr: errors.New("connection refused")}, nil)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got: %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode json response: %v", err)
	}

	if body["status"] != "unavailable" {
		t.Errorf("expected status 'unavailable', got: %v", body["status"])
	}
	if body["database"] != "down" {
		t.Errorf("expected database 'down', got: %v", body["database"])
	}
}

func TestRecoverer_HandlesPanicGracefully(t *testing.T) {
	router := internalhttp.NewRouter(testLogger(), &mockDB{}, nil)

	// Add panic route for testing
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("unexpected runtime error!")
	})

	testMux := internalhttp.WithTestPanicRoute(router, "/test-panic", panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/test-panic", nil)
	rec := httptest.NewRecorder()

	testMux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected recoverer to catch panic and return 500, got: %d", rec.Code)
	}
}

func TestNotFoundHandler(t *testing.T) {
	router := internalhttp.NewRouter(testLogger(), &mockDB{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/non-existent-route", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got: %d", rec.Code)
	}
}

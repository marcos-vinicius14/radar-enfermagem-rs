package logger_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/logger"
)

func TestNew_LevelsAndFormats(t *testing.T) {
	tests := []struct {
		name         string
		levelStr     string
		env          string
		logAction    func(l *slog.Logger)
		expectLogged bool
		expectJSON   bool
	}{
		{
			name:     "production json format",
			levelStr: "info",
			env:      "production",
			logAction: func(l *slog.Logger) {
				l.Info("test prod message")
			},
			expectLogged: true,
			expectJSON:   true,
		},
		{
			name:     "development text format",
			levelStr: "debug",
			env:      "development",
			logAction: func(l *slog.Logger) {
				l.Debug("test dev debug")
			},
			expectLogged: true,
			expectJSON:   false,
		},
		{
			name:     "level filtering filters lower priority",
			levelStr: "warn",
			env:      "production",
			logAction: func(l *slog.Logger) {
				l.Info("should not appear")
			},
			expectLogged: false,
			expectJSON:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := logger.NewWithWriter(&buf, tt.levelStr, tt.env)

			tt.logAction(l)
			output := buf.String()

			if tt.expectLogged && len(output) == 0 {
				t.Fatalf("expected log output, got empty string")
			}
			if !tt.expectLogged && len(output) > 0 {
				t.Fatalf("expected no log output, got: %s", output)
			}
			if tt.expectLogged && tt.expectJSON {
				if !strings.HasPrefix(output, "{") {
					t.Errorf("expected JSON format starting with '{', got: %s", output)
				}
			}
		})
	}
}

func TestMiddleware_RequestLogging(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewWithWriter(&buf, "debug", "production")

	handler := logger.Middleware(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "/test-path") {
		t.Errorf("expected log to contain path '/test-path', got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "201") {
		t.Errorf("expected log to contain status '201', got: %s", logOutput)
	}
}

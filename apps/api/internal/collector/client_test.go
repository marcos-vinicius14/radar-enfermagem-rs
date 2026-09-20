package collector_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
)

func TestDomainRateLimiter(t *testing.T) {
	limiter := collector.NewDomainRateLimiter(collector.RateLimiterConfig{
		DefaultRPS:   10,
		DefaultBurst: 1,
		DomainOverrides: map[string]collector.RateLimitRule{
			"custom.example.com": {RPS: 5, Burst: 1},
		},
	}, nil)

	ctx := context.Background()

	// Primeira chamada não deve bloquear
	start := time.Now()
	if err := limiter.Wait(ctx, "custom.example.com"); err != nil {
		t.Fatalf("Wait() falhou: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Errorf("primeira chamada demorou muito: %v", elapsed)
	}

	// Respeita cancelamento de contexto
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	err := limiter.Wait(canceledCtx, "custom.example.com")
	if err == nil {
		t.Errorf("esperava erro ao chamar Wait com contexto cancelado")
	}
}

func TestRetryTransport_RecuperaDeFalhaTransitoria(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("sucesso"))
	}))
	defer server.Close()

	client := collector.NewResilientHTTPClient(collector.ResilientClientConfig{
		Timeout:           2 * time.Second,
		MaxRetries:        3,
		InitialRetryDelay: 10 * time.Millisecond,
		DefaultRPS:        50,
		DefaultBurst:      10,
	}, nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest falhou: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do falhou: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, esperado %d", resp.StatusCode, http.StatusOK)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("tentativas = %d, esperado 3", attempts)
	}
}

func TestRetryTransport_NaoRetentaErrosDefinitivos4xx(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := collector.NewResilientHTTPClient(collector.ResilientClientConfig{
		Timeout:           2 * time.Second,
		MaxRetries:        3,
		InitialRetryDelay: 10 * time.Millisecond,
		DefaultRPS:        50,
		DefaultBurst:      10,
	}, nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest falhou: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do falhou: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, esperado %d", resp.StatusCode, http.StatusNotFound)
	}
	// Erro 404 não é transitório: deve executar apenas 1 tentativa
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("tentativas = %d, esperado 1 (sem retries para 404)", attempts)
	}
}

func TestRetryTransport_RetentaTooManyRequests429(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := collector.NewResilientHTTPClient(collector.ResilientClientConfig{
		Timeout:           2 * time.Second,
		MaxRetries:        2,
		InitialRetryDelay: 10 * time.Millisecond,
		DefaultRPS:        50,
		DefaultBurst:      10,
	}, nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest falhou: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do falhou: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, esperado %d", resp.StatusCode, http.StatusOK)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("tentativas = %d, esperado 2 (retentou 429)", attempts)
	}
}

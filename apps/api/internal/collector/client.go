package collector

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitRule define os parâmetros de vazão e burst para requisições a um domínio específico.
type RateLimitRule struct {
	RPS   float64
	Burst int
}

// RateLimiterConfig configura os limites padrão e sobrescritas específicas por host/domínio.
type RateLimiterConfig struct {
	DefaultRPS      float64
	DefaultBurst    int
	DomainOverrides map[string]RateLimitRule
}

// DomainRateLimiter gerencia limitadores de taxa por domínio de forma thread-safe.
type DomainRateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	cfg      RateLimiterConfig
	logger   *slog.Logger
}

// NewDomainRateLimiter instancia o gerenciador de rate limit por domínio.
func NewDomainRateLimiter(cfg RateLimiterConfig, logger *slog.Logger) *DomainRateLimiter {
	if cfg.DefaultRPS <= 0 {
		cfg.DefaultRPS = 3.0
	}
	if cfg.DefaultBurst <= 0 {
		cfg.DefaultBurst = 5
	}
	if cfg.DomainOverrides == nil {
		cfg.DomainOverrides = make(map[string]RateLimitRule)
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &DomainRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		cfg:      cfg,
		logger:   logger,
	}
}

// Wait aguarda liberação de taxa para enviar requisições ao domínio especificado.
func (d *DomainRateLimiter) Wait(ctx context.Context, host string) error {
	limiter := d.getLimiter(host)

	start := time.Now()
	err := limiter.Wait(ctx)
	if err != nil {
		return fmt.Errorf("aguardar rate limit para %s: %w", host, err)
	}

	waited := time.Since(start)
	if waited > 50*time.Millisecond {
		d.logger.DebugContext(ctx, "requisicao aguardou rate limit",
			slog.String("dominio", host),
			slog.Duration("espera", waited),
		)
	}

	return nil
}

func (d *DomainRateLimiter) getLimiter(host string) *rate.Limiter {
	// Normaliza host (remove porta se existir)
	cleanHost := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		cleanHost = h
	}
	cleanHost = strings.ToLower(cleanHost)

	d.mu.RLock()
	lim, exists := d.limiters[cleanHost]
	d.mu.RUnlock()
	if exists {
		return lim
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Checagem dupla após adquirir lock de escrita
	if lim, exists = d.limiters[cleanHost]; exists {
		return lim
	}

	rps := d.cfg.DefaultRPS
	burst := d.cfg.DefaultBurst

	// Verifica se há regra direta ou por sufixo de domínio (ex: gupy.io)
	for domain, rule := range d.cfg.DomainOverrides {
		if cleanHost == domain || strings.HasSuffix(cleanHost, "."+domain) {
			if rule.RPS > 0 {
				rps = rule.RPS
			}
			if rule.Burst > 0 {
				burst = rule.Burst
			}
			break
		}
	}

	lim = rate.NewLimiter(rate.Limit(rps), burst)
	d.limiters[cleanHost] = lim
	return lim
}

// RateLimitedTransport intercepta requisições HTTP e aplica rate limiting por domínio.
type RateLimitedTransport struct {
	base    http.RoundTripper
	limiter *DomainRateLimiter
}

func NewRateLimitedTransport(base http.RoundTripper, limiter *DomainRateLimiter) *RateLimitedTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &RateLimitedTransport{
		base:    base,
		limiter: limiter,
	}
}

func (t *RateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.limiter != nil && req.URL != nil {
		if err := t.limiter.Wait(req.Context(), req.URL.Host); err != nil {
			return nil, err
		}
	}
	return t.base.RoundTrip(req)
}

// RetryTransport intercepta requisições HTTP e retenta falhas transitórias com backoff escalonado.
type RetryTransport struct {
	base              http.RoundTripper
	maxRetries        int
	initialRetryDelay time.Duration
	logger            *slog.Logger
}

func NewRetryTransport(base http.RoundTripper, maxRetries int, initialDelay time.Duration, logger *slog.Logger) *RetryTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	if maxRetries < 0 {
		maxRetries = 0
	}
	if initialDelay <= 0 {
		initialDelay = 200 * time.Millisecond
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &RetryTransport{
		base:              base,
		maxRetries:        maxRetries,
		initialRetryDelay: initialDelay,
		logger:            logger,
	}
}

func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Se o corpo da requisição estiver presente e não houver GetBody, lemos o corpo para permitir repetição
	if req.Body != nil && req.GetBody == nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("ler corpo da requisicao para retentativas: %w", err)
		}
		_ = req.Body.Close()
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
		req.Body, _ = req.GetBody()
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		// Restaura o corpo da requisição para cada tentativa
		if req.GetBody != nil {
			body, getErr := req.GetBody()
			if getErr != nil {
				return nil, fmt.Errorf("obter corpo da requisicao na tentativa %d: %w", attempt+1, getErr)
			}
			req.Body = body
		}

		resp, err = t.base.RoundTrip(req)

		// Verifica se a resposta ou o erro exige retentativa
		isTransient, reason, retryAfter := t.isTransientFailure(req.Context(), resp, err)
		if !isTransient || attempt == t.maxRetries {
			break
		}

		// Fecha o corpo da resposta anterior antes de tentar novamente
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}

		backoff := t.calculateBackoff(attempt, retryAfter)

		t.logger.WarnContext(req.Context(), "retentando requisicao http apos falha transitoria",
			slog.String("url", req.URL.String()),
			slog.String("metodo", req.Method),
			slog.Int("tentativa", attempt+1),
			slog.Int("max_tentativas", t.maxRetries),
			slog.Duration("espera_backoff", backoff),
			slog.String("motivo", reason),
		)

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(backoff):
		}
	}

	return resp, err
}

func (t *RetryTransport) isTransientFailure(ctx context.Context, resp *http.Response, err error) (bool, string, time.Duration) {
	// Se o contexto pai já foi cancelado, não devemos retentar
	if ctx.Err() != nil {
		return false, "contexto da requisicao cancelado", 0
	}

	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) {
			return true, fmt.Sprintf("erro de rede/timeout: %v", netErr), 0
		}
		return true, fmt.Sprintf("erro de transporte: %v", err), 0
	}

	if resp != nil {
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			return true, "limite de taxa excedido (HTTP 429 Too Many Requests)", retryAfter
		}
		if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
			return true, fmt.Sprintf("falha no servidor remoto (HTTP %d)", resp.StatusCode), 0
		}
	}

	return false, "", 0
}

func (t *RetryTransport) calculateBackoff(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}

	// Backoff com jitter de 20%
	multiplier := 1 << attempt
	delay := t.initialRetryDelay * time.Duration(multiplier)

	// Jitter entre 0.8x e 1.2x
	jitterFactor := 0.8 + (rand.Float64() * 0.4) // nolint:gosec
	jittered := time.Duration(float64(delay) * jitterFactor)

	if jittered > 10*time.Second {
		jittered = 10 * time.Second
	}
	return jittered
}

func parseRetryAfter(headerVal string) time.Duration {
	if headerVal == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(headerVal); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if parsedTime, err := http.ParseTime(headerVal); err == nil {
		diff := time.Until(parsedTime)
		if diff > 0 {
			return diff
		}
	}
	return 0
}

// ResilientClientConfig consolida as configurações de resiliência e concorrência HTTP.
type ResilientClientConfig struct {
	Timeout           time.Duration
	MaxRetries        int
	InitialRetryDelay time.Duration
	DefaultRPS        float64
	DefaultBurst      int
	DomainOverrides   map[string]RateLimitRule
}

// NewResilientHTTPClient cria um *http.Client totalmente equipado com rate limiting por domínio e retries.
func NewResilientHTTPClient(cfg ResilientClientConfig, logger *slog.Logger) *http.Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 15 * time.Second
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.InitialRetryDelay <= 0 {
		cfg.InitialRetryDelay = 200 * time.Millisecond
	}
	if logger == nil {
		logger = slog.Default()
	}

	rateLimiter := NewDomainRateLimiter(RateLimiterConfig{
		DefaultRPS:      cfg.DefaultRPS,
		DefaultBurst:    cfg.DefaultBurst,
		DomainOverrides: cfg.DomainOverrides,
	}, logger)

	baseTransport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	rateLimitedTransport := NewRateLimitedTransport(baseTransport, rateLimiter)
	retryTransport := NewRetryTransport(rateLimitedTransport, cfg.MaxRetries, cfg.InitialRetryDelay, logger)

	return &http.Client{
		Transport: retryTransport,
		Timeout:   cfg.Timeout,
	}
}

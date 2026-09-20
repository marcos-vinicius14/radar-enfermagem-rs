package http

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/web"
	"golang.org/x/time/rate"
)

type clientVisitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter gerencia limitadores de taxa por endereço IP com limpeza automática.
type IPRateLimiter struct {
	mu       sync.RWMutex
	visitors map[string]*clientVisitor
	rate     rate.Limit
	burst    int
	logger   *slog.Logger
	view     *web.ViewEngine
}

// NewIPRateLimiter cria um gerenciador de rate limiting com os parâmetros definidos.
func NewIPRateLimiter(r rate.Limit, burst int, view *web.ViewEngine, l *slog.Logger) *IPRateLimiter {
	if l == nil {
		l = slog.Default()
	}

	rl := &IPRateLimiter{
		visitors: make(map[string]*clientVisitor),
		rate:     r,
		burst:    burst,
		logger:   l,
		view:     view,
	}

	return rl
}

// CleanupInactiveRemove remove visitantes inativos há mais de maxAge.
func (rl *IPRateLimiter) CleanupInactive(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, v := range rl.visitors {
		if now.Sub(v.lastSeen) > maxAge {
			delete(rl.visitors, ip)
		}
	}
}

// StartCleanupRoutine inicia a limpeza periódica de IPs inativos em background.
func (rl *IPRateLimiter) StartCleanupRoutine(stopCh <-chan struct{}, interval, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				rl.CleanupInactive(maxAge)
			case <-stopCh:
				return
			}
		}
	}()
}

// getLimiter obtém ou cria um limitador para o IP fornecido.
func (rl *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = &clientVisitor{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// Middleware retorna um middleware padrão Chi para aplicar rate limiting por IP.
func (rl *IPRateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			limiter := rl.getLimiter(ip)

			if !limiter.Allow() {
				rl.logger.WarnContext(r.Context(), "limite de taxa de requisições excedido",
					slog.String("ip", ip),
					slog.String("path", r.URL.Path),
				)

				w.Header().Set("Retry-After", "2")

				// Se for requisição HTML do navegador (F5 contínuo ou clique repetido)
				if isBrowserOrHTMLRequest(r) && rl.view != nil {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.WriteHeader(http.StatusTooManyRequests)
					_ = rl.view.RenderRateLimit(w)
					return
				}

				// Para API JSON
				respondError(w, http.StatusTooManyRequests, "muitas requisições em pouco tempo, por favor aguarde alguns instantes")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getClientIP(r *http.Request) string {
	// X-Forwarded-For pode conter múltiplos IPs: client, proxy1, proxy2
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}

	xri := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if xri != "" {
		return xri
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}

func isBrowserOrHTMLRequest(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html")
}

package server

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/axiom-idp/axiom/internal/auth"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// RateLimiter implements token bucket rate limiting.
type RateLimiter struct {
	mu               sync.Mutex
	limiters         map[string]*rateEntry
	rate             rate.Limit
	burst            int
	window           time.Duration
	staleAfter       time.Duration
	lastCleanup      time.Time
	requestsSeen     int
	cleanupThreshold int
	metrics          *Metrics
	store            runtimeStateStore
	logger           *logrus.Logger
}

type RateLimiterStats struct {
	Enabled        bool      `json:"enabled"`
	TrackedKeys    int       `json:"tracked_keys"`
	RequestsSeen   int       `json:"requests_seen"`
	LastCleanupAt  time.Time `json:"last_cleanup_at,omitempty"`
	RequestsPerMin int       `json:"requests_per_min"`
}

type rateEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	if requests <= 0 || window <= 0 {
		return &RateLimiter{
			limiters: make(map[string]*rateEntry),
		}
	}

	return &RateLimiter{
		limiters:         make(map[string]*rateEntry),
		rate:             rate.Limit(float64(requests) / window.Seconds()),
		burst:            requests,
		window:           window,
		staleAfter:       window * 2,
		cleanupThreshold: requests,
	}
}

// SetMetrics wires rate limiter events into the shared telemetry collector.
func (r *RateLimiter) SetMetrics(metrics *Metrics) {
	if r == nil {
		return
	}

	r.mu.Lock()
	r.metrics = metrics
	r.mu.Unlock()
}

// SetStore wires the rate limiter into the shared runtime state store.
func (r *RateLimiter) SetStore(store runtimeStateStore) {
	if r == nil {
		return
	}

	r.mu.Lock()
	r.store = store
	r.mu.Unlock()
}

// SetLogger wires store warnings into the application logger.
func (r *RateLimiter) SetLogger(logger *logrus.Logger) {
	if r == nil {
		return
	}

	r.mu.Lock()
	r.logger = logger
	r.mu.Unlock()
}

// Allow checks whether the provided key can make a request.
func (r *RateLimiter) Allow(key string) bool {
	if key == "" {
		key = "anonymous"
	}
	if r == nil || r.rate <= 0 || r.burst <= 0 {
		return true
	}

	r.mu.Lock()
	store := r.store
	logger := r.logger
	r.mu.Unlock()

	if store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		allowed, err := store.AllowRateLimit(ctx, key, r.burst, r.window)
		cancel()
		if err == nil {
			return allowed
		}
		if logger != nil {
			logger.WithError(err).Warn("failed to evaluate shared runtime rate limit; falling back to local limiter")
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	entry, exists := r.limiters[key]
	if !exists {
		entry = &rateEntry{
			limiter: rate.NewLimiter(r.rate, r.burst),
		}
		r.limiters[key] = entry
	}
	entry.lastSeen = now

	r.requestsSeen++
	if r.cleanupThreshold > 0 && r.requestsSeen >= r.cleanupThreshold {
		r.cleanup(now)
		r.requestsSeen = 0
	}

	if entry.limiter == nil {
		return true
	}

	return entry.limiter.Allow()
}

func (r *RateLimiter) cleanup(now time.Time) {
	if len(r.limiters) == 0 || r.staleAfter <= 0 {
		return
	}

	for key, entry := range r.limiters {
		if now.Sub(entry.lastSeen) > r.staleAfter {
			delete(r.limiters, key)
		}
	}
	r.lastCleanup = now
}

func rateLimitKey(r *http.Request) string {
	if userID := auth.UserIDFromContext(r.Context()); userID != "" {
		return "user:" + userID
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return "ip:" + host
	}

	if r.RemoteAddr != "" {
		return "ip:" + r.RemoteAddr
	}

	return "anonymous"
}

// Middleware returns a rate limiting middleware.
func (r *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/health", "/live", "/ready", "/metrics":
			next.ServeHTTP(w, req)
			return
		}

		if req.Method == http.MethodOptions {
			next.ServeHTTP(w, req)
			return
		}

		if r == nil {
			next.ServeHTTP(w, req)
			return
		}

		if !r.Allow(rateLimitKey(req)) {
			r.mu.Lock()
			metrics := r.metrics
			r.mu.Unlock()
			if metrics != nil {
				metrics.RecordRateLimit()
			}
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, req)
	})
}

func (r *RateLimiter) Stats() RateLimiterStats {
	if r == nil {
		return RateLimiterStats{}
	}

	r.mu.Lock()
	store := r.store
	ratePerMin := r.burst
	r.mu.Unlock()
	if store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if stats, err := store.RateLimitStats(ctx); err == nil {
			stats.Enabled = r.rate > 0 && r.burst > 0
			stats.RequestsPerMin = ratePerMin
			return stats
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return RateLimiterStats{
		Enabled:        r.rate > 0 && r.burst > 0,
		TrackedKeys:    len(r.limiters),
		RequestsSeen:   r.requestsSeen,
		LastCleanupAt:  r.lastCleanup,
		RequestsPerMin: r.burst,
	}
}

// SecurityHeaders adds security headers to responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

func containsOrigin(origins []string, origin string) bool {
	for _, allowed := range origins {
		if allowed == "*" || strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

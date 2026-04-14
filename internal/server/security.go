package server

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/axiom-idp/axiom/internal/auth"
	"golang.org/x/time/rate"
)

// RateLimiter implements token bucket rate limiting.
type RateLimiter struct {
	mu               sync.Mutex
	limiters         map[string]*rateEntry
	rate             rate.Limit
	burst            int
	staleAfter       time.Duration
	lastCleanup      time.Time
	requestsSeen     int
	cleanupThreshold int
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
		staleAfter:       window * 2,
		cleanupThreshold: requests,
	}
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
		if req.Method == http.MethodOptions || req.URL.Path == "/health" {
			next.ServeHTTP(w, req)
			return
		}

		if r == nil {
			next.ServeHTTP(w, req)
			return
		}

		if !r.Allow(rateLimitKey(req)) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, req)
	})
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

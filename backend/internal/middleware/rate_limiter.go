package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/OpenNSW/nsw/internal/auth"
)

// clientLimiter holds the rate limiter for a specific key (IP or TraderID)
type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IdentityRateLimiter manages rate limiting by identity (TraderID) then falls back to IP
type IdentityRateLimiter struct {
	mu      sync.RWMutex
	clients map[string]*clientLimiter
	limit   rate.Limit
	burst   int
}

// NewIdentityRateLimiter creates a new rate limiter middleware.
// limit: events per second (e.g., rate.Limit(5))
// burst: maximum burst size (e.g., 20)
func NewIdentityRateLimiter(limit rate.Limit, burst int) *IdentityRateLimiter {
	l := &IdentityRateLimiter{
		clients: make(map[string]*clientLimiter),
		limit:   limit,
		burst:   burst,
	}

	go l.cleanupRoutine()

	return l
}

// cleanupRoutine periodically removes old client limiters
func (l *IdentityRateLimiter) cleanupRoutine() {
	for {
		time.Sleep(10 * time.Minute)

		l.mu.Lock()
		for key, client := range l.clients {
			// Remove clients not seen in the last hour
			if time.Since(client.lastSeen) > time.Hour {
				delete(l.clients, key)
			}
		}
		l.mu.Unlock()
	}
}

// getLimiter retrieves or creates a limiter for the given key
func (l *IdentityRateLimiter) getLimiter(key string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	cl, exists := l.clients[key]
	if !exists {
		limiter := rate.NewLimiter(l.limit, l.burst)
		cl = &clientLimiter{limiter: limiter, lastSeen: time.Now()}
		l.clients[key] = cl
		return limiter
	}

	cl.lastSeen = time.Now()
	return cl.limiter
}

// Handler returns the middleware implementation
func (l *IdentityRateLimiter) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract identity key: TraderID or fallback to RemoteAddr
			key := r.RemoteAddr
			authCtx := auth.GetAuthContext(r.Context())
			if authCtx != nil && authCtx.TraderID != "" {
				key = authCtx.TraderID
			}

			limiter := l.getLimiter(key)

			if !limiter.Allow() {
				slog.WarnContext(r.Context(), "Rate limit exceeded", "key", key)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error": "Too Many Requests", "message": "Rate limit exceeded. Please try again later."}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

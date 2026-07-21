package auth

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"central-devtron/internal/httpx"
)

const cookieName = "cd_session"

type ctxKey int

const userCtxKey ctxKey = 0

// Service bundles everything the auth handlers and middleware need.
type Service struct {
	store        *Store
	secret       []byte
	ttl          time.Duration
	cookieSecure bool
	limiter      *limiter
}

func NewService(store *Store, secret []byte, ttl time.Duration, cookieSecure bool) *Service {
	return &Service{
		store:        store,
		secret:       secret,
		ttl:          ttl,
		cookieSecure: cookieSecure,
		limiter:      newLimiter(5, 15*time.Minute),
	}
}

// RequireAuth wraps a handler, rejecting any request without a valid session.
func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := s.readSession(r)
		if !ok {
			httpx.Error(w, http.StatusUnauthorized, "authentication required")
			return
		}
		ctx := context.WithValue(r.Context(), userCtxKey, c)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// readSession extracts and verifies the session cookie.
func (s *Service) readSession(r *http.Request) (claims, bool) {
	ck, err := r.Cookie(cookieName)
	if err != nil || ck.Value == "" {
		return claims{}, false
	}
	c, err := verify(s.secret, ck.Value)
	if err != nil {
		return claims{}, false
	}
	return c, true
}

func (s *Service) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(s.ttl.Seconds()),
	})
}

func (s *Service) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

// ── Rate limiter (per client IP) ────────────────────────────────────────────

type limiter struct {
	mu       sync.Mutex
	hits     map[string][]time.Time
	max      int
	window   time.Duration
}

func newLimiter(max int, window time.Duration) *limiter {
	return &limiter{hits: map[string][]time.Time{}, max: max, window: window}
}

// allow records an attempt and reports whether it is within the limit.
func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-l.window)

	recent := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if len(recent) >= l.max {
		l.hits[key] = recent
		return false
	}
	l.hits[key] = append(recent, now)
	return true
}

func clientIP(r *http.Request) string {
	// Trust the socket peer; do NOT trust X-Forwarded-For unless behind a
	// known proxy (configure that at the proxy layer).
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Package server wires all module handlers into a single HTTP mux.
package server

import (
	"database/sql"
	"net/http"

	"knowhere/internal/auth"
	"knowhere/internal/config"
	"knowhere/internal/devtron"
	"knowhere/internal/httpx"
	"knowhere/internal/license"
	"knowhere/internal/onboarding"
	"knowhere/internal/web"
	"knowhere/internal/zoho"
)

// New builds the top-level HTTP handler with all routes and middleware.
func New(cfg *config.Config, database *sql.DB, authSvc *auth.Service) http.Handler {
	// Protected mux: every module endpoint. Only reachable through RequireAuth.
	protected := http.NewServeMux()
	zoho.NewHandler(zoho.New(cfg.Zoho)).Register(protected)
	devtron.NewHandler(devtron.New(cfg.Devtron)).Register(protected)
	license.NewHandler(license.NewStore(database)).Register(protected)
	onboarding.NewHandler(onboarding.NewStore(database)).Register(protected)
	// Authenticated view of integration status.
	protected.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]any{"missing": cfg.MissingIntegrations()})
	})

	// Public mux: liveness + auth endpoints only. Nothing data-bearing here.
	root := http.NewServeMux()
	root.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	auth.NewHandler(authSvc).Register(root)

	// Everything else under /api/ requires a valid session. More specific
	// patterns above (health, auth/*) win over this catch-all.
	root.Handle("/api/", authSvc.RequireAuth(protected))

	// Serve the embedded frontend for all non-API paths (single-container pod).
	// In local dev this is a placeholder; Vite serves the real UI on :5173.
	root.Handle("/", web.Handler())

	// Cap request bodies at 1 MiB for every route.
	return httpx.Logger(securityHeaders(httpx.CORS(cfg.AllowOrigin, httpx.LimitBody(1<<20, root))))
}

// securityHeaders sets conservative security-related response headers.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

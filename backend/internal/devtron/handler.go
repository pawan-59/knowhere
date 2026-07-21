package devtron

import (
	"net/http"
	"time"

	"central-devtron/internal/httpx"
)

type Handler struct {
	client *Client
}

func NewHandler(client *Client) *Handler { return &Handler{client: client} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/devtron/summary", h.summary)
	mux.HandleFunc("GET /api/devtron/deployments", h.deployments)
	mux.HandleFunc("GET /api/devtron/version", h.version)
}

func (h *Handler) guard(w http.ResponseWriter) bool {
	if !h.client.Configured() {
		httpx.Error(w, http.StatusServiceUnavailable, "devtron not configured",
			"set DEVTRON_BASE_URL and DEVTRON_API_TOKEN")
		return false
	}
	return true
}

func (h *Handler) version(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w) {
		return
	}
	v, err := h.client.ServerVersion(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "failed to load devtron version", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, v)
}

func (h *Handler) deployments(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w) {
		return
	}
	deps, err := h.client.Deployments(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "failed to load deployments", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"count": len(deps), "deployments": deps})
}

// summary combines version + deployment health counts into one payload.
func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w) {
		return
	}
	ctx := r.Context()

	deps, err := h.client.Deployments(ctx)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "failed to load deployments", err.Error())
		return
	}
	byStatus := map[string]int{}
	for _, d := range deps {
		byStatus[d.Status]++
	}

	// Version is best-effort; a failure here shouldn't blank the whole card.
	var version *Version
	if v, err := h.client.ServerVersion(ctx); err == nil {
		version = v
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"version":         version,
		"totalDeployments": len(deps),
		"byStatus":        byStatus,
		"healthy":         byStatus["Healthy"],
		"degraded":        byStatus["Degraded"],
		"progressing":     byStatus["Progressing"],
		"generatedAt":     time.Now().UTC(),
	})
}

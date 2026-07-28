package zoho

import (
	"net/http"
	"strconv"
	"time"

	"knowhere/internal/httpx"
)

// Handler exposes Zoho Desk ticket data over HTTP.
type Handler struct {
	client *Client
}

func NewHandler(client *Client) *Handler { return &Handler{client: client} }

// Register wires the zoho routes onto the mux under /api/zoho.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/zoho/summary", h.summary)
	mux.HandleFunc("GET /api/zoho/tickets", h.tickets)
}

func (h *Handler) guard(w http.ResponseWriter) bool {
	if !h.client.Configured() {
		httpx.Error(w, http.StatusServiceUnavailable, "zoho not configured",
			"set ZOHO_CLIENT_ID, ZOHO_CLIENT_SECRET, ZOHO_REFRESH_TOKEN, ZOHO_ORG_ID")
		return false
	}
	return true
}

// summary returns ticket counts by status plus a few headline metrics.
func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w) {
		return
	}
	counts, err := h.client.CountByStatus(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "failed to load ticket counts", err.Error())
		return
	}

	total := 0
	for _, c := range counts {
		total += c
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"total":         total,
		"byStatus":      counts,
		"open":          counts["Open"],
		"onHold":        counts["On Hold"],
		"escalated":     counts["Escalated"],
		"generatedAt":   time.Now().UTC(),
	})
}

// tickets returns recent tickets, optionally filtered by ?status= and ?limit=.
func (h *Handler) tickets(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w) {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	status := r.URL.Query().Get("status")

	tickets, err := h.client.ListTickets(r.Context(), limit, status)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "failed to load tickets", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"count":   len(tickets),
		"tickets": tickets,
	})
}

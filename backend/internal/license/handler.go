package license

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"central-devtron/internal/httpx"
)

type Handler struct{ store *Store }

func NewHandler(store *Store) *Handler { return &Handler{store: store} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/licenses/summary", h.summary)
	mux.HandleFunc("GET /api/licenses", h.list)
	mux.HandleFunc("POST /api/licenses", h.upsert)
	mux.HandleFunc("GET /api/licenses/{id}", h.get)
	mux.HandleFunc("DELETE /api/licenses/{id}", h.delete)
}

// licenseView wraps License with computed fields for the response.
type licenseView struct {
	License
	DaysToExpiry *int `json:"daysToExpiry"`
}

func view(l License) licenseView { return licenseView{License: l, DaysToExpiry: l.DaysToExpiry()} }

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	s, err := h.store.Summary(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load summary", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, s)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list licenses", err.Error())
		return
	}
	views := make([]licenseView, len(items))
	for i, l := range items {
		views[i] = view(l)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"count": len(views), "licenses": views})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	l, err := h.store.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "license not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load license", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, view(l))
}

func (h *Handler) upsert(w http.ResponseWriter, r *http.Request) {
	var l License
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body", err.Error())
		return
	}
	if l.Customer == "" || l.Installation == "" {
		httpx.Error(w, http.StatusBadRequest, "customer and installation are required")
		return
	}
	if l.Edition == "" {
		l.Edition = "enterprise"
	}
	if l.Status == "" {
		l.Status = "active"
	}
	saved, err := h.store.Upsert(r.Context(), l)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to save license", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, view(saved))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.Delete(r.Context(), id); errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "license not found")
		return
	} else if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to delete license", err.Error())
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

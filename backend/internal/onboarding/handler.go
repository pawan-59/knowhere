package onboarding

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
	mux.HandleFunc("GET /api/onboarding/summary", h.summary)
	mux.HandleFunc("GET /api/onboarding", h.list)
	mux.HandleFunc("POST /api/onboarding", h.upsert)
	mux.HandleFunc("GET /api/onboarding/{id}", h.get)
	mux.HandleFunc("DELETE /api/onboarding/{id}", h.delete)
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	s, err := h.store.Summary(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load summary", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, s)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := h.store.List(r.Context(), q.Get("stage"), q.Get("status"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list onboardings", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"count": len(items), "onboardings": items})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	o, err := h.store.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "onboarding not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load onboarding", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}

func (h *Handler) upsert(w http.ResponseWriter, r *http.Request) {
	var o Onboarding
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body", err.Error())
		return
	}
	if o.Customer == "" {
		httpx.Error(w, http.StatusBadRequest, "customer is required")
		return
	}
	if o.Stage == "" {
		o.Stage = "Discovery Call"
	}
	if o.Status == "" {
		o.Status = "on_track"
	}
	if o.Progress < 0 {
		o.Progress = 0
	} else if o.Progress > 100 {
		o.Progress = 100
	}
	saved, err := h.store.Upsert(r.Context(), o)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to save onboarding", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, saved)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.Delete(r.Context(), id); errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "onboarding not found")
		return
	} else if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to delete onboarding", err.Error())
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

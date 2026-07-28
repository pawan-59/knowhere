package onboarding

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"knowhere/internal/httpx"
)

type Handler struct{ store *Store }

func NewHandler(store *Store) *Handler { return &Handler{store: store} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/onboarding/summary", h.summary)
	mux.HandleFunc("GET /api/onboarding", h.list)
	mux.HandleFunc("POST /api/onboarding", h.upsert)
	mux.HandleFunc("GET /api/onboarding/{code}", h.get)
	mux.HandleFunc("DELETE /api/onboarding/{code}", h.delete)
	mux.HandleFunc("GET /api/onboarding/{code}/logs", h.listLogs)
	mux.HandleFunc("POST /api/onboarding/{code}/logs", h.addLog)
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
	status := q.Get("status")
	if status != "" && !ValidStatus(status) {
		httpx.Error(w, http.StatusBadRequest, "invalid status")
		return
	}
	limit := 0
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			httpx.Error(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = n
	}
	items, err := h.store.List(r.Context(), status, limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list onboardings", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"count": len(items), "onboardings": items})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	o, err := h.store.GetByShortCode(r.Context(), code)
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
	if o.Company == "" {
		httpx.Error(w, http.StatusBadRequest, "company is required")
		return
	}
	if o.Status == "" {
		o.Status = "in_progress"
	}
	if !ValidStatus(o.Status) {
		httpx.Error(w, http.StatusBadRequest, "invalid status")
		return
	}
	if o.Phase == "" {
		o.Phase = "Discovery Call"
	}
	if !ValidPhase(o.Phase) {
		httpx.Error(w, http.StatusBadRequest, "invalid phase")
		return
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

func (h *Handler) listLogs(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	logs, err := h.store.ListLogs(r.Context(), code)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "onboarding not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list logs", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"logs": logs})
}

func (h *Handler) addLog(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	var body struct {
		ContactDate   string `json:"contactDate"`
		ContactType   string `json:"contactType"`
		ReachedBy     string `json:"reachedBy"`
		ContactPerson string `json:"contactPerson"`
		Description   string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body", err.Error())
		return
	}
	if strings.TrimSpace(body.Description) == "" {
		httpx.Error(w, http.StatusBadRequest, "description is required")
		return
	}
	if body.ContactType == "" {
		body.ContactType = "call"
	}
	if !ValidContactType(body.ContactType) {
		httpx.Error(w, http.StatusBadRequest, "invalid contact type")
		return
	}
	contactDate := time.Now().UTC()
	if body.ContactDate != "" {
		t, err := time.Parse(time.RFC3339, body.ContactDate)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid contactDate")
			return
		}
		contactDate = t
	}
	l := Log{ContactDate: contactDate, ContactType: body.ContactType, Description: strings.TrimSpace(body.Description)}
	if body.ReachedBy != "" {
		l.ReachedBy = &body.ReachedBy
	}
	if body.ContactPerson != "" {
		l.ContactPerson = &body.ContactPerson
	}
	saved, err := h.store.AddLog(r.Context(), code, l)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "onboarding not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to add log", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, saved)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if err := h.store.DeleteByShortCode(r.Context(), code); errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "onboarding not found")
		return
	} else if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to delete onboarding", err.Error())
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

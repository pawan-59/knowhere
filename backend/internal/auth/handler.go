package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"knowhere/internal/httpx"
)

// dummyHash is a valid bcrypt hash compared against when a user is not found,
// so login timing does not reveal whether an email exists (no enumeration).
var dummyHash = []byte("$2a$10$CwTycUXWue0Thq9StjUM0uJ8m9d8oJt9r2fJ8fF3q7d7d0oJt9r2")

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("POST /api/auth/logout", h.logout)
	mux.HandleFunc("GET /api/auth/me", h.me)
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	// Rate-limit brute force by client IP.
	if !h.svc.limiter.allow(clientIP(r)) {
		httpx.Error(w, http.StatusTooManyRequests, "too many attempts, try again later")
		return
	}

	var req loginReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}

	u, err := h.svc.store.byEmail(r.Context(), req.Email)
	if err != nil {
		// Compare against a dummy hash to keep timing constant, then fail.
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
		httpx.Error(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if u.Disabled || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := sign(h.svc.secret, claims{
		Sub:   u.ID,
		Email: u.Email,
		Role:  u.Role,
		Exp:   time.Now().Add(h.svc.ttl).Unix(),
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not create session")
		return
	}
	h.svc.setSessionCookie(w, token)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{"id": u.ID, "email": u.Email, "name": u.Name, "role": u.Role},
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	h.svc.clearSessionCookie(w)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// me returns the current user, or 401 if not authenticated.
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	c, ok := h.svc.readSession(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	u, err := h.svc.store.byID(r.Context(), c.Sub)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{"id": u.ID, "email": u.Email, "name": u.Name, "role": u.Role},
	})
}

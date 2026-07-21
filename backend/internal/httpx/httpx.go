// Package httpx provides small HTTP helpers shared by all module handlers.
package httpx

import (
	"encoding/json"
	"log"
	"net/http"
)

// JSON writes v as an application/json response with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("httpx: encode response: %v", err)
	}
}

// ErrorBody is the standard error envelope returned to the frontend.
type ErrorBody struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// Error writes a JSON error response.
func Error(w http.ResponseWriter, status int, msg string, details ...string) {
	body := ErrorBody{Error: msg}
	if len(details) > 0 {
		body.Details = details[0]
	}
	JSON(w, status, body)
}

// CORS restricts cross-origin access to exactly the configured origin and
// permits credentials (cookies). It never emits a wildcard origin, which is
// required for credentialed requests and avoids opening the API to any site.
func CORS(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqOrigin := r.Header.Get("Origin")
		if reqOrigin != "" && reqOrigin == origin {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Content-Type")
			h.Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// LimitBody caps request body size to guard against large-payload DoS.
func LimitBody(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		}
		next.ServeHTTP(w, r)
	})
}

// Logger logs each request method, path, and duration.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

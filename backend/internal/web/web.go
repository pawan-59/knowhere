// Package web serves the built React frontend from the Go binary so the whole
// app ships as a single container. The Docker build copies frontend/dist into
// ./dist before compiling; for local `go build` a placeholder index.html keeps
// the embed valid (in local dev the frontend is served by Vite on :5173).
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

//go:embed all:dist
var embedded embed.FS

// Handler returns an http.Handler that serves the embedded SPA. Unknown paths
// fall back to index.html so client-side routing (BrowserRouter) works on deep
// links and refreshes.
func Handler() http.Handler {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic("web: cannot open embedded dist: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the requested asset; if it doesn't exist, serve the SPA
		// shell so the client router can handle the route.
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(sub, p); err != nil {
			if os.IsNotExist(err) {
				r = r.Clone(r.Context())
				r.URL.Path = "/"
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}

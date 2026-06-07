package api

import (
	"io/fs"
	"net/http"
	"strings"
)

// staticHandler serves embedded frontend assets with SPA fallback.
// Requests for paths that don't match an embedded file fall back to
// index.html so client-side routing works. Paths under /api/ and /ws/
// are left to the surrounding router (they will not reach this handler
// when mounted on a sub-router).
//
// Note: we use http.ServeFileFS rather than http.FileServer+http.FS
// because the latter canonicalizes /index.html -> /, which would loop
// indefinitely when paired with the SPA fallback below.
func staticHandler(fsys fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip leading slash; embed.FS uses "index.html" not "/index.html".
		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean == "" {
			clean = "index.html"
		}
		if _, err := fs.Stat(fsys, clean); err != nil {
			// SPA fallback: serve index.html for unknown paths.
			clean = "index.html"
		}
		http.ServeFileFS(w, r, fsys, clean)
	})
}

package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static/*
var staticFS embed.FS

func Handler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	index, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" || path == "/search" || path == "/adoption" || path == "/upload" || path == "/reviews" ||
			strings.HasPrefix(path, "/packages/") || strings.HasPrefix(path, "/assets/") {
			if strings.HasPrefix(path, "/assets/") {
				files.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(index)
			return
		}
		http.NotFound(w, r)
	})
}

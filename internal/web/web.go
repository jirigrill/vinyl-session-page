// Package web holds the HTTP handlers.
package web

import (
	"crypto/subtle"
	"embed"
	"html/template"
	"net/http"
)

//go:embed templates/*.html
var templates embed.FS

var pages = template.Must(template.ParseFS(templates, "templates/*.html"))

// HardcodedVideoID is the single video served by the walking skeleton.
const HardcodedVideoID = "HBNd3VZS3jA"

// NewHandler returns the app's routes wrapped in basic auth.
func NewHandler(user, password string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		pages.ExecuteTemplate(w, "index.html", map[string]string{"VideoID": HardcodedVideoID})
	})
	return basicAuth(user, password, mux)
}

func basicAuth(user, password string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		userOK := subtle.ConstantTimeCompare([]byte(u), []byte(user)) == 1
		passOK := subtle.ConstantTimeCompare([]byte(p), []byte(password)) == 1
		if !ok || !userOK || !passOK {
			w.Header().Set("WWW-Authenticate", `Basic realm="vinyl", charset="UTF-8"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

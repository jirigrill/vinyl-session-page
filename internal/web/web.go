// Package web holds the HTTP handlers.
package web

import (
	"crypto/subtle"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

//go:embed templates/*.html
var templates embed.FS

var pages = template.Must(template.New("").Funcs(template.FuncMap{"duration": formatDuration}).ParseFS(templates, "templates/*.html"))

// Video is a catalog row as rendered on the pages.
type Video struct {
	ID, Title, ChannelTitle, ThumbnailURL string
	Duration                              time.Duration
}

// NewHandler returns the app's routes wrapped in basic auth.
func NewHandler(conn *sql.DB, user, password string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		videos, err := listVideos(conn)
		if err != nil {
			serverError(w, err)
			return
		}
		render(w, "index.html", videos)
	})
	mux.HandleFunc("GET /watch/{id}", func(w http.ResponseWriter, r *http.Request) {
		v, err := getVideo(conn, r.PathValue("id"))
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}
		render(w, "watch.html", v)
	})
	return basicAuth(user, password, mux)
}

const videoColumns = "id, title, channel_title, thumbnail_url, duration_seconds"

func scanVideo(row interface{ Scan(...any) error }) (Video, error) {
	var v Video
	var secs int64
	err := row.Scan(&v.ID, &v.Title, &v.ChannelTitle, &v.ThumbnailURL, &secs)
	v.Duration = time.Duration(secs) * time.Second
	return v, err
}

func listVideos(conn *sql.DB) ([]Video, error) {
	rows, err := conn.Query("SELECT " + videoColumns + " FROM videos ORDER BY published_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Video
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func getVideo(conn *sql.DB, id string) (Video, error) {
	return scanVideo(conn.QueryRow("SELECT "+videoColumns+" FROM videos WHERE id = ?", id))
}

func render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
	}
}

func serverError(w http.ResponseWriter, err error) {
	log.Print(err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

// formatDuration renders 1h02m03s as "1:02:03" and 58m as "58:00".
func formatDuration(d time.Duration) string {
	s := int(d.Seconds())
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
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

// Command vinyl runs the vinyl session web server.
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jirigrill/vinyl-session-page/internal/db"
	"github.com/jirigrill/vinyl-session-page/internal/web"
)

func main() {
	addr := getenv("VINYL_ADDR", "127.0.0.1:8080")
	dataDir := getenv("VINYL_DATA_DIR", ".")
	user := os.Getenv("VINYL_USER")
	password := os.Getenv("VINYL_PASSWORD")
	if user == "" || password == "" {
		log.Fatal("VINYL_USER and VINYL_PASSWORD must be set")
	}

	conn, err := db.Open(filepath.Join(dataDir, "vinyl.db"))
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer conn.Close()

	srv := &http.Server{
		Addr:              addr,
		Handler:           web.NewHandler(user, password),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

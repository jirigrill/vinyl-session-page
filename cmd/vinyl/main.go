// Command vinyl runs the vinyl session web server.
//
//	vinyl          serve the web UI
//	vinyl ingest   one-shot import of the seed channels (needs YT_DATA_API_KEY)
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jirigrill/vinyl-session-page/internal/db"
	"github.com/jirigrill/vinyl-session-page/internal/ingest"
	"github.com/jirigrill/vinyl-session-page/internal/web"
	"github.com/jirigrill/vinyl-session-page/internal/youtube"
)

func main() {
	dataDir := getenv("VINYL_DATA_DIR", ".")
	if len(os.Args) > 1 && os.Args[1] == "ingest" {
		runIngest(dataDir)
		return
	}

	addr := getenv("VINYL_ADDR", "127.0.0.1:8080")
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
		Handler:           web.NewHandler(conn, user, password),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func runIngest(dataDir string) {
	key := os.Getenv("YT_DATA_API_KEY")
	if key == "" {
		log.Fatal("YT_DATA_API_KEY must be set")
	}
	seed, err := os.ReadFile(getenv("VINYL_SEED_FILE", "data/seed-channels.txt"))
	if err != nil {
		log.Fatalf("read seed file: %v", err)
	}
	conn, err := db.Open(filepath.Join(dataDir, "vinyl.db"))
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer conn.Close()

	st, err := ingest.Run(context.Background(), conn, youtube.NewClient(key), ingest.HandlesFromSeed(string(seed)))
	if err != nil {
		log.Fatalf("ingest: %v", err)
	}
	log.Printf("fetched %d videos, stored %d sessions, skipped %d non-sessions", st.Fetched, st.Stored, st.Skipped)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

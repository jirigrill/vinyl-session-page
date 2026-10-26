package ingest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jirigrill/vinyl-session-page/internal/db"
	"github.com/jirigrill/vinyl-session-page/internal/youtube"
)

// recordedAPI serves the recorded YouTube responses in ../youtube/testdata.
// Pass a videos file to simulate changed metadata between runs.
func recordedAPI(t *testing.T, videosFile *string) *youtube.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file := map[string]string{
			"/channels":      "channels.json",
			"/playlistItems": "playlist_page1.json",
			"/videos":        *videosFile,
		}[r.URL.Path]
		if r.URL.Path == "/playlistItems" && r.URL.Query().Get("pageToken") != "" {
			file = "playlist_page2.json"
		}
		body, err := os.ReadFile(filepath.Join("..", "youtube", "testdata", file))
		if err != nil {
			t.Fatal(err)
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return &youtube.Client{BaseURL: srv.URL, Key: "k", HTTP: srv.Client()}
}

func TestRunStoresSessionsIdempotently(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	videos := "videos.json"
	src := recordedAPI(t, &videos)
	ctx := context.Background()

	st, err := Run(ctx, conn, src, []string{"@CrateGuest"})
	if err != nil {
		t.Fatal(err)
	}
	if st != (Stats{Fetched: 4, Stored: 2, Skipped: 2}) {
		t.Fatalf("stats = %+v", st)
	}

	// Second run with an edited title: no duplicates, metadata updated.
	videos = "../../ingest/testdata/videos_edited.json"
	if _, err := Run(ctx, conn, src, []string{"@CrateGuest"}); err != nil {
		t.Fatal(err)
	}
	var n int
	conn.QueryRow("SELECT COUNT(*) FROM videos").Scan(&n)
	if n != 2 {
		t.Fatalf("rows = %d, want 2", n)
	}
	var title string
	var secs int
	conn.QueryRow("SELECT title, duration_seconds FROM videos WHERE id = 'vidSession1'").Scan(&title, &secs)
	if title != "Sunday Morning Jazz Vinyl Session (Remastered)" || secs != 3723 {
		t.Fatalf("got %q %d", title, secs)
	}
}

func TestIsSession(t *testing.T) {
	long := time.Hour
	cases := []struct {
		name string
		v    youtube.Video
		want bool
	}{
		{"full session", youtube.Video{Title: "Jazz Session", Duration: long, LiveContent: "none"}, true},
		{"too short", youtube.Video{Title: "Clip", Duration: 9 * time.Minute}, false},
		{"shorts tag", youtube.Video{Title: "Drop #Shorts", Duration: long}, false},
		{"trailer", youtube.Video{Title: "Season 2 TRAILER", Duration: long}, false},
		{"upcoming premiere", youtube.Video{Title: "Live set", Duration: long, LiveContent: "upcoming"}, false},
	}
	for _, tc := range cases {
		if got := IsSession(tc.v); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestHandlesFromSeed(t *testing.T) {
	got := HandlesFromSeed("https://www.youtube.com/@REANALOG\n\n# comment\nhttps://www.youtube.com/@CrateGuest/\n@bare\n")
	want := []string{"@REANALOG", "@CrateGuest", "@bare"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v", got)
		}
	}
}

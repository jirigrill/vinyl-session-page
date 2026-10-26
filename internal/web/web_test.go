package web

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jirigrill/vinyl-session-page/internal/db"
)

func TestIndexRequiresAuth(t *testing.T) {
	h := NewHandler(testDB(t), "u", "p")
	cases := []struct {
		name       string
		user, pass string
		setAuth    bool
		want       int
	}{
		{"no credentials", "", "", false, http.StatusUnauthorized},
		{"wrong password", "u", "x", true, http.StatusUnauthorized},
		{"wrong user", "x", "p", true, http.StatusUnauthorized},
		{"correct", "u", "p", true, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tc.setAuth {
				req.SetBasicAuth(tc.user, tc.pass)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	_, err = conn.Exec(`INSERT INTO videos VALUES
		('old1', 'Old Set', 'UC1', 'Crate Guest', 3600, 'https://i.ytimg.com/vi/old1/hqdefault.jpg', '2025-01-01T00:00:00Z', '', '2026-01-01T00:00:00Z'),
		('new1', 'New <Set>', 'UC2', 'REANALOG', 3723, 'https://i.ytimg.com/vi/new1/hqdefault.jpg', '2026-01-01T00:00:00Z', '', '2026-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	req.SetBasicAuth("u", "p")
	rec := httptest.NewRecorder()
	NewHandler(testDB(t), "u", "p").ServeHTTP(rec, req)
	return rec
}

func TestIndexListsCatalogNewestFirst(t *testing.T) {
	body := get(t, "/").Body.String()
	for _, want := range []string{`href="/watch/new1"`, "New &lt;Set&gt;", "REANALOG", "1:02:03", "https://i.ytimg.com/vi/old1/hqdefault.jpg", "1:00:00"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
	if strings.Index(body, "/watch/new1") > strings.Index(body, "/watch/old1") {
		t.Error("want newest first")
	}
}

func TestWatchEmbedsVideoWithFallback(t *testing.T) {
	rec := get(t, "/watch/new1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	for _, want := range []string{"https://www.youtube.com/embed/new1", "https://www.youtube.com/watch?v=new1"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("body missing %q", want)
		}
	}
}

func TestWatchUnknownVideoIs404(t *testing.T) {
	if rec := get(t, "/watch/nope"); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestUnknownPathIs404(t *testing.T) {
	if rec := get(t, "/nope"); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

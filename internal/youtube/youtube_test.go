package youtube

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// fixtureServer replays recorded API responses from testdata/ and counts calls.
func fixtureServer(t *testing.T) (*Client, *int) {
	t.Helper()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		q := r.URL.Query()
		if q.Get("key") != "test-key" {
			http.Error(w, "missing key", http.StatusForbidden)
			return
		}
		var file string
		switch {
		case r.URL.Path == "/channels" && q.Get("forHandle") == "@CrateGuest":
			file = "channels.json"
		case r.URL.Path == "/playlistItems" && q.Get("pageToken") == "":
			file = "playlist_page1.json"
		case r.URL.Path == "/playlistItems" && q.Get("pageToken") == "EAAaBlBUOkNBTQ":
			file = "playlist_page2.json"
		case r.URL.Path == "/videos":
			file = "videos.json"
		case r.URL.Path == "/channels":
			w.Write([]byte(`{"items":[]}`))
			return
		default:
			http.NotFound(w, r)
			return
		}
		body, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return &Client{BaseURL: srv.URL, Key: "test-key", HTTP: srv.Client()}, &calls
}

func TestChannelByHandle(t *testing.T) {
	c, _ := fixtureServer(t)
	ch, err := c.ChannelByHandle(context.Background(), "@CrateGuest")
	if err != nil {
		t.Fatal(err)
	}
	if ch.ID != "UCcrateguest0000000000" || ch.Title != "Crate Guest" || ch.UploadsPlaylistID != "UUcrateguest0000000000" {
		t.Fatalf("got %+v", ch)
	}
	if _, err := c.ChannelByHandle(context.Background(), "@nobody"); err == nil {
		t.Fatal("want error for unknown handle")
	}
}

func TestPlaylistVideoIDsFollowsPagination(t *testing.T) {
	c, calls := fixtureServer(t)
	ids, err := c.PlaylistVideoIDs(context.Background(), "UUcrateguest0000000000")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"vidSession1", "vidShort001", "vidTrailer1", "vidSession2"}
	if len(ids) != len(want) {
		t.Fatalf("ids = %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("ids = %v, want %v", ids, want)
		}
	}
	if *calls != 2 {
		t.Fatalf("calls = %d, want 2", *calls)
	}
}

func TestVideos(t *testing.T) {
	c, _ := fixtureServer(t)
	vs, err := c.Videos(context.Background(), []string{"vidSession1", "vidShort001", "vidTrailer1", "vidSession2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 4 {
		t.Fatalf("len = %d, want 4", len(vs))
	}
	v := vs[0]
	if v.Title != "Sunday Morning Jazz Vinyl Session" || v.ChannelTitle != "Crate Guest" ||
		v.Duration != time.Hour+2*time.Minute+3*time.Second ||
		v.ThumbnailURL != "https://i.ytimg.com/vi/vidSession1/hqdefault.jpg" ||
		v.Description != "All vinyl, one take." ||
		!v.PublishedAt.Equal(time.Date(2026, 9, 1, 18, 0, 0, 0, time.UTC)) {
		t.Fatalf("got %+v", v)
	}
	if vs[3].ThumbnailURL != "https://i.ytimg.com/vi/vidSession2/mqdefault.jpg" {
		t.Fatalf("thumbnail fallback = %q", vs[3].ThumbnailURL)
	}
}

func TestVideosBatchesBy50(t *testing.T) {
	c, calls := fixtureServer(t)
	ids := make([]string, 120)
	for i := range ids {
		ids[i] = "x"
	}
	if _, err := c.Videos(context.Background(), ids); err != nil {
		t.Fatal(err)
	}
	if *calls != 3 {
		t.Fatalf("calls = %d, want 3", *calls)
	}
}

func TestParseDuration(t *testing.T) {
	cases := map[string]time.Duration{
		"PT45S":    45 * time.Second,
		"PT58M":    58 * time.Minute,
		"PT1H2M3S": time.Hour + 2*time.Minute + 3*time.Second,
		"P1DT2H":   26 * time.Hour,
		"P0D":      0,
	}
	for in, want := range cases {
		got, err := ParseDuration(in)
		if err != nil || got != want {
			t.Errorf("ParseDuration(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseDuration("1h"); err == nil {
		t.Error("want error for non-ISO input")
	}
}

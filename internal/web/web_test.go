package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIndexRequiresAuth(t *testing.T) {
	h := NewHandler("u", "p")
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

func TestIndexEmbedsVideoWithFallback(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("u", "p")
	rec := httptest.NewRecorder()
	NewHandler("u", "p").ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{
		"https://www.youtube.com/embed/" + HardcodedVideoID,
		"https://www.youtube.com/watch?v=" + HardcodedVideoID,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
}

func TestUnknownPathIs404(t *testing.T) {
	req := httptest.NewRequest("GET", "/nope", nil)
	req.SetBasicAuth("u", "p")
	rec := httptest.NewRecorder()
	NewHandler("u", "p").ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

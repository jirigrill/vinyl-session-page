package db

import (
	"path/filepath"
	"testing"
)

func TestOpenCreatesAndMigrates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	for i := 0; i < 2; i++ { // second open must be a no-op migration
		conn, err := Open(path)
		if err != nil {
			t.Fatalf("open #%d: %v", i+1, err)
		}
		var version int
		if err := conn.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
			t.Fatal(err)
		}
		if version != 2 {
			t.Fatalf("user_version = %d, want 2", version)
		}
		if _, err := conn.Exec("INSERT OR REPLACE INTO app_meta(key, value) VALUES ('k', 'v')"); err != nil {
			t.Fatalf("app_meta missing: %v", err)
		}
		conn.Close()
	}
}

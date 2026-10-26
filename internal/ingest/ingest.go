// Package ingest pulls seed channel uploads from YouTube into the database.
package ingest

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jirigrill/vinyl-session-page/internal/youtube"
)

// MinSessionLength drops shorts, trailers, teasers and other clips.
const MinSessionLength = 10 * time.Minute

// Source is the slice of the YouTube client the ingest uses.
type Source interface {
	ChannelByHandle(ctx context.Context, handle string) (youtube.Channel, error)
	PlaylistVideoIDs(ctx context.Context, playlistID string) ([]string, error)
	Videos(ctx context.Context, ids []string) ([]youtube.Video, error)
}

// Stats summarises one ingest run.
type Stats struct {
	Fetched, Stored, Skipped int
}

// IsSession reports whether v looks like a full session rather than a clip.
func IsSession(v youtube.Video) bool {
	if v.LiveContent == "live" || v.LiveContent == "upcoming" {
		return false
	}
	if v.Duration < MinSessionLength {
		return false
	}
	t := strings.ToLower(v.Title)
	for _, w := range []string{"#shorts", "trailer", "teaser"} {
		if strings.Contains(t, w) {
			return false
		}
	}
	return true
}

// Run ingests every upload of the given channel handles. Re-running updates
// existing rows instead of duplicating them.
func Run(ctx context.Context, conn *sql.DB, src Source, handles []string) (Stats, error) {
	var st Stats
	now := time.Now().UTC().Format(time.RFC3339)
	for _, h := range handles {
		ch, err := src.ChannelByHandle(ctx, h)
		if err != nil {
			return st, err
		}
		ids, err := src.PlaylistVideoIDs(ctx, ch.UploadsPlaylistID)
		if err != nil {
			return st, fmt.Errorf("%s uploads: %w", h, err)
		}
		videos, err := src.Videos(ctx, ids)
		if err != nil {
			return st, fmt.Errorf("%s videos: %w", h, err)
		}
		st.Fetched += len(videos)
		for _, v := range videos {
			if !IsSession(v) {
				st.Skipped++
				continue
			}
			if err := upsert(ctx, conn, v, now); err != nil {
				return st, err
			}
			st.Stored++
		}
	}
	return st, nil
}

func upsert(ctx context.Context, conn *sql.DB, v youtube.Video, now string) error {
	_, err := conn.ExecContext(ctx, `
		INSERT INTO videos (id, title, channel_id, channel_title, duration_seconds, thumbnail_url, published_at, description, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			channel_id = excluded.channel_id,
			channel_title = excluded.channel_title,
			duration_seconds = excluded.duration_seconds,
			thumbnail_url = excluded.thumbnail_url,
			published_at = excluded.published_at,
			description = excluded.description,
			updated_at = excluded.updated_at`,
		v.ID, v.Title, v.ChannelID, v.ChannelTitle, int(v.Duration.Seconds()), v.ThumbnailURL,
		v.PublishedAt.UTC().Format(time.RFC3339), v.Description, now)
	if err != nil {
		return fmt.Errorf("store %s: %w", v.ID, err)
	}
	return nil
}

// HandlesFromSeed turns lines like "https://www.youtube.com/@REANALOG" into
// "@REANALOG", ignoring blanks and # comments.
func HandlesFromSeed(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.LastIndex(line, "/@"); i >= 0 {
			line = line[i+1:]
		}
		out = append(out, strings.TrimSuffix(line, "/"))
	}
	return out
}

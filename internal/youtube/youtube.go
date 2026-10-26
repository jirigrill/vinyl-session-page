// Package youtube is a minimal YouTube Data API v3 client covering what the
// ingest needs. Quota cost: every call below is 1 unit.
package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://www.googleapis.com/youtube/v3"

type Client struct {
	BaseURL string
	Key     string
	HTTP    *http.Client
}

func NewClient(key string) *Client {
	return &Client{BaseURL: DefaultBaseURL, Key: key, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

// Channel is a resolved channel handle.
type Channel struct {
	ID                string
	Title             string
	UploadsPlaylistID string
}

// Video is the metadata stored per upload.
type Video struct {
	ID           string
	Title        string
	ChannelID    string
	ChannelTitle string
	Duration     time.Duration
	ThumbnailURL string
	PublishedAt  time.Time
	Description  string
	LiveContent  string // "none", "live" or "upcoming"
}

// ChannelByHandle resolves a handle such as "@REANALOG".
func (c *Client) ChannelByHandle(ctx context.Context, handle string) (Channel, error) {
	var resp struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title string `json:"title"`
			} `json:"snippet"`
			ContentDetails struct {
				RelatedPlaylists struct {
					Uploads string `json:"uploads"`
				} `json:"relatedPlaylists"`
			} `json:"contentDetails"`
		} `json:"items"`
	}
	q := url.Values{"part": {"snippet,contentDetails"}, "forHandle": {handle}}
	if err := c.get(ctx, "channels", q, &resp); err != nil {
		return Channel{}, err
	}
	if len(resp.Items) == 0 {
		return Channel{}, fmt.Errorf("channel %s not found", handle)
	}
	it := resp.Items[0]
	return Channel{ID: it.ID, Title: it.Snippet.Title, UploadsPlaylistID: it.ContentDetails.RelatedPlaylists.Uploads}, nil
}

// PlaylistVideoIDs returns every video ID in a playlist, following pagination.
func (c *Client) PlaylistVideoIDs(ctx context.Context, playlistID string) ([]string, error) {
	var ids []string
	pageToken := ""
	for {
		var resp struct {
			NextPageToken string `json:"nextPageToken"`
			Items         []struct {
				ContentDetails struct {
					VideoID string `json:"videoId"`
				} `json:"contentDetails"`
			} `json:"items"`
		}
		q := url.Values{"part": {"contentDetails"}, "playlistId": {playlistID}, "maxResults": {"50"}}
		if pageToken != "" {
			q.Set("pageToken", pageToken)
		}
		if err := c.get(ctx, "playlistItems", q, &resp); err != nil {
			return nil, err
		}
		for _, it := range resp.Items {
			ids = append(ids, it.ContentDetails.VideoID)
		}
		if resp.NextPageToken == "" {
			return ids, nil
		}
		pageToken = resp.NextPageToken
	}
}

// Videos fetches metadata for the given IDs in batches of 50.
func (c *Client) Videos(ctx context.Context, ids []string) ([]Video, error) {
	var out []Video
	for start := 0; start < len(ids); start += 50 {
		end := min(start+50, len(ids))
		var resp struct {
			Items []struct {
				ID      string `json:"id"`
				Snippet struct {
					Title                string               `json:"title"`
					Description          string               `json:"description"`
					ChannelID            string               `json:"channelId"`
					ChannelTitle         string               `json:"channelTitle"`
					PublishedAt          time.Time            `json:"publishedAt"`
					LiveBroadcastContent string               `json:"liveBroadcastContent"`
					Thumbnails           map[string]thumbnail `json:"thumbnails"`
				} `json:"snippet"`
				ContentDetails struct {
					Duration string `json:"duration"`
				} `json:"contentDetails"`
			} `json:"items"`
		}
		q := url.Values{"part": {"snippet,contentDetails"}, "id": {strings.Join(ids[start:end], ",")}, "maxResults": {"50"}}
		if err := c.get(ctx, "videos", q, &resp); err != nil {
			return nil, err
		}
		for _, it := range resp.Items {
			d, err := ParseDuration(it.ContentDetails.Duration)
			if err != nil {
				return nil, fmt.Errorf("video %s: %w", it.ID, err)
			}
			out = append(out, Video{
				ID:           it.ID,
				Title:        it.Snippet.Title,
				ChannelID:    it.Snippet.ChannelID,
				ChannelTitle: it.Snippet.ChannelTitle,
				Duration:     d,
				ThumbnailURL: bestThumbnail(it.Snippet.Thumbnails),
				PublishedAt:  it.Snippet.PublishedAt,
				Description:  it.Snippet.Description,
				LiveContent:  it.Snippet.LiveBroadcastContent,
			})
		}
	}
	return out, nil
}

type thumbnail struct {
	URL string `json:"url"`
}

// bestThumbnail picks "high" (480x360, always present) over larger sizes to
// keep the card grid light on a phone.
func bestThumbnail(thumbs map[string]thumbnail) string {
	for _, size := range []string{"high", "medium", "standard", "maxres", "default"} {
		if t, ok := thumbs[size]; ok {
			return t.URL
		}
	}
	return ""
}

var durationRe = regexp.MustCompile(`^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)

// ParseDuration parses the ISO 8601 durations YouTube returns, e.g. "PT1H2M3S".
func ParseDuration(s string) (time.Duration, error) {
	m := durationRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("bad duration %q", s)
	}
	units := []time.Duration{24 * time.Hour, time.Hour, time.Minute, time.Second}
	var d time.Duration
	for i, u := range units {
		if m[i+1] != "" {
			n, _ := strconv.Atoi(m[i+1])
			d += time.Duration(n) * u
		}
	}
	return d, nil
}

func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	q.Set("key", c.Key)
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/"+path+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("youtube %s: request failed", path) // the wrapped error would leak the key in the URL
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("youtube %s: %s", path, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

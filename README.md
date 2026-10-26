# vinyl-session-page

Aggregator of vinyl/DJ session videos with rich filtering (mood, scene, genre, format). Embeds official players only; no media is hosted.

Status: planning. See [REQUIREMENTS.md](REQUIREMENTS.md).

## Ingest

One-shot import of the channels in `data/seed-channels.txt`:

```sh
set -a; . ./.env; set +a     # YT_DATA_API_KEY
go run ./cmd/vinyl ingest     # writes ./vinyl.db (VINYL_DATA_DIR to override)
```

Re-running is safe: existing videos are updated, not duplicated. Quota cost is
about 1 unit per channel plus 2 per 50 uploads (daily limit is 10,000).

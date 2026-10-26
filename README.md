# vinyl-session-page

Aggregator of vinyl/DJ session videos with rich filtering (mood, scene, genre, format). Embeds official players only; no media is hosted.

Status: MVP in progress; seed catalog ingested and browsable. See [REQUIREMENTS.md](REQUIREMENTS.md).

## Data

Production on the droplet holds the only real database. Ingest runs only
there, because the YouTube API key is IP-restricted to the droplet. Local
development uses a disposable snapshot of it.

## Local development

Manual for now; automating it with a `Makefile` is tracked in #16.

```sh
scp root@142.93.99.205:'/var/lib/vinyl/vinyl.db*' .   # snapshot of prod catalog
VINYL_USER=dev VINYL_PASSWORD=dev go run ./cmd/vinyl  # http://127.0.0.1:8080
go test ./...
```

Copy all three `vinyl.db*` files: the database uses SQLite WAL mode, so the
main file alone can miss recent writes. Don't commit them.

## Ingest

One-shot import of the channels in `data/seed-channels.txt`. Run it on the
droplet (the API key is rejected elsewhere):

```sh
scp data/seed-channels.txt root@142.93.99.205:/tmp/
ssh root@142.93.99.205 'cd /tmp && runuser -u vinyl -- env YT_DATA_API_KEY=... \
  VINYL_DATA_DIR=/var/lib/vinyl VINYL_SEED_FILE=/tmp/seed-channels.txt /opt/vinyl/vinyl ingest'
```

Re-running is safe: existing videos are updated, not duplicated. Quota cost is
about 1 unit per channel plus 2 per 50 uploads (daily limit is 10,000).

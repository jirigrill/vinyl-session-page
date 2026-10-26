# vinyl-session-page

Aggregator of vinyl/DJ session videos with rich filtering (mood, scene, genre, format). Embeds official players only; no media is hosted.

Status: MVP in progress; seed catalog ingested and browsable. See [REQUIREMENTS.md](REQUIREMENTS.md).

## Data

Production on the droplet holds the only real database. Ingest runs only
there, because the YouTube API key is IP-restricted to the droplet. Local
development uses a disposable snapshot of it.

## Local development

```sh
cp .env.example .env   # fill in values; VINYL_USER/VINYL_PASSWORD default to dev/dev
make pull-db           # snapshot prod catalog into data/local/
make run               # http://127.0.0.1:8080
make test vet fmt
```

`make` lists all targets. The droplet is `HOST` in the `Makefile`; override
with `make deploy HOST=root@other`. The snapshot copies all three `vinyl.db*`
files (SQLite WAL mode) and is git-ignored.

## Ingest

One-shot import of the channels in `data/seed-channels.txt`. It runs on the
droplet (the API key is rejected elsewhere), using `YT_DATA_API_KEY` from `.env`:

```sh
make ingest-remote
```

Re-running is safe: existing videos are updated, not duplicated. Quota cost is
about 1 unit per channel plus 2 per 50 uploads (daily limit is 10,000).

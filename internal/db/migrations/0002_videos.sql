CREATE TABLE videos (
    id               TEXT PRIMARY KEY,
    title            TEXT NOT NULL,
    channel_id       TEXT NOT NULL,
    channel_title    TEXT NOT NULL,
    duration_seconds INTEGER NOT NULL,
    thumbnail_url    TEXT NOT NULL,
    published_at     TEXT NOT NULL,
    description      TEXT NOT NULL,
    updated_at       TEXT NOT NULL
);

CREATE INDEX videos_published_at ON videos(published_at);

# Vinyl Session Page: Requirements (v0.2)

## Vision
Personal aggregator of vinyl/DJ session videos with filtering by mood, genre, scene/setting and duration. Pick a session on the iPhone, it plays on the Android TV. No media hosted; official YouTube embeds only.

## Success criterion (MVP done when)
"I pick a mood on my phone and a session plays on my TV."

## Constraints
- Official YouTube IFrame player only, unmodified. No overlays on the player, no ad blocking, no audio-only extraction, no restreaming, no paywall on content.
- Ads: Premium users are ad-free only via their own YouTube session. Not controllable, not promised.
- No video/media storage. Only metadata (and thumbnails by URL).
- Budget: only the existing DigitalOcean droplet.
- English-only UI.
- Single user (the owner) in MVP. Public repo, but no public launch yet.
- Czech Republic, personal project. No payments in MVP, so no legal entity needed yet.

## Stack
Go + SQLite + HTMX. Server-rendered, single binary, systemd on droplet. Owner has no Go experience (learning goal); Python avoided.

## MVP scope

### 1. Ingest (one-shot, manual run)
- [x] Seed channels from `data/seed-channels.txt` (REANALOG, Chilled_Sundays, breakfasttilt, CrateGuest, incrediblecoffee2020)
- [x] Resolve handle to channel and uploads playlist via YouTube Data API v3 (key in `.env`, never committed)
- [x] Store per video: id, title, channel, duration, thumbnail URL, published date, description
- [x] Filter out non-sessions (shorts, trailers, very short videos) with simple rules
- [x] Idempotent: re-running does not duplicate
- [ ] Automated refresh is NOT in MVP (later)

### 2. AI tagging (fully automatic, no manual tagging)
- [ ] Pass 1: AI proposes tag taxonomy from the seed catalog
- [ ] Pass 2: AI tags each video using title, description, channel, duration
- [ ] Tag dimensions: **mood/time of day** (morning breakfast, evening dinner, others as AI finds), **genre**, **scene/setting**, **duration** (derived)
- [ ] Tags stored in SQLite with taxonomy version, so re-tagging is possible when prompts improve
- [ ] No manual tag editing UI. Improvement path is better prompts/models, not manual work.

### 3. Browse and filter (phone-first)
- [ ] Filters: mood, genre, scene/setting, duration range
- [ ] Combinable, shareable URL state
- [ ] Video cards: thumbnail, title, channel, duration, tags
- [ ] Responsive, mobile-first

### 4. Playback: select and play
- [ ] Select one video and play it. No queue, autoplay-next, shuffle or playlists in MVP.
- [ ] Embedded YouTube player page
- [ ] "Open on YouTube" fallback

### 5. TV remote (phone as remote)
- [ ] TV opens a "player" page showing a pairing code/QR
- [ ] iPhone enters/scans the code and pairs
- [ ] Choosing a video on the phone makes the TV page load and play it (WebSocket)
- [ ] Basic controls from phone: play/pause, stop (nice to have)
- [x] Test early: YouTube embed playback on the TV (passed, ticket #1)
- [ ] Browsing with the TV's own remote is deferred (wanted later)

### 6. Hosting
- [ ] Deploy on existing DigitalOcean droplet behind existing reverse proxy, HTTPS
- [ ] Basic auth on phone UI; TV uses expiring single-use pairing code
- [ ] Backup = SQLite file copy

## Out of MVP (later)
- Automated refresh of new uploads
- Better AI tagging quality, re-tagging pipeline, evaluation
- TV-remote browsing (D-pad navigation)
- Queue, autoplay, shuffle, playlists, favorites
- Other platforms (Mixcloud, Twitch, Vimeo, SoundCloud)
- Accounts/Google login, multi-user, public launch
- Creator tipping and payments (non-custodial, creator opt-in)
- Native Android TV app, Cast
- Community features

## Decisions (open questions resolved)
- TV browser: Philips One TV default browser. YouTube embed, sound, fullscreen and D-pad focus verified (ticket #1).
- Video-playing pages must be served over HTTPS from a real domain. Plain HTTP on an IP origin triggers YouTube error 150 for many videos.
- Tagging LLM: free tiers only (e.g. Gemini, Groq, OpenRouter free models, Mistral). Separate tagging pipeline/app, decoupled from the web app, provider-swappable. Input is public metadata only.
- Legal check: owner skips it. No payments in scope until revisited by owner.

- Access protection: HTTP basic auth at reverse proxy for phone/UI. TV page is unauthenticated except for a short-lived, random pairing code created on the TV and confirmed from the authenticated phone. Pairing code expires and is single-use.

- Data: the droplet's SQLite file is the source of truth. YouTube API key is IP-restricted to the droplet, so ingest runs there; local development uses a pulled snapshot (automation: #16).

## Open questions
None blocking.

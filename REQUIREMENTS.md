# Vinyl Session Page: Requirements (draft v0.1)

## Vision
Aggregator of vinyl/DJ session videos with rich filtering (mood, scene, ...) and optional creator tipping. No video hosting; official embeds only.

## Constraints (from research)
- Embed via official players only (YouTube IFrame API etc.). No overlays on the player, no ad blocking, no audio-only extraction, no restreaming.
- No paywall on YouTube content.
- Ads: Premium users ad-free only via their own YouTube session. Not controllable, not promised.
- Money non-custodial. Creators supply own wallet/Lightning address and must opt in.
- Creator consent (email) before featuring branding or enabling payments.

## MVP (Phase 1)
### Catalog
- [ ] Curated list of YouTube videos (add by URL, metadata fetched via YouTube Data API)
- [ ] Fields: title, channel, duration, thumbnail, embed id, tags, added date
- [ ] Admin-only add/edit/remove (no public submissions yet)
- [ ] Dead/removed video detection

### Filtering and discovery
- [ ] Tag dimensions: mood, scene/setting, genre, format (vinyl/digital), duration, BPM range (optional), channel
- [ ] Multi-select filters, combinable, URL-shareable
- [ ] Text search (title, channel, tags)
- [ ] Sort: newest, duration, popular

### Playback
- [ ] Embedded YouTube player page per session
- [ ] Respect YouTube API ToS (unmodified player, no overlays)
- [ ] Link-out/"open on YouTube" fallback

### Accounts
- [ ] Sign in with Google (OAuth, minimal scopes)
- [ ] Favorites / watch later
- [ ] Anonymous browsing allowed

### Creator tipping (DEFERRED, not in Phase 1)
- [ ] Per-channel tip link (Ko-fi, Lightning address, etc.) shown next to the player
- [ ] Creator opt-in flag; hidden if not opted in
- [ ] No funds held by us

### Legal/compliance
- [ ] Terms, privacy policy, cookie consent (GDPR)
- [ ] Takedown/contact process
- [ ] Lawyer review before any payment or fee handling

### Non-functional
- [ ] Responsive/mobile-first
- [ ] Fast load (SSR or static + API)
- [ ] YouTube API quota handling (cache metadata)
- [ ] Basic analytics (privacy-friendly)

## Phase 2
- [ ] In-page payments (non-custodial: BTCPay/Strike/Stripe Connect) with 10% platform fee
- [ ] Creator onboarding portal (claim channel, set wallet, edit tags)
- [ ] Other platform embeds (Mixcloud, Twitch, Vimeo, SoundCloud)
- [ ] Account linking for platform-specific perks
- [ ] Community tagging / submissions with moderation

## Phase 3 (ideas)
- [ ] Playlists, queues, sharing
- [ ] Recommendations by mood
- [ ] Per-creator pages
- [ ] Streaming-sats while watching

## Decisions
- Public repo on personal GitHub (jirigrill).
- Hosting: existing DigitalOcean droplet. No video/media storage. Minimal budget.
- Jurisdiction: Czech Republic, personal project (no entity yet).
- Tagging: AI-assisted. Seed list first, then AI defines taxonomy and tags.
- Seed catalog: scrape metadata from a few vinyl-session YouTube channels (official YouTube Data API preferred over scraping HTML).
- Tipping: deferred out of Phase 1. Revisit after core page works.
- Ultimate goal: Android TV at home, session picked from iPhone and cast/streamed to TV.

## Platform strategy (TV goal)
- Phase 1: responsive web app (PWA-capable).
- TV options to evaluate: (a) web app opened in Android TV browser/WebView, (b) iPhone as remote controlling the TV page via WebSocket "pairing" (session code/QR), (c) native Android TV app (Kotlin/Compose) later, (d) Google Cast sender/receiver.
- Likely pick: web app + remote-control pairing (b). Works with YouTube embeds on TV without native app.
- Stack to be decided; SvelteKit is a candidate (skills available).

## Open questions
1. Final stack choice (SvelteKit vs alternatives)?
2. Which channels form the seed set?
3. Czech legal check (hobby income, VAT, crypto) before Phase 2 payments.

# Changelog — Online Voter Registration & Polling

All notable changes to this package are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- **Admin API key guard** — `POST /polls` requires `Authorization: Bearer <key>` matching
  `ADMIN_API_KEY` env var; server exits at startup if the key is not set
- **Poll lifecycle scheduler** — background goroutine auto-opens draft polls at `opens_at`
  and auto-closes open polls at `closes_at` (configurable tick interval, default 1 minute)
- **CORS middleware** — `CORSAllowedOrigins` env var wires the existing `nzcommon.CORS`
  middleware; defaults to `http://localhost:3006`
- **Redis poll-list cache** — active poll list is cached in Redis with a 30-second TTL;
  degrades gracefully when Redis is unavailable
- **NATS JetStream integration** — optional NATS connection for multi-instance SSE fanout;
  degrades gracefully when NATS is unavailable
- **Ranked-choice (IRV) ballots** — polls may be created with `ballotType: "ranked"`;
  voters rank options by preference; Instant Runoff Voting eliminates the lowest-scoring
  candidate each round until a majority winner is found
- **Merkle inclusion proof endpoint** — `GET /polls/{id}/merkle-proof?receipt={token}`
  returns an O(log n) path through the binary Merkle tree, allowing a voter to prove
  ballot inclusion without downloading the entire ballot list
- **Server-Sent Events live tally** — `GET /polls/{id}/live-results` streams `TallyEvent`
  updates as ballots are cast; 25-second keepalive comments prevent proxy timeouts
- **Audit proof pagination** — `GET /polls/{id}/audit` accepts `?offset` and `?limit`;
  the total ballot count is always returned so the client can paginate incrementally
- **Audit choice labels** — audit proof display shows the option text (e.g. "Alice Smith")
  instead of a raw numeric `choiceIndex`; ranked ballots show ordered preferences
- **CSV and JSON audit export** — one-click download buttons on the Audit Proof tab for
  offline independent verification
- **Countdown timer** — poll detail page shows a live countdown to the closing time,
  switching to red when less than one hour remains
- **QR code receipt** — after voting, a scannable QR code encodes the verification URL
  (`/results/{id}?receipt={token}`) so mobile voters can save proof without copy-pasting
- **IRV round display** — results page shows each elimination round's vote counts and
  which candidate was eliminated, alongside the final IRV winner badge
- **SSE live results chart** — results page subscribes to the SSE endpoint and
  re-fetches the tally on each event, with a pulsing "Live" indicator
- **Ranked-choice ballot form** — dropdown rank selector for each option with mutual-
  exclusion (each rank can only be assigned to one option); IRV info notice inline

### Changed
- `WriteTimeout` set to `0` on the HTTP server to allow indefinite SSE connections
- Health endpoint now pings the database and returns `503` if unreachable
- `CastBallot` handler accepts `{ choiceIndex }` for FPTP or `{ rankings }` for ranked
  ballots in a single `POST /polls/{id}/vote` endpoint
- `GetAuditProof` service method fetches all ballots for audit root computation even when
  responding with a paginated slice, so the root hash remains consistent across pages
- `NewPollService` and `NewResultHandler` now accept a `*TallyHub` for non-blocking
  tally-event publishing after each ballot insert

### Fixed
- Database migration `002_ranked_ballots.sql` adds `ballot_type` column to `polls` and
  `rankings` (nullable JSON) column to `ballots` with no breaking change to existing rows

---

## [0.1.0] — 2026-06-15

### Added
- Core voter registration with RealMe Verified identity (SHA-256 FLT hashing, Privacy Act 2020 compliant)
- Local body poll management: create, open, and close lifecycle with draft/open/closed states
- Anonymous ballot casting using per-poll voter-token derivation: `sha256(flt_hash + poll_id + poll_salt)`
- Public audit proof: Helios-style commitment scheme (`sha256(voter_token + choice + receipt_token)`) with lexicographically sorted audit root for independent tally verification
- Receipt token: voters can prove their ballot was counted without revealing their choice
- Next.js 14 frontend with ballot form, bar-chart results, and full audit proof display with in-browser receipt search
- RealMe SAML 2.0 integration supporting MTS, ITE, and Production environments
- Mock IdP (`packages/realme-go/testenv/`) for local development without DIA credentials
- PostgreSQL schema with Atlas-compatible migrations and `voter_portal` schema isolation
- Docker Compose local development setup (PostgreSQL, Redis, NATS)
- Go unit tests covering voter token derivation, commitment scheme, audit root, and handler auth guards

### Security
- FLT is never stored — only `sha256(FLT)` persists in the voters table
- UNIQUE(poll_id, voter_token) constraint enforces one-vote-per-voter at the database level
- Per-poll random salt ensures voter tokens cannot be linked across polls
- `ForceAuthn: true` on all RealMe sessions to prevent session replay

### Fixed (pre-release)
- `go.sum` removed from `.gitignore` — was incorrectly excluded, breaking reproducible builds
- PostgreSQL `search_path` not set on connection pool — all queries failed with "relation does not exist" in the `voter_portal` schema
- RealMe FLT removed from `/auth/status` response — was unnecessarily exposing a unique identifier to the browser
- Unique constraint violation in `CastBallot` now returns a clean "already voted" error instead of leaking the raw PostgreSQL error message
- "Verify Your Vote" link in poll page pointed to a non-existent route; corrected to `/results/{id}?receipt={token}`
- Silent `catch {}` in registration page now surfaces network errors to the user
- Added `next.config.js` with API proxy rewrites — relative `fetch("/polls/...")` calls previously returned 404 from Next.js
- Added Tailwind CSS (`tailwindcss`, `autoprefixer`, `postcss`) to `web/` — the app was completely unstyled without these devDependencies and config files
- Removed dead `CookieDomain` config field that was loaded from env but never passed to `realme.Config`

[Unreleased]: https://github.com/tpt-nz/tpt-voter-portal-nz/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/tpt-nz/tpt-voter-portal-nz/releases/tag/v0.1.0

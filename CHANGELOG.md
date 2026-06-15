# Changelog — Online Voter Registration & Polling

All notable changes to this package are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

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

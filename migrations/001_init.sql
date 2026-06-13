SET search_path TO voter_portal;

-- 001_init.sql: Initial schema for the Online Voter Registration & Polling app.
-- Atlas-compatible: run with `atlas schema apply --dir file://migrations --url $DATABASE_URL`
--
-- Security design notes:
-- - voters: only flt_hash (sha256 of FLT) is stored — no FLT, name, DOB, or address
-- - ballots: voter_token = sha256(flt_hash + poll_id + poll_salt), no PII whatsoever
-- - UNIQUE(poll_id, voter_token) enforces one-vote-per-voter at the database level
-- - receipt_token allows a voter to verify their vote in the public audit
-- - commitment = sha256(voter_token + choice + receipt_token) for tamper-evidence
-- - tallies: precomputed results with audit_root for independent verification
-- Electoral Act 1993 scope: local body polling only; not for Parliamentary elections.

CREATE TABLE IF NOT EXISTS voters (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    flt_hash      TEXT        NOT NULL UNIQUE,  -- sha256(RealMe FLT), never the FLT itself
    status        TEXT        NOT NULL DEFAULT 'registered' CHECK (status IN ('registered', 'suspended')),
    registered_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_voters_flt_hash ON voters (flt_hash);

CREATE TABLE IF NOT EXISTS polls (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT        NOT NULL,
    description TEXT,
    options     TEXT        NOT NULL DEFAULT '[]',  -- JSON array of option strings
    status      TEXT        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'open', 'closed')),
    poll_salt   TEXT        NOT NULL,               -- random hex; participates in voter_token derivation
    opens_at    TIMESTAMPTZ NOT NULL,
    closes_at   TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT polls_closes_after_opens CHECK (closes_at > opens_at)
);

CREATE INDEX idx_polls_status    ON polls (status);
CREATE INDEX idx_polls_opens_at  ON polls (opens_at);
CREATE INDEX idx_polls_closes_at ON polls (closes_at);

-- ballots: the anonymous vote store.
-- No PII or FLT is present in this table.
CREATE TABLE IF NOT EXISTS ballots (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id       UUID        NOT NULL REFERENCES polls(id) ON DELETE RESTRICT,
    voter_token   TEXT        NOT NULL,   -- sha256(flt_hash + poll_id + poll_salt)
    choice_index  INTEGER     NOT NULL CHECK (choice_index >= 0),
    receipt_token TEXT        NOT NULL UNIQUE,  -- voter's proof of participation
    commitment    TEXT        NOT NULL,   -- sha256(voter_token + choice_str + receipt_token)
    cast_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (poll_id, voter_token)         -- one vote per voter per poll
);

CREATE INDEX idx_ballots_poll_id       ON ballots (poll_id);
CREATE INDEX idx_ballots_receipt_token ON ballots (receipt_token);
CREATE INDEX idx_ballots_cast_at       ON ballots (cast_at DESC);

-- tallies: computed results, updated on demand or when a poll closes.
CREATE TABLE IF NOT EXISTS tallies (
    poll_id     UUID        PRIMARY KEY REFERENCES polls(id) ON DELETE CASCADE,
    counts      TEXT        NOT NULL DEFAULT '{}',  -- JSON {"0": N, "1": M, ...}
    total_votes INTEGER     NOT NULL DEFAULT 0,
    audit_root  TEXT        NOT NULL,   -- sha256 of lexicographically-sorted commitments
    computed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

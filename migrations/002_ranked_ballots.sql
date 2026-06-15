SET search_path TO voter_portal;

-- 002_ranked_ballots.sql: Add ranked-choice (IRV) ballot support.
--
-- ballot_type: 'fptp' (first-past-the-post, default) or 'ranked' (IRV).
-- rankings: ordered JSON array of choice indices for ranked ballots.
--           NULL for FPTP ballots.
-- choice_index: kept for FPTP; set to -1 for ranked ballots.

ALTER TABLE polls
    ADD COLUMN IF NOT EXISTS ballot_type TEXT NOT NULL DEFAULT 'fptp'
        CHECK (ballot_type IN ('fptp', 'ranked'));

ALTER TABLE ballots
    ADD COLUMN IF NOT EXISTS rankings TEXT;  -- JSON array, e.g. [2,0,1]; NULL for FPTP

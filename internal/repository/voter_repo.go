// Package repository provides database access for voter portal entities.
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/models"
)

// VoterRepository handles all database operations for voters, polls, ballots, and tallies.
type VoterRepository struct {
	pool *pgxpool.Pool
}

// NewVoterRepository creates a new repository backed by the given connection pool.
func NewVoterRepository(pool *pgxpool.Pool) *VoterRepository {
	return &VoterRepository{pool: pool}
}

// --- Voters ---

// UpsertVoter inserts a voter by FLT hash if not already registered.
// On conflict (same flt_hash) it returns the existing record unchanged.
func (r *VoterRepository) UpsertVoter(ctx context.Context, fltHash string) (*models.Voter, error) {
	id := uuid.New()
	now := time.Now().UTC()

	query := `
		INSERT INTO voters (id, flt_hash, status, registered_at)
		VALUES ($1, $2, 'registered', $3)
		ON CONFLICT (flt_hash) DO NOTHING`

	_, err := r.pool.Exec(ctx, query, id, fltHash, now)
	if err != nil {
		return nil, fmt.Errorf("voter_repo: upsert voter: %w", err)
	}

	return r.GetVoterByFLTHash(ctx, fltHash)
}

// GetVoterByFLTHash retrieves a voter by their FLT hash. Returns nil if not found.
func (r *VoterRepository) GetVoterByFLTHash(ctx context.Context, fltHash string) (*models.Voter, error) {
	query := `SELECT id, flt_hash, status, registered_at FROM voters WHERE flt_hash = $1`

	v := &models.Voter{}
	err := r.pool.QueryRow(ctx, query, fltHash).Scan(&v.ID, &v.FLTHash, &v.Status, &v.RegisteredAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("voter_repo: get voter by flt hash: %w", err)
	}
	return v, nil
}

// --- Polls ---

// CreatePoll inserts a new poll record.
func (r *VoterRepository) CreatePoll(ctx context.Context, p *models.Poll) error {
	p.ID = uuid.New()
	p.CreatedAt = time.Now().UTC()

	optionsJSON, err := json.Marshal(p.Options)
	if err != nil {
		return fmt.Errorf("voter_repo: marshal poll options: %w", err)
	}

	query := `
		INSERT INTO polls (id, title, description, options, status, poll_salt, opens_at, closes_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = r.pool.Exec(ctx, query,
		p.ID, p.Title, p.Description, string(optionsJSON),
		p.Status, p.PollSalt, p.OpensAt, p.ClosesAt, p.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("voter_repo: create poll: %w", err)
	}
	return nil
}

// GetPollByID retrieves a poll by its UUID. Returns nil if not found.
func (r *VoterRepository) GetPollByID(ctx context.Context, id uuid.UUID) (*models.Poll, error) {
	query := `
		SELECT id, title, description, options, status, poll_salt, opens_at, closes_at, created_at
		FROM polls WHERE id = $1`

	p := &models.Poll{}
	var optionsJSON string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Title, &p.Description, &optionsJSON,
		&p.Status, &p.PollSalt, &p.OpensAt, &p.ClosesAt, &p.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("voter_repo: get poll by id %s: %w", id, err)
	}
	if err := json.Unmarshal([]byte(optionsJSON), &p.Options); err != nil {
		return nil, fmt.Errorf("voter_repo: unmarshal poll options: %w", err)
	}
	return p, nil
}

// GetActivePolls returns all polls currently open for voting, ordered by opens_at.
func (r *VoterRepository) GetActivePolls(ctx context.Context) ([]models.Poll, error) {
	query := `
		SELECT id, title, description, options, status, poll_salt, opens_at, closes_at, created_at
		FROM polls WHERE status = 'open' ORDER BY opens_at DESC`

	return r.scanPolls(ctx, query)
}

// GetAllPolls returns all polls regardless of status.
func (r *VoterRepository) GetAllPolls(ctx context.Context) ([]models.Poll, error) {
	query := `
		SELECT id, title, description, options, status, poll_salt, opens_at, closes_at, created_at
		FROM polls ORDER BY created_at DESC`

	return r.scanPolls(ctx, query)
}

// UpdatePollStatus updates a poll's status.
func (r *VoterRepository) UpdatePollStatus(ctx context.Context, id uuid.UUID, status models.PollStatus) error {
	_, err := r.pool.Exec(ctx, `UPDATE polls SET status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("voter_repo: update poll status: %w", err)
	}
	return nil
}

func (r *VoterRepository) scanPolls(ctx context.Context, query string, args ...interface{}) ([]models.Poll, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("voter_repo: query polls: %w", err)
	}
	defer rows.Close()

	var polls []models.Poll
	for rows.Next() {
		var p models.Poll
		var optionsJSON string
		if err := rows.Scan(
			&p.ID, &p.Title, &p.Description, &optionsJSON,
			&p.Status, &p.PollSalt, &p.OpensAt, &p.ClosesAt, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("voter_repo: scan poll: %w", err)
		}
		if err := json.Unmarshal([]byte(optionsJSON), &p.Options); err != nil {
			return nil, fmt.Errorf("voter_repo: unmarshal poll options: %w", err)
		}
		polls = append(polls, p)
	}
	return polls, rows.Err()
}

// --- Ballots ---

// InsertBallot records an anonymous ballot. The UNIQUE constraint on
// (poll_id, voter_token) prevents double-voting at the database level.
func (r *VoterRepository) InsertBallot(ctx context.Context, b *models.Ballot) error {
	b.ID = uuid.New()
	b.CastAt = time.Now().UTC()

	query := `
		INSERT INTO ballots (id, poll_id, voter_token, choice_index, receipt_token, commitment, cast_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query,
		b.ID, b.PollID, b.VoterToken, b.ChoiceIndex, b.ReceiptToken, b.Commitment, b.CastAt,
	)
	if err != nil {
		return fmt.Errorf("voter_repo: insert ballot: %w", err)
	}
	return nil
}

// HasVoted checks whether a voter_token has already voted in the given poll.
func (r *VoterRepository) HasVoted(ctx context.Context, pollID uuid.UUID, voterToken string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ballots WHERE poll_id = $1 AND voter_token = $2`,
		pollID, voterToken,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("voter_repo: has voted: %w", err)
	}
	return count > 0, nil
}

// GetReceiptByVoterToken returns the ballot receipt for a voter in a given poll.
func (r *VoterRepository) GetReceiptByVoterToken(ctx context.Context, pollID uuid.UUID, voterToken string) (*models.BallotReceipt, error) {
	query := `SELECT poll_id, choice_index, receipt_token, cast_at FROM ballots WHERE poll_id = $1 AND voter_token = $2`

	rec := &models.BallotReceipt{}
	err := r.pool.QueryRow(ctx, query, pollID, voterToken).Scan(
		&rec.PollID, &rec.ChoiceIndex, &rec.ReceiptToken, &rec.CastAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("voter_repo: get receipt by voter token: %w", err)
	}
	return rec, nil
}

// GetBallotsByPoll returns all ballots for a poll (for audit purposes).
func (r *VoterRepository) GetBallotsByPoll(ctx context.Context, pollID uuid.UUID) ([]models.Ballot, error) {
	query := `
		SELECT id, poll_id, voter_token, choice_index, receipt_token, commitment, cast_at
		FROM ballots WHERE poll_id = $1 ORDER BY cast_at ASC`

	rows, err := r.pool.Query(ctx, query, pollID)
	if err != nil {
		return nil, fmt.Errorf("voter_repo: get ballots by poll: %w", err)
	}
	defer rows.Close()

	var ballots []models.Ballot
	for rows.Next() {
		var b models.Ballot
		if err := rows.Scan(&b.ID, &b.PollID, &b.VoterToken, &b.ChoiceIndex, &b.ReceiptToken, &b.Commitment, &b.CastAt); err != nil {
			return nil, fmt.Errorf("voter_repo: scan ballot: %w", err)
		}
		ballots = append(ballots, b)
	}
	return ballots, rows.Err()
}

// CountVotesByPoll returns a map of choice_index -> count for a given poll.
func (r *VoterRepository) CountVotesByPoll(ctx context.Context, pollID uuid.UUID) (map[int]int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT choice_index, COUNT(*) FROM ballots WHERE poll_id = $1 GROUP BY choice_index`,
		pollID,
	)
	if err != nil {
		return nil, fmt.Errorf("voter_repo: count votes by poll: %w", err)
	}
	defer rows.Close()

	counts := make(map[int]int)
	for rows.Next() {
		var idx, count int
		if err := rows.Scan(&idx, &count); err != nil {
			return nil, fmt.Errorf("voter_repo: scan vote count: %w", err)
		}
		counts[idx] = count
	}
	return counts, rows.Err()
}

// GetBallotByReceiptToken looks up a ballot by its public receipt token.
func (r *VoterRepository) GetBallotByReceiptToken(ctx context.Context, receiptToken string) (*models.Ballot, error) {
	query := `
		SELECT id, poll_id, voter_token, choice_index, receipt_token, commitment, cast_at
		FROM ballots WHERE receipt_token = $1`

	b := &models.Ballot{}
	err := r.pool.QueryRow(ctx, query, receiptToken).Scan(
		&b.ID, &b.PollID, &b.VoterToken, &b.ChoiceIndex, &b.ReceiptToken, &b.Commitment, &b.CastAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("voter_repo: get ballot by receipt token: %w", err)
	}
	return b, nil
}

// --- Tallies ---

// UpsertTally inserts or replaces the tally for a poll.
func (r *VoterRepository) UpsertTally(ctx context.Context, t *models.StoredTally) error {
	countsJSON, err := json.Marshal(t.Counts)
	if err != nil {
		return fmt.Errorf("voter_repo: marshal tally counts: %w", err)
	}

	query := `
		INSERT INTO tallies (poll_id, counts, total_votes, audit_root, computed_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (poll_id) DO UPDATE
		SET counts = EXCLUDED.counts,
		    total_votes = EXCLUDED.total_votes,
		    audit_root = EXCLUDED.audit_root,
		    computed_at = EXCLUDED.computed_at`

	_, err = r.pool.Exec(ctx, query,
		t.PollID, string(countsJSON), t.TotalVotes, t.AuditRoot, t.ComputedAt,
	)
	if err != nil {
		return fmt.Errorf("voter_repo: upsert tally: %w", err)
	}
	return nil
}

// GetTallyByPoll retrieves the stored tally for a poll. Returns nil if not yet computed.
func (r *VoterRepository) GetTallyByPoll(ctx context.Context, pollID uuid.UUID) (*models.StoredTally, error) {
	query := `SELECT poll_id, counts, total_votes, audit_root, computed_at FROM tallies WHERE poll_id = $1`

	t := &models.StoredTally{}
	var countsJSON string
	err := r.pool.QueryRow(ctx, query, pollID).Scan(&t.PollID, &countsJSON, &t.TotalVotes, &t.AuditRoot, &t.ComputedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("voter_repo: get tally by poll: %w", err)
	}
	if err := json.Unmarshal([]byte(countsJSON), &t.Counts); err != nil {
		return nil, fmt.Errorf("voter_repo: unmarshal tally counts: %w", err)
	}
	return t, nil
}

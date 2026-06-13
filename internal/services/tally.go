package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/models"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/repository"
)

// TallyService computes and serves poll results with a public audit proof.
//
// Audit design (Helios-style):
//   - Each ballot has a commitment = sha256(voter_token + choice_str + receipt_token).
//   - The audit_root is sha256 of all commitments sorted lexicographically.
//   - Anyone can re-derive audit_root from the public ballot list to verify integrity.
//   - A voter verifies their vote was counted by finding their receipt_token in the list.
type TallyService struct {
	repo   *repository.VoterRepository
	logger *slog.Logger
}

// NewTallyService creates a new tally service.
func NewTallyService(repo *repository.VoterRepository, logger *slog.Logger) *TallyService {
	return &TallyService{
		repo:   repo,
		logger: logger.With("service", "tally"),
	}
}

// ComputeAndStoreTally counts votes for a poll, computes the audit root, and
// persists the result. Safe to call multiple times — it upserts on conflict.
func (s *TallyService) ComputeAndStoreTally(ctx context.Context, pollID uuid.UUID) (*models.Tally, error) {
	poll, err := s.repo.GetPollByID(ctx, pollID)
	if err != nil {
		return nil, fmt.Errorf("tally: get poll: %w", err)
	}
	if poll == nil {
		return nil, fmt.Errorf("tally: poll not found")
	}

	ballots, err := s.repo.GetBallotsByPoll(ctx, pollID)
	if err != nil {
		return nil, fmt.Errorf("tally: get ballots: %w", err)
	}

	// Count votes
	counts := make(map[string]int)
	for i := range poll.Options {
		counts[strconv.Itoa(i)] = 0
	}
	commitments := make([]string, 0, len(ballots))
	for _, b := range ballots {
		key := strconv.Itoa(b.ChoiceIndex)
		counts[key]++
		commitments = append(commitments, b.Commitment)
	}

	auditRoot := computeAuditRoot(commitments)
	now := time.Now().UTC()

	stored := &models.StoredTally{
		PollID:     pollID,
		Counts:     counts,
		TotalVotes: len(ballots),
		AuditRoot:  auditRoot,
		ComputedAt: now,
	}

	if err := s.repo.UpsertTally(ctx, stored); err != nil {
		return nil, fmt.Errorf("tally: store: %w", err)
	}

	s.logger.InfoContext(ctx, "tally computed", "pollID", pollID, "total", len(ballots))

	return &models.Tally{
		Poll:       poll,
		Counts:     counts,
		TotalVotes: len(ballots),
		AuditRoot:  auditRoot,
		ComputedAt: now,
	}, nil
}

// GetTally returns the tally for a poll, computing it on-demand if not yet stored.
func (s *TallyService) GetTally(ctx context.Context, pollID uuid.UUID) (*models.Tally, error) {
	poll, err := s.repo.GetPollByID(ctx, pollID)
	if err != nil {
		return nil, fmt.Errorf("tally: get poll: %w", err)
	}
	if poll == nil {
		return nil, fmt.Errorf("tally: poll not found")
	}

	stored, err := s.repo.GetTallyByPoll(ctx, pollID)
	if err != nil {
		return nil, fmt.Errorf("tally: get stored: %w", err)
	}

	// Compute on-demand if no stored tally yet
	if stored == nil {
		return s.ComputeAndStoreTally(ctx, pollID)
	}

	return &models.Tally{
		Poll:       poll,
		Counts:     stored.Counts,
		TotalVotes: stored.TotalVotes,
		AuditRoot:  stored.AuditRoot,
		ComputedAt: stored.ComputedAt,
	}, nil
}

// GetAuditProof returns the complete public ballot list for a poll.
// Each entry includes the receipt_token, commitment, choice, and timestamp.
// The audit_root can be independently verified by sorting commitments and hashing.
func (s *TallyService) GetAuditProof(ctx context.Context, pollID uuid.UUID) (*models.AuditProof, error) {
	ballots, err := s.repo.GetBallotsByPoll(ctx, pollID)
	if err != nil {
		return nil, fmt.Errorf("tally: get audit proof: %w", err)
	}

	entries := make([]models.AuditEntry, 0, len(ballots))
	commitments := make([]string, 0, len(ballots))
	for _, b := range ballots {
		entries = append(entries, models.AuditEntry{
			ReceiptToken: b.ReceiptToken,
			ChoiceIndex:  b.ChoiceIndex,
			Commitment:   b.Commitment,
			CastAt:       b.CastAt,
		})
		commitments = append(commitments, b.Commitment)
	}

	return &models.AuditProof{
		PollID:    pollID,
		Entries:   entries,
		AuditRoot: computeAuditRoot(commitments),
		Total:     len(entries),
	}, nil
}

// VerifyReceipt checks whether a receipt token appears in the poll's ballot list,
// confirming the vote was counted.
func (s *TallyService) VerifyReceipt(ctx context.Context, pollID uuid.UUID, receiptToken string) (bool, *models.AuditEntry, error) {
	ballot, err := s.repo.GetBallotByReceiptToken(ctx, receiptToken)
	if err != nil {
		return false, nil, fmt.Errorf("tally: verify receipt: %w", err)
	}
	if ballot == nil || ballot.PollID != pollID {
		return false, nil, nil
	}

	entry := &models.AuditEntry{
		ReceiptToken: ballot.ReceiptToken,
		ChoiceIndex:  ballot.ChoiceIndex,
		Commitment:   ballot.Commitment,
		CastAt:       ballot.CastAt,
	}
	return true, entry, nil
}

// computeAuditRoot produces a tamper-evident root hash from a set of ballot commitments.
// Commitments are sorted lexicographically before hashing so the result is deterministic.
func computeAuditRoot(commitments []string) string {
	if len(commitments) == 0 {
		return hex.EncodeToString(sha256.New().Sum(nil))
	}
	sorted := make([]string, len(commitments))
	copy(sorted, commitments)
	sort.Strings(sorted)

	h := sha256.New()
	for _, c := range sorted {
		h.Write([]byte(c))
	}
	return hex.EncodeToString(h.Sum(nil))
}

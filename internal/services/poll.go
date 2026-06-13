package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/models"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/repository"
)

// PollService handles poll creation, voting, and ballot management.
type PollService struct {
	repo   *repository.VoterRepository
	logger *slog.Logger
}

// NewPollService creates a new poll service.
func NewPollService(repo *repository.VoterRepository, logger *slog.Logger) *PollService {
	return &PollService{
		repo:   repo,
		logger: logger.With("service", "poll"),
	}
}

// CreatePoll creates a new poll with a randomly-generated per-poll salt.
// The salt is used in voter_token derivation and is stored server-side only.
func (s *PollService) CreatePoll(ctx context.Context, req *models.CreatePollRequest) (*models.Poll, error) {
	if err := validateCreatePollRequest(req); err != nil {
		return nil, err
	}

	salt, err := generateSalt()
	if err != nil {
		return nil, fmt.Errorf("poll: generate salt: %w", err)
	}

	poll := &models.Poll{
		Title:       req.Title,
		Description: req.Description,
		Options:     req.Options,
		Status:      models.PollStatusDraft,
		PollSalt:    salt,
		OpensAt:     req.OpensAt,
		ClosesAt:    req.ClosesAt,
	}

	// Automatically open the poll if OpensAt is in the past or now
	if !req.OpensAt.After(time.Now()) {
		poll.Status = models.PollStatusOpen
	}

	if err := s.repo.CreatePoll(ctx, poll); err != nil {
		return nil, fmt.Errorf("poll: create: %w", err)
	}

	s.logger.InfoContext(ctx, "poll created", "pollID", poll.ID, "status", poll.Status)
	return poll, nil
}

// GetActivePolls returns all polls currently open for voting.
func (s *PollService) GetActivePolls(ctx context.Context) ([]models.Poll, error) {
	polls, err := s.repo.GetActivePolls(ctx)
	if err != nil {
		return nil, fmt.Errorf("poll: get active: %w", err)
	}
	return polls, nil
}

// GetPollByID returns a poll by its UUID. Returns nil if not found.
func (s *PollService) GetPollByID(ctx context.Context, id uuid.UUID) (*models.Poll, error) {
	poll, err := s.repo.GetPollByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("poll: get by id: %w", err)
	}
	return poll, nil
}

// CastBallot records an anonymous vote for a poll.
//
// ZK deduplication design:
//   - voter_token = sha256(flt_hash + poll_id + poll_salt) — unique per voter per poll, not reversible
//   - receipt_token = random UUID — voter's personal proof; returned to them, stored publicly
//   - commitment = sha256(voter_token + choice_str + receipt_token) — public audit anchor
//   - The database UNIQUE(poll_id, voter_token) enforces one-vote-per-voter at the storage layer
//
// No FLT or PII is stored in the ballot table.
func (s *PollService) CastBallot(ctx context.Context, flt string, pollID uuid.UUID, choiceIndex int) (*models.BallotReceipt, error) {
	poll, err := s.repo.GetPollByID(ctx, pollID)
	if err != nil {
		return nil, fmt.Errorf("poll: cast ballot: get poll: %w", err)
	}
	if poll == nil {
		return nil, fmt.Errorf("poll: cast ballot: poll not found")
	}
	if poll.Status != models.PollStatusOpen {
		return nil, fmt.Errorf("poll: cast ballot: poll is not open for voting")
	}
	if choiceIndex < 0 || choiceIndex >= len(poll.Options) {
		return nil, fmt.Errorf("poll: cast ballot: choice index %d out of range [0, %d)", choiceIndex, len(poll.Options))
	}

	fltHash := FLTHash(flt)
	voterToken := deriveVoterToken(fltHash, poll.ID.String(), poll.PollSalt)

	// Check for existing vote before attempting insert (cleaner error message)
	hasVoted, err := s.repo.HasVoted(ctx, pollID, voterToken)
	if err != nil {
		return nil, fmt.Errorf("poll: cast ballot: check has voted: %w", err)
	}
	if hasVoted {
		return nil, fmt.Errorf("poll: cast ballot: voter has already voted in this poll")
	}

	receiptToken := uuid.New().String()
	commitment := deriveCommitment(voterToken, strconv.Itoa(choiceIndex), receiptToken)

	ballot := &models.Ballot{
		PollID:       pollID,
		VoterToken:   voterToken,
		ChoiceIndex:  choiceIndex,
		ReceiptToken: receiptToken,
		Commitment:   commitment,
	}

	if err := s.repo.InsertBallot(ctx, ballot); err != nil {
		return nil, fmt.Errorf("poll: cast ballot: insert: %w", err)
	}

	s.logger.InfoContext(ctx, "ballot cast", "pollID", pollID, "choiceIndex", choiceIndex)

	return &models.BallotReceipt{
		ReceiptToken: receiptToken,
		PollID:       pollID,
		ChoiceIndex:  choiceIndex,
		CastAt:       ballot.CastAt,
	}, nil
}

// GetVoterReceipt returns the ballot receipt for a given voter in a given poll,
// or nil if they have not voted. Used by the "my-receipt" endpoint.
func (s *PollService) GetVoterReceipt(ctx context.Context, flt string, pollID uuid.UUID) (*models.BallotReceipt, error) {
	poll, err := s.repo.GetPollByID(ctx, pollID)
	if err != nil {
		return nil, fmt.Errorf("poll: get voter receipt: get poll: %w", err)
	}
	if poll == nil {
		return nil, fmt.Errorf("poll: get voter receipt: poll not found")
	}

	fltHash := FLTHash(flt)
	voterToken := deriveVoterToken(fltHash, poll.ID.String(), poll.PollSalt)

	receipt, err := s.repo.GetReceiptByVoterToken(ctx, pollID, voterToken)
	if err != nil {
		return nil, fmt.Errorf("poll: get voter receipt: %w", err)
	}
	return receipt, nil
}

// deriveVoterToken computes the anonymous per-poll voter identifier.
// voter_token = sha256(flt_hash || poll_id || poll_salt)
func deriveVoterToken(fltHash, pollID, pollSalt string) string {
	h := sha256.New()
	h.Write([]byte(fltHash + pollID + pollSalt))
	return hex.EncodeToString(h.Sum(nil))
}

// deriveCommitment computes the public audit commitment for a ballot.
// commitment = sha256(voter_token || choice_str || receipt_token)
func deriveCommitment(voterToken, choiceStr, receiptToken string) string {
	h := sha256.New()
	h.Write([]byte(voterToken + choiceStr + receiptToken))
	return hex.EncodeToString(h.Sum(nil))
}

// generateSalt produces 32 random bytes encoded as a hex string.
func generateSalt() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func validateCreatePollRequest(req *models.CreatePollRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(req.Options) < 2 {
		return fmt.Errorf("at least two options are required")
	}
	if req.ClosesAt.IsZero() {
		return fmt.Errorf("closesAt is required")
	}
	if !req.OpensAt.IsZero() && !req.ClosesAt.After(req.OpensAt) {
		return fmt.Errorf("closesAt must be after opensAt")
	}
	return nil
}

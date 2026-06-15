package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/models"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/repository"
)

// PollService handles poll creation, voting, and ballot management.
type PollService struct {
	repo   *repository.VoterRepository
	hub    *TallyHub
	logger *slog.Logger
}

// NewPollService creates a new poll service.
func NewPollService(repo *repository.VoterRepository, hub *TallyHub, logger *slog.Logger) *PollService {
	return &PollService{
		repo:   repo,
		hub:    hub,
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

	bt := req.BallotType
	if bt == "" {
		bt = models.BallotTypeFPTP
	}

	poll := &models.Poll{
		Title:       req.Title,
		Description: req.Description,
		Options:     req.Options,
		BallotType:  bt,
		Status:      models.PollStatusDraft,
		PollSalt:    salt,
		OpensAt:     req.OpensAt,
		ClosesAt:    req.ClosesAt,
	}

	if !req.OpensAt.After(time.Now()) {
		poll.Status = models.PollStatusOpen
	}

	if err := s.repo.CreatePoll(ctx, poll); err != nil {
		return nil, fmt.Errorf("poll: create: %w", err)
	}

	s.logger.InfoContext(ctx, "poll created", "pollID", poll.ID, "status", poll.Status, "ballotType", poll.BallotType)
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

// CastBallot records an anonymous vote for a poll. Supports both FPTP and
// ranked-choice (IRV) ballot types.
//
// ZK deduplication design:
//   - voter_token = sha256(flt_hash + poll_id + poll_salt) — unique per voter per poll, not reversible
//   - receipt_token = random UUID — voter's personal proof; returned to them, stored publicly
//   - commitment = sha256(voter_token + choice_str + receipt_token) — public audit anchor
//     For ranked ballots, choice_str is the JSON-encoded rankings array.
//   - UNIQUE(poll_id, voter_token) enforces one-vote-per-voter at the storage layer
//
// No FLT or PII is stored in the ballot table.
func (s *PollService) CastBallot(ctx context.Context, flt string, pollID uuid.UUID, req *models.CastBallotRequest) (*models.BallotReceipt, error) {
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

	fltHash := FLTHash(flt)
	voterToken := deriveVoterToken(fltHash, poll.ID.String(), poll.PollSalt)

	hasVoted, err := s.repo.HasVoted(ctx, pollID, voterToken)
	if err != nil {
		return nil, fmt.Errorf("poll: cast ballot: check has voted: %w", err)
	}
	if hasVoted {
		return nil, fmt.Errorf("poll: cast ballot: voter has already voted in this poll")
	}

	receiptToken := uuid.New().String()

	var ballot *models.Ballot

	if poll.BallotType == models.BallotTypeRanked {
		if err := validateRankings(req.Rankings, len(poll.Options)); err != nil {
			return nil, fmt.Errorf("poll: cast ballot: %w", err)
		}
		rankingsJSON, _ := json.Marshal(req.Rankings)
		commitment := deriveCommitment(voterToken, string(rankingsJSON), receiptToken)
		ballot = &models.Ballot{
			PollID:       pollID,
			VoterToken:   voterToken,
			ChoiceIndex:  -1,
			Rankings:     req.Rankings,
			ReceiptToken: receiptToken,
			Commitment:   commitment,
		}
	} else {
		if req.ChoiceIndex < 0 || req.ChoiceIndex >= len(poll.Options) {
			return nil, fmt.Errorf("poll: cast ballot: choice index %d out of range [0, %d)", req.ChoiceIndex, len(poll.Options))
		}
		commitment := deriveCommitment(voterToken, strconv.Itoa(req.ChoiceIndex), receiptToken)
		ballot = &models.Ballot{
			PollID:       pollID,
			VoterToken:   voterToken,
			ChoiceIndex:  req.ChoiceIndex,
			ReceiptToken: receiptToken,
			Commitment:   commitment,
		}
	}

	if err := s.repo.InsertBallot(ctx, ballot); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("poll: cast ballot: voter has already voted in this poll")
		}
		return nil, fmt.Errorf("poll: cast ballot: insert: %w", err)
	}

	s.logger.InfoContext(ctx, "ballot cast", "pollID", pollID, "ballotType", poll.BallotType)

	// Notify SSE subscribers (non-blocking via hub)
	if s.hub != nil {
		// Count ballots for live event without blocking the response path
		go func() {
			count, _ := s.repo.CountVotesByPoll(context.Background(), pollID)
			total := 0
			for _, n := range count {
				total += n
			}
			s.hub.Publish(models.TallyEvent{PollID: pollID.String(), TotalVotes: total})
		}()
	}

	return &models.BallotReceipt{
		ReceiptToken: receiptToken,
		PollID:       pollID,
		ChoiceIndex:  ballot.ChoiceIndex,
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

func validateRankings(rankings []int, numOptions int) error {
	if len(rankings) == 0 {
		return fmt.Errorf("rankings must not be empty for a ranked-choice poll")
	}
	seen := make(map[int]bool, len(rankings))
	for _, idx := range rankings {
		if idx < 0 || idx >= numOptions {
			return fmt.Errorf("ranking index %d out of range [0, %d)", idx, numOptions)
		}
		if seen[idx] {
			return fmt.Errorf("duplicate ranking index %d", idx)
		}
		seen[idx] = true
	}
	return nil
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

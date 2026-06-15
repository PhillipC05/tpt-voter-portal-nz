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

// AuditPageSize is the default page size for audit proof requests.
const AuditPageSize = repository.DefaultAuditPageSize

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
// persists the result. For ranked-choice polls it also runs IRV and stores the result.
// Safe to call multiple times — it upserts on conflict.
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

	counts := make(map[string]int)
	for i := range poll.Options {
		counts[strconv.Itoa(i)] = 0
	}

	commitments := make([]string, 0, len(ballots))
	var irvResult *models.IRVResult

	if poll.BallotType == models.BallotTypeRanked {
		rankings := make([][]int, 0, len(ballots))
		for _, b := range ballots {
			commitments = append(commitments, b.Commitment)
			if len(b.Rankings) > 0 {
				rankings = append(rankings, b.Rankings)
			}
		}
		result := ComputeIRV(rankings, len(poll.Options))
		irvResult = &result
		if result.Winner != nil {
			counts[strconv.Itoa(*result.Winner)] = len(ballots)
		}
	} else {
		for _, b := range ballots {
			counts[strconv.Itoa(b.ChoiceIndex)]++
			commitments = append(commitments, b.Commitment)
		}
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
		IRVResult:  irvResult,
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

// GetAuditProof returns a paginated page of the public ballot list for a poll.
// AuditRoot is always the root over ALL commitments, regardless of the page.
func (s *TallyService) GetAuditProof(ctx context.Context, pollID uuid.UUID, offset, limit int) (*models.AuditProof, error) {
	if limit <= 0 || limit > 500 {
		limit = AuditPageSize
	}

	ballots, total, err := s.repo.GetBallotsByPollPaginated(ctx, pollID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("tally: get audit proof: %w", err)
	}

	entries := make([]models.AuditEntry, 0, len(ballots))
	for _, b := range ballots {
		entries = append(entries, models.AuditEntry{
			ReceiptToken: b.ReceiptToken,
			ChoiceIndex:  b.ChoiceIndex,
			Rankings:     b.Rankings,
			Commitment:   b.Commitment,
			CastAt:       b.CastAt,
		})
	}

	// Audit root is always over the full set — fetch all commitments
	allBallots, err := s.repo.GetBallotsByPoll(ctx, pollID)
	if err != nil {
		return nil, fmt.Errorf("tally: compute audit root: %w", err)
	}
	commitments := make([]string, len(allBallots))
	for i, b := range allBallots {
		commitments[i] = b.Commitment
	}

	return &models.AuditProof{
		PollID:    pollID,
		Entries:   entries,
		AuditRoot: computeAuditRoot(commitments),
		Total:     total,
		Offset:    offset,
		Limit:     limit,
	}, nil
}

// GetMerkleProof returns an O(log n) Merkle inclusion proof for a single receipt token.
// Leaves are sha256(commitment) sorted lexicographically. The Merkle root is separate
// from the audit_root (which is a flat hash of sorted commitments).
func (s *TallyService) GetMerkleProof(ctx context.Context, pollID uuid.UUID, receiptToken string) (*models.MerkleProof, error) {
	ballot, err := s.repo.GetBallotByReceiptToken(ctx, receiptToken)
	if err != nil {
		return nil, fmt.Errorf("tally: merkle proof: %w", err)
	}
	if ballot == nil || ballot.PollID != pollID {
		return nil, nil
	}

	allBallots, err := s.repo.GetBallotsByPoll(ctx, pollID)
	if err != nil {
		return nil, fmt.Errorf("tally: merkle proof: get ballots: %w", err)
	}

	commitments := make([]string, len(allBallots))
	for i, b := range allBallots {
		commitments[i] = b.Commitment
	}
	sort.Strings(commitments)

	// Build leaves as sha256(commitment)
	leaves := make([]string, len(commitments))
	targetIdx := -1
	for i, c := range commitments {
		leaves[i] = hashHex(c)
		if c == ballot.Commitment {
			targetIdx = i
		}
	}
	if targetIdx == -1 {
		return nil, fmt.Errorf("tally: merkle proof: commitment not found in ballot set")
	}

	merkleRoot, proofPath := buildMerkleProof(leaves, targetIdx)

	return &models.MerkleProof{
		PollID:       pollID,
		ReceiptToken: receiptToken,
		Commitment:   ballot.Commitment,
		LeafIndex:    targetIdx,
		MerkleRoot:   merkleRoot,
		ProofPath:    proofPath,
		LeafCount:    len(leaves),
	}, nil
}

// buildMerkleProof constructs a binary Merkle tree from leaves and returns the
// root and proof path for the leaf at targetIdx.
// Siblings are hashed with sha256(left || right) up the tree.
func buildMerkleProof(leaves []string, targetIdx int) (root string, path []models.MerkleNode) {
	if len(leaves) == 0 {
		return hashHex(""), nil
	}
	if len(leaves) == 1 {
		return leaves[0], nil
	}

	current := make([]string, len(leaves))
	copy(current, leaves)

	idx := targetIdx
	for len(current) > 1 {
		if len(current)%2 != 0 {
			current = append(current, current[len(current)-1]) // duplicate last
		}
		var next []string
		for i := 0; i < len(current); i += 2 {
			parent := hashHex(current[i] + current[i+1])
			next = append(next, parent)

			siblingIdx := idx ^ 1 // XOR toggles last bit: 0↔1, 2↔3, …
			if i == idx || i+1 == idx {
				sibling := current[siblingIdx]
				dir := "right"
				if siblingIdx < idx {
					dir = "left"
				}
				path = append(path, models.MerkleNode{Hash: sibling, Direction: dir})
				idx = idx / 2
			}
		}
		current = next
	}
	return current[0], path
}

func hashHex(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
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

// Package models defines the domain types for the voter registration and polling system.
package models

import (
	"time"

	"github.com/google/uuid"
)

// VoterStatus is the lifecycle state of a registered voter.
type VoterStatus string

const (
	VoterStatusRegistered VoterStatus = "registered"
	VoterStatusSuspended  VoterStatus = "suspended"
)

// Voter is a RealMe-verified person registered to participate in polls.
// FLT is never stored — only a SHA-256 hash of it.
type Voter struct {
	ID           uuid.UUID   `json:"id"`
	FLTHash      string      `json:"-"`          // sha256(FLT), never exposed
	Status       VoterStatus `json:"status"`
	RegisteredAt time.Time   `json:"registeredAt"`
}

// PollStatus tracks the lifecycle of a poll.
type PollStatus string

const (
	PollStatusDraft  PollStatus = "draft"
	PollStatusOpen   PollStatus = "open"
	PollStatusClosed PollStatus = "closed"
)

// Poll is a voting question with multiple options.
type Poll struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Options     []string   `json:"options"`     // ordered list of choices
	BallotType  BallotType `json:"ballotType"`  // "fptp" or "ranked"
	Status      PollStatus `json:"status"`
	PollSalt    string     `json:"-"`           // random hex; used for voter_token derivation, never exposed
	OpensAt     time.Time  `json:"opensAt"`
	ClosesAt    time.Time  `json:"closesAt"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// Ballot is an anonymous vote record. No PII or FLT is stored here.
//
// Security model:
//   - VoterToken = sha256(flt_hash + poll_id + poll_salt) — anonymous per-poll identifier
//   - ReceiptToken = random UUID — voter's personal proof their vote was counted
//   - Commitment = sha256(voter_token + choice_str + receipt_token) — public audit anchor
//     For ranked ballots choice_str is the JSON-encoded rankings array.
//   - UNIQUE(poll_id, voter_token) enforces one vote per voter per poll
type Ballot struct {
	ID           uuid.UUID `json:"id"`
	PollID       uuid.UUID `json:"pollId"`
	VoterToken   string    `json:"-"`            // anonymous, not exposed
	ChoiceIndex  int       `json:"choiceIndex"`  // -1 for ranked ballots
	Rankings     []int     `json:"rankings,omitempty"` // ranked-choice preference order
	ReceiptToken string    `json:"receiptToken"` // returned to voter; used for verification
	Commitment   string    `json:"commitment"`   // public audit hash
	CastAt       time.Time `json:"castAt"`
}

// BallotReceipt is returned to the voter after casting. It contains the receipt
// token needed to verify the vote was included in the public audit.
type BallotReceipt struct {
	ReceiptToken string    `json:"receiptToken"`
	PollID       uuid.UUID `json:"pollId"`
	ChoiceIndex  int       `json:"choiceIndex"`
	CastAt       time.Time `json:"castAt"`
}

// StoredTally is the persisted tally record.
type StoredTally struct {
	PollID     uuid.UUID      `json:"pollId"`
	Counts     map[string]int `json:"counts"`     // choice_index_str -> count
	TotalVotes int            `json:"totalVotes"`
	AuditRoot  string         `json:"auditRoot"`  // sha256 of sorted commitments
	ComputedAt time.Time      `json:"computedAt"`
}

// Tally is the full tally result returned to clients, including poll metadata.
type Tally struct {
	Poll       *Poll          `json:"poll"`
	Counts     map[string]int `json:"counts"`
	TotalVotes int            `json:"totalVotes"`
	AuditRoot  string         `json:"auditRoot"`
	ComputedAt time.Time      `json:"computedAt"`
	IRVResult  *IRVResult     `json:"irvResult,omitempty"` // non-nil for ranked-choice polls
}

// AuditEntry is one entry in the public audit proof list.
type AuditEntry struct {
	ReceiptToken string    `json:"receiptToken"`
	ChoiceIndex  int       `json:"choiceIndex"`
	Rankings     []int     `json:"rankings,omitempty"` // non-nil for ranked-choice ballots
	Commitment   string    `json:"commitment"`
	CastAt       time.Time `json:"castAt"`
}

// AuditProof is a paginated page of the public audit record for a poll.
// Anyone can verify the AuditRoot by fetching all pages, sorting by Commitment,
// and computing SHA-256 of the concatenated sorted commitments.
type AuditProof struct {
	PollID    uuid.UUID    `json:"pollId"`
	Entries   []AuditEntry `json:"entries"`
	AuditRoot string       `json:"auditRoot"`
	Total     int          `json:"total"`    // total ballots in the poll
	Offset    int          `json:"offset"`
	Limit     int          `json:"limit"`
}

// MerkleNode is one step in a Merkle inclusion proof.
type MerkleNode struct {
	Hash      string `json:"hash"`      // hex-encoded SHA-256
	Direction string `json:"direction"` // "left" or "right"
}

// MerkleProof is an O(log n) inclusion proof for a single ballot commitment.
type MerkleProof struct {
	PollID      uuid.UUID    `json:"pollId"`
	ReceiptToken string      `json:"receiptToken"`
	Commitment  string       `json:"commitment"`
	LeafIndex   int          `json:"leafIndex"`
	MerkleRoot  string       `json:"merkleRoot"`
	ProofPath   []MerkleNode `json:"proofPath"`
	LeafCount   int          `json:"leafCount"`
}

// BallotType distinguishes first-past-the-post from ranked-choice polls.
type BallotType string

const (
	BallotTypeFPTP   BallotType = "fptp"
	BallotTypeRanked BallotType = "ranked"
)

// CreatePollRequest is the payload for POST /polls.
type CreatePollRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Options     []string   `json:"options"`
	BallotType  BallotType `json:"ballotType"` // "fptp" (default) or "ranked"
	OpensAt     time.Time  `json:"opensAt"`
	ClosesAt    time.Time  `json:"closesAt"`
}

// CastBallotRequest is the payload for POST /polls/{id}/vote.
// For FPTP polls set ChoiceIndex. For ranked polls set Rankings (ordered
// preference list of choice indices, most preferred first).
type CastBallotRequest struct {
	ChoiceIndex int   `json:"choiceIndex"` // FPTP
	Rankings    []int `json:"rankings"`    // ranked-choice; overrides ChoiceIndex
}

// IRVRound records vote counts at one stage of the instant-runoff count.
type IRVRound struct {
	Counts      map[int]int `json:"counts"`      // candidate index → votes
	Eliminated  []int       `json:"eliminated"`  // candidates eliminated this round
	TotalActive int         `json:"totalActive"` // active ballots
}

// IRVResult is the full outcome of an instant-runoff count.
type IRVResult struct {
	Winner *int       `json:"winner"` // winning choice index; nil on tie
	Rounds []IRVRound `json:"rounds"`
}

// TallyEvent is published over SSE / NATS when a ballot is cast.
type TallyEvent struct {
	PollID     string `json:"pollId"`
	TotalVotes int    `json:"totalVotes"`
}

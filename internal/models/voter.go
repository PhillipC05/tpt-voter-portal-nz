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
	Options     []string   `json:"options"`    // ordered list of choices
	Status      PollStatus `json:"status"`
	PollSalt    string     `json:"-"`          // random hex; used for voter_token derivation, never exposed
	OpensAt     time.Time  `json:"opensAt"`
	ClosesAt    time.Time  `json:"closesAt"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// Ballot is an anonymous vote record. No PII or FLT is stored here.
//
// Security model:
//   - VoterToken = sha256(flt_hash + poll_id + poll_salt) — anonymous per-poll identifier
//   - ReceiptToken = random UUID — voter's personal proof their vote was counted
//   - Commitment = sha256(voter_token + choice_index_str + receipt_token) — public audit anchor
//   - UNIQUE(poll_id, voter_token) enforces one vote per voter per poll
type Ballot struct {
	ID           uuid.UUID `json:"id"`
	PollID       uuid.UUID `json:"pollId"`
	VoterToken   string    `json:"-"`           // anonymous, not exposed
	ChoiceIndex  int       `json:"choiceIndex"`
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
}

// AuditEntry is one entry in the public audit proof list.
type AuditEntry struct {
	ReceiptToken string    `json:"receiptToken"`
	ChoiceIndex  int       `json:"choiceIndex"`
	Commitment   string    `json:"commitment"`
	CastAt       time.Time `json:"castAt"`
}

// AuditProof is the full public audit record for a poll.
// Anyone can verify the AuditRoot by sorting Entries by Commitment and hashing.
type AuditProof struct {
	PollID    uuid.UUID    `json:"pollId"`
	Entries   []AuditEntry `json:"entries"`
	AuditRoot string       `json:"auditRoot"`
	Total     int          `json:"total"`
}

// CreatePollRequest is the payload for POST /polls.
type CreatePollRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Options     []string  `json:"options"`
	OpensAt     time.Time `json:"opensAt"`
	ClosesAt    time.Time `json:"closesAt"`
}

// CastBallotRequest is the payload for POST /polls/{id}/vote.
type CastBallotRequest struct {
	ChoiceIndex int `json:"choiceIndex"`
}

package services

import (
	"testing"
	"time"

	"github.com/tpt-nz/tpt-voter-portal-nz/internal/models"
)

func TestDeriveVoterToken_Deterministic(t *testing.T) {
	fltHash := "abc123flthashabcde"
	pollID := "550e8400-e29b-41d4-a716-446655440000"
	salt := "deadbeefcafe1234"

	tok1 := deriveVoterToken(fltHash, pollID, salt)
	tok2 := deriveVoterToken(fltHash, pollID, salt)

	if tok1 != tok2 {
		t.Errorf("voter token is not deterministic: %q vs %q", tok1, tok2)
	}
	if tok1 == "" {
		t.Error("voter token must not be empty")
	}
}

func TestDeriveVoterToken_DifferentPerPoll(t *testing.T) {
	fltHash := "abc123"
	salt := "salt1"

	tok1 := deriveVoterToken(fltHash, "poll-id-1", salt)
	tok2 := deriveVoterToken(fltHash, "poll-id-2", salt)

	if tok1 == tok2 {
		t.Error("voter tokens must differ across polls for the same voter")
	}
}

func TestDeriveVoterToken_DifferentPerVoter(t *testing.T) {
	pollID := "poll-id-1"
	salt := "salt1"

	tok1 := deriveVoterToken("flt-hash-voter-a", pollID, salt)
	tok2 := deriveVoterToken("flt-hash-voter-b", pollID, salt)

	if tok1 == tok2 {
		t.Error("voter tokens must differ across voters for the same poll")
	}
}

func TestDeriveCommitment_Deterministic(t *testing.T) {
	c1 := deriveCommitment("votertoken", "1", "receipt-token-uuid")
	c2 := deriveCommitment("votertoken", "1", "receipt-token-uuid")

	if c1 != c2 {
		t.Errorf("commitment is not deterministic: %q vs %q", c1, c2)
	}
}

func TestDeriveCommitment_UniquePerChoice(t *testing.T) {
	c0 := deriveCommitment("votertoken", "0", "receipt")
	c1 := deriveCommitment("votertoken", "1", "receipt")

	if c0 == c1 {
		t.Error("commitments for different choices must differ")
	}
}

func TestComputeAuditRoot_EmptySet(t *testing.T) {
	root := computeAuditRoot(nil)
	if root == "" {
		t.Error("audit root for empty set must not be empty string")
	}
}

func TestComputeAuditRoot_OrderIndependent(t *testing.T) {
	c1 := "aaabbbccc"
	c2 := "dddeeefff"

	root1 := computeAuditRoot([]string{c1, c2})
	root2 := computeAuditRoot([]string{c2, c1})

	if root1 != root2 {
		t.Errorf("audit root must be order-independent: %q vs %q", root1, root2)
	}
}

func TestComputeAuditRoot_ChangesWithDifferentInputs(t *testing.T) {
	root1 := computeAuditRoot([]string{"commitment-a"})
	root2 := computeAuditRoot([]string{"commitment-b"})

	if root1 == root2 {
		t.Error("audit root must change when commitments differ")
	}
}

func TestFLTHash_Deterministic(t *testing.T) {
	flt := "some-flt-value-12345"
	h1 := hashFLT(flt)
	h2 := hashFLT(flt)

	if h1 != h2 {
		t.Errorf("FLT hash is not deterministic: %q vs %q", h1, h2)
	}
	if len(h1) != 64 {
		t.Errorf("expected sha256 hex length 64, got %d", len(h1))
	}
}

func TestFLTHash_DifferentFLTs(t *testing.T) {
	h1 := hashFLT("flt-person-a")
	h2 := hashFLT("flt-person-b")

	if h1 == h2 {
		t.Error("different FLTs must produce different hashes")
	}
}

func TestValidateCreatePollRequest(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	farFuture := time.Now().Add(48 * time.Hour)

	tests := []struct {
		name    string
		req     models.CreatePollRequest
		wantErr bool
	}{
		{
			name:    "valid",
			req:     models.CreatePollRequest{Title: "Referendum", Options: []string{"Yes", "No"}, ClosesAt: farFuture},
			wantErr: false,
		},
		{
			name:    "missing title",
			req:     models.CreatePollRequest{Title: "", Options: []string{"Yes", "No"}, ClosesAt: farFuture},
			wantErr: true,
		},
		{
			name:    "only one option",
			req:     models.CreatePollRequest{Title: "Vote", Options: []string{"Yes"}, ClosesAt: farFuture},
			wantErr: true,
		},
		{
			name:    "no options",
			req:     models.CreatePollRequest{Title: "Vote", Options: nil, ClosesAt: farFuture},
			wantErr: true,
		},
		{
			name:    "missing closesAt",
			req:     models.CreatePollRequest{Title: "Vote", Options: []string{"Yes", "No"}},
			wantErr: true,
		},
		{
			name:    "closesAt before opensAt",
			req:     models.CreatePollRequest{Title: "Vote", Options: []string{"Yes", "No"}, OpensAt: farFuture, ClosesAt: future},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req
			err := validateCreatePollRequest(&req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCreatePollRequest() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

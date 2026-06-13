// Package services implements the business logic for voter registration and polling.
package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/tpt-nz/tpt-voter-portal-nz/internal/models"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/repository"
)

// RegistrationService handles voter eligibility and registration.
type RegistrationService struct {
	repo   *repository.VoterRepository
	logger *slog.Logger
}

// NewRegistrationService creates a new registration service.
func NewRegistrationService(repo *repository.VoterRepository, logger *slog.Logger) *RegistrationService {
	return &RegistrationService{
		repo:   repo,
		logger: logger.With("service", "registration"),
	}
}

// UpsertVoter registers a voter identified by their RealMe FLT.
// The FLT is hashed before storage — it is never persisted directly.
// Returns the existing record if the voter is already registered.
func (s *RegistrationService) UpsertVoter(ctx context.Context, flt string) (*models.Voter, error) {
	fltHash := hashFLT(flt)

	voter, err := s.repo.UpsertVoter(ctx, fltHash)
	if err != nil {
		return nil, fmt.Errorf("registration: upsert voter: %w", err)
	}

	s.logger.InfoContext(ctx, "voter registered", "voterID", voter.ID)
	return voter, nil
}

// GetVoterByFLT retrieves a voter by their RealMe FLT (hashed internally).
// Returns nil if the voter has not yet registered.
func (s *RegistrationService) GetVoterByFLT(ctx context.Context, flt string) (*models.Voter, error) {
	fltHash := hashFLT(flt)
	voter, err := s.repo.GetVoterByFLTHash(ctx, fltHash)
	if err != nil {
		return nil, fmt.Errorf("registration: get voter: %w", err)
	}
	return voter, nil
}

// IsEligible returns true if the voter is registered and not suspended.
func (s *RegistrationService) IsEligible(ctx context.Context, flt string) (bool, error) {
	voter, err := s.GetVoterByFLT(ctx, flt)
	if err != nil {
		return false, err
	}
	if voter == nil {
		return false, nil
	}
	return voter.Status == models.VoterStatusRegistered, nil
}

// FLTHash returns the SHA-256 hash of a FLT, for use in voter_token derivation.
// Exposed so poll.go can call it without duplicating the logic.
func FLTHash(flt string) string {
	return hashFLT(flt)
}

// hashFLT computes sha256(flt) as a lowercase hex string.
func hashFLT(flt string) string {
	h := sha256.New()
	h.Write([]byte(flt))
	return hex.EncodeToString(h.Sum(nil))
}

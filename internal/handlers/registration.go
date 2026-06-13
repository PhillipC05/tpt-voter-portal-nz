package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/tpt-nz/tpt-voter-portal-nz/internal/models"
	"github.com/tpt-nz/realme-go"
)

// RegistrationService is the interface the registration handler depends on.
type RegistrationService interface {
	UpsertVoter(ctx context.Context, flt string) (*models.Voter, error)
	GetVoterByFLT(ctx context.Context, flt string) (*models.Voter, error)
	IsEligible(ctx context.Context, flt string) (bool, error)
}

// RegistrationHandler handles voter registration endpoints.
type RegistrationHandler struct {
	svc    RegistrationService
	logger *slog.Logger
}

// NewRegistrationHandler creates a new registration handler.
func NewRegistrationHandler(svc RegistrationService, logger *slog.Logger) *RegistrationHandler {
	return &RegistrationHandler{
		svc:    svc,
		logger: logger.With("handler", "registration"),
	}
}

// Register handles POST /register.
// Requires RealMe Verified identity. Idempotent — safe to call multiple times.
func (h *RegistrationHandler) Register(w http.ResponseWriter, r *http.Request) {
	identity := realme.IdentityFromContext(r.Context())
	if identity == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !identity.IsVerified() {
		respondJSON(w, http.StatusForbidden, map[string]string{"error": "RealMe Verified identity required to register as a voter"})
		return
	}

	voter, err := h.svc.UpsertVoter(r.Context(), identity.FLT)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "voter registration failed", "error", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "registration failed"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":           voter.ID,
		"status":       voter.Status,
		"registeredAt": voter.RegisteredAt,
	})
}

// Status handles GET /register/status.
// Returns whether the current user is registered and eligible to vote.
func (h *RegistrationHandler) Status(w http.ResponseWriter, r *http.Request) {
	identity := realme.IdentityFromContext(r.Context())
	if identity == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	voter, err := h.svc.GetVoterByFLT(r.Context(), identity.FLT)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "get voter status failed", "error", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to retrieve status"})
		return
	}

	if voter == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"registered": false,
			"eligible":   false,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"registered":   true,
		"eligible":     voter.Status == models.VoterStatusRegistered,
		"status":       voter.Status,
		"registeredAt": voter.RegisteredAt,
	})
}

// respondJSON writes a JSON response with the given status code and body.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/models"
	"github.com/tpt-nz/realme-go"
)

// PollService is the interface the poll handler depends on.
type PollService interface {
	CreatePoll(ctx context.Context, req *models.CreatePollRequest) (*models.Poll, error)
	GetActivePolls(ctx context.Context) ([]models.Poll, error)
	GetPollByID(ctx context.Context, id uuid.UUID) (*models.Poll, error)
	CastBallot(ctx context.Context, flt string, pollID uuid.UUID, req *models.CastBallotRequest) (*models.BallotReceipt, error)
	GetVoterReceipt(ctx context.Context, flt string, pollID uuid.UUID) (*models.BallotReceipt, error)
}

// PollHandler handles poll management and voting endpoints.
type PollHandler struct {
	svc     PollService
	regSvc  RegistrationService
	logger  *slog.Logger
}

// NewPollHandler creates a new poll handler.
func NewPollHandler(svc PollService, regSvc RegistrationService, logger *slog.Logger) *PollHandler {
	return &PollHandler{
		svc:    svc,
		regSvc: regSvc,
		logger: logger.With("handler", "poll"),
	}
}

// Create handles POST /polls.
func (h *PollHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreatePollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	poll, err := h.svc.CreatePoll(r.Context(), &req)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "create poll failed", "error", err)
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, poll)
}

// ListActive handles GET /polls.
func (h *PollHandler) ListActive(w http.ResponseWriter, r *http.Request) {
	polls, err := h.svc.GetActivePolls(r.Context())
	if err != nil {
		h.logger.ErrorContext(r.Context(), "list polls failed", "error", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list polls"})
		return
	}

	if polls == nil {
		polls = []models.Poll{}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": polls, "total": len(polls)})
}

// GetByID handles GET /polls/{id}.
func (h *PollHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parsePollID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid poll id"})
		return
	}

	poll, err := h.svc.GetPollByID(r.Context(), id)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "get poll failed", "error", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to retrieve poll"})
		return
	}
	if poll == nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "poll not found"})
		return
	}

	respondJSON(w, http.StatusOK, poll)
}

// CastBallot handles POST /polls/{id}/vote.
// Requires RealMe Verified identity and prior voter registration.
func (h *PollHandler) CastBallot(w http.ResponseWriter, r *http.Request) {
	identity := realme.IdentityFromContext(r.Context())
	if identity == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	id, err := parsePollID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid poll id"})
		return
	}

	var req models.CastBallotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	eligible, err := h.regSvc.IsEligible(r.Context(), identity.FLT)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "eligibility check failed", "error", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "eligibility check failed"})
		return
	}
	if !eligible {
		respondJSON(w, http.StatusForbidden, map[string]string{"error": "you must register as a voter before casting a ballot"})
		return
	}

	receipt, err := h.svc.CastBallot(r.Context(), identity.FLT, id, &req)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "cast ballot failed", "error", err)
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("ballot rejected: %s", err.Error())})
		return
	}

	respondJSON(w, http.StatusCreated, receipt)
}

// MyReceipt handles GET /polls/{id}/my-receipt.
// Returns the voter's receipt for this poll if they have voted, or a 404.
func (h *PollHandler) MyReceipt(w http.ResponseWriter, r *http.Request) {
	identity := realme.IdentityFromContext(r.Context())
	if identity == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	id, err := parsePollID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid poll id"})
		return
	}

	receipt, err := h.svc.GetVoterReceipt(r.Context(), identity.FLT, id)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "get voter receipt failed", "error", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to retrieve receipt"})
		return
	}
	if receipt == nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "no vote found for this poll"})
		return
	}

	respondJSON(w, http.StatusOK, receipt)
}

func parsePollID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "id"))
}

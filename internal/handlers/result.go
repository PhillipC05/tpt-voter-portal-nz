package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/models"
)

// TallyService is the interface the result handler depends on.
type TallyService interface {
	GetTally(ctx context.Context, pollID uuid.UUID) (*models.Tally, error)
	GetAuditProof(ctx context.Context, pollID uuid.UUID) (*models.AuditProof, error)
	VerifyReceipt(ctx context.Context, pollID uuid.UUID, receiptToken string) (bool, *models.AuditEntry, error)
}

// ResultHandler serves public results and audit proof endpoints.
// No authentication is required — public verifiability is a design goal.
type ResultHandler struct {
	svc    TallyService
	logger *slog.Logger
}

// NewResultHandler creates a new result handler.
func NewResultHandler(svc TallyService, logger *slog.Logger) *ResultHandler {
	return &ResultHandler{
		svc:    svc,
		logger: logger.With("handler", "result"),
	}
}

// GetResults handles GET /polls/{id}/results.
// Returns the vote tally and audit root for the given poll.
func (h *ResultHandler) GetResults(w http.ResponseWriter, r *http.Request) {
	id, err := parsePollID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid poll id"})
		return
	}

	tally, err := h.svc.GetTally(r.Context(), id)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "get tally failed", "error", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to retrieve results"})
		return
	}

	respondJSON(w, http.StatusOK, tally)
}

// GetAuditProof handles GET /polls/{id}/audit.
// Returns the complete public ballot list so anyone can independently verify the tally.
// Each entry includes receipt_token, commitment, choice, and timestamp.
func (h *ResultHandler) GetAuditProof(w http.ResponseWriter, r *http.Request) {
	id, err := parsePollID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid poll id"})
		return
	}

	proof, err := h.svc.GetAuditProof(r.Context(), id)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "get audit proof failed", "error", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to retrieve audit proof"})
		return
	}

	respondJSON(w, http.StatusOK, proof)
}

// VerifyReceipt handles GET /polls/{id}/verify?receipt=TOKEN.
// Lets a voter confirm their receipt token appears in the public ballot list.
func (h *ResultHandler) VerifyReceipt(w http.ResponseWriter, r *http.Request) {
	id, err := parsePollID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid poll id"})
		return
	}

	receiptToken := r.URL.Query().Get("receipt")
	if receiptToken == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "receipt query parameter is required"})
		return
	}

	found, entry, err := h.svc.VerifyReceipt(r.Context(), id, receiptToken)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "verify receipt failed", "error", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "verification failed"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"verified": found,
		"entry":    entry,
	})
}

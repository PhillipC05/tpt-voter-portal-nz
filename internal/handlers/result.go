package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/models"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/services"
)

// TallyService is the interface the result handler depends on.
type TallyService interface {
	GetTally(ctx context.Context, pollID uuid.UUID) (*models.Tally, error)
	GetAuditProof(ctx context.Context, pollID uuid.UUID, offset, limit int) (*models.AuditProof, error)
	VerifyReceipt(ctx context.Context, pollID uuid.UUID, receiptToken string) (bool, *models.AuditEntry, error)
	GetMerkleProof(ctx context.Context, pollID uuid.UUID, receiptToken string) (*models.MerkleProof, error)
}

// ResultHandler serves public results and audit proof endpoints.
// No authentication is required — public verifiability is a design goal.
type ResultHandler struct {
	svc    TallyService
	hub    *services.TallyHub
	logger *slog.Logger
}

// NewResultHandler creates a new result handler.
func NewResultHandler(svc TallyService, hub *services.TallyHub, logger *slog.Logger) *ResultHandler {
	return &ResultHandler{
		svc:    svc,
		hub:    hub,
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

// GetAuditProof handles GET /polls/{id}/audit?offset=0&limit=100.
// Returns a paginated page of the public ballot list for independent verification.
func (h *ResultHandler) GetAuditProof(w http.ResponseWriter, r *http.Request) {
	id, err := parsePollID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid poll id"})
		return
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	proof, err := h.svc.GetAuditProof(r.Context(), id, offset, limit)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "get audit proof failed", "error", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to retrieve audit proof"})
		return
	}

	respondJSON(w, http.StatusOK, proof)
}

// GetMerkleProof handles GET /polls/{id}/merkle-proof?receipt=TOKEN.
// Returns an O(log n) Merkle inclusion proof for the given receipt token.
func (h *ResultHandler) GetMerkleProof(w http.ResponseWriter, r *http.Request) {
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

	proof, err := h.svc.GetMerkleProof(r.Context(), id, receiptToken)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "get merkle proof failed", "error", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "merkle proof computation failed"})
		return
	}
	if proof == nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "receipt not found in this poll"})
		return
	}

	respondJSON(w, http.StatusOK, proof)
}

// LiveResults handles GET /polls/{id}/live-results (Server-Sent Events).
// Streams TallyEvent JSON objects to the client whenever a ballot is cast.
// Clients reconnect automatically via the EventSource API.
func (h *ResultHandler) LiveResults(w http.ResponseWriter, r *http.Request) {
	id, err := parsePollID(r)
	if err != nil {
		http.Error(w, "invalid poll id", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	ch := h.hub.Subscribe(r.Context(), id.String())

	// Send a keepalive comment every 25 s to prevent proxy timeouts
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: {\"pollId\":%q,\"totalVotes\":%d}\n\n", ev.PollID, ev.TotalVotes)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
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

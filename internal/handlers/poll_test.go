package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/models"
	"github.com/tpt-nz/realme-go"
)

// withPollID injects a chi route parameter "id" into the request context.
func withPollID(r *http.Request, id string) *http.Request {
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
}

// --- Mock implementations ---

type mockPollService struct {
	createFn       func(ctx context.Context, req *models.CreatePollRequest) (*models.Poll, error)
	listActiveFn   func(ctx context.Context) ([]models.Poll, error)
	getByIDFn      func(ctx context.Context, id uuid.UUID) (*models.Poll, error)
	castBallotFn   func(ctx context.Context, flt string, pollID uuid.UUID, choiceIndex int) (*models.BallotReceipt, error)
	getReceiptFn   func(ctx context.Context, flt string, pollID uuid.UUID) (*models.BallotReceipt, error)
}

func (m *mockPollService) CreatePoll(ctx context.Context, req *models.CreatePollRequest) (*models.Poll, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return &models.Poll{ID: uuid.New(), Title: req.Title, Options: req.Options}, nil
}

func (m *mockPollService) GetActivePolls(ctx context.Context) ([]models.Poll, error) {
	if m.listActiveFn != nil {
		return m.listActiveFn(ctx)
	}
	return []models.Poll{}, nil
}

func (m *mockPollService) GetPollByID(ctx context.Context, id uuid.UUID) (*models.Poll, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockPollService) CastBallot(ctx context.Context, flt string, pollID uuid.UUID, req *models.CastBallotRequest) (*models.BallotReceipt, error) {
	if m.castBallotFn != nil {
		return m.castBallotFn(ctx, flt, pollID, req.ChoiceIndex)
	}
	return &models.BallotReceipt{ReceiptToken: "test-receipt", PollID: pollID, ChoiceIndex: req.ChoiceIndex, CastAt: time.Now()}, nil
}

func (m *mockPollService) GetVoterReceipt(ctx context.Context, flt string, pollID uuid.UUID) (*models.BallotReceipt, error) {
	if m.getReceiptFn != nil {
		return m.getReceiptFn(ctx, flt, pollID)
	}
	return nil, nil
}

type mockRegService struct {
	eligible bool
}

func (m *mockRegService) UpsertVoter(ctx context.Context, flt string) (*models.Voter, error) {
	return &models.Voter{ID: uuid.New(), Status: models.VoterStatusRegistered}, nil
}
func (m *mockRegService) GetVoterByFLT(ctx context.Context, flt string) (*models.Voter, error) {
	return nil, nil
}
func (m *mockRegService) IsEligible(ctx context.Context, flt string) (bool, error) {
	return m.eligible, nil
}

// --- Tests ---

func TestPollHandler_CastBallot_Unauthorized(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewPollHandler(&mockPollService{}, &mockRegService{eligible: true}, logger)

	req := httptest.NewRequest(http.MethodPost, "/polls/"+uuid.New().String()+"/vote",
		bytes.NewBufferString(`{"choiceIndex":0}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.CastBallot(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestPollHandler_CastBallot_NotEligible(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewPollHandler(&mockPollService{}, &mockRegService{eligible: false}, logger)

	pollID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/polls/"+pollID+"/vote",
		bytes.NewBufferString(`{"choiceIndex":0}`))
	req.Header.Set("Content-Type", "application/json")
	req = withIdentity(req)
	req = withPollID(req, pollID)

	w := httptest.NewRecorder()
	h.CastBallot(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d. body: %s", w.Code, w.Body.String())
	}
}

func TestPollHandler_ListActive_Empty(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewPollHandler(&mockPollService{}, &mockRegService{}, logger)

	req := httptest.NewRequest(http.MethodGet, "/polls", nil)
	w := httptest.NewRecorder()
	h.ListActive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["total"] != float64(0) {
		t.Errorf("expected total 0, got %v", resp["total"])
	}
}

func TestPollHandler_GetByID_NotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := &mockPollService{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Poll, error) {
			return nil, nil
		},
	}
	h := NewPollHandler(svc, &mockRegService{}, logger)

	pollID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/polls/"+pollID, nil)
	req = withPollID(req, pollID)
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRegistrationHandler_Register_RequiresAuth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewRegistrationHandler(&mockRegService{}, logger)

	req := httptest.NewRequest(http.MethodPost, "/register", nil)
	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRegistrationHandler_Register_RequiresVerified(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewRegistrationHandler(&mockRegService{}, logger)

	req := httptest.NewRequest(http.MethodPost, "/register", nil)
	req = req.WithContext(realme.IdentityToContext(req.Context(), &realme.Identity{
		FLT:            "test-flt",
		AssuranceLevel: realme.LevelLogin, // login-only, not verified
	}))

	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-verified identity, got %d", w.Code)
	}
}

// withIdentity injects a verified RealMe identity into the request context.
func withIdentity(r *http.Request) *http.Request {
	return r.WithContext(realme.IdentityToContext(r.Context(), &realme.Identity{
		FLT:            "test-flt-12345",
		FullName:       "Test Voter",
		AssuranceLevel: realme.LevelVerified,
	}))
}

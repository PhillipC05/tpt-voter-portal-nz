// Package scheduler runs background jobs that manage poll lifecycle transitions.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/tpt-nz/tpt-voter-portal-nz/internal/repository"
)

// PollScheduler periodically opens draft polls whose opens_at has passed
// and closes open polls whose closes_at has passed.
type PollScheduler struct {
	repo     *repository.VoterRepository
	logger   *slog.Logger
	interval time.Duration
}

// New creates a PollScheduler that runs every interval.
func New(repo *repository.VoterRepository, logger *slog.Logger, interval time.Duration) *PollScheduler {
	return &PollScheduler{
		repo:     repo,
		logger:   logger.With("component", "scheduler"),
		interval: interval,
	}
}

// Run starts the scheduler loop and blocks until ctx is cancelled.
func (s *PollScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.logger.Info("poll scheduler started", "interval", s.interval)
	s.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("poll scheduler stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *PollScheduler) tick(ctx context.Context) {
	opened, err := s.repo.OpenDraftPolls(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "scheduler: open draft polls failed", "error", err)
	} else if opened > 0 {
		s.logger.InfoContext(ctx, "scheduler: opened polls", "count", opened)
	}

	closed, err := s.repo.CloseExpiredPolls(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "scheduler: close expired polls failed", "error", err)
	} else if closed > 0 {
		s.logger.InfoContext(ctx, "scheduler: closed polls", "count", closed)
	}
}

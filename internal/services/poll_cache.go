package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/models"
	"github.com/tpt-nz/tpt-voter-portal-nz/internal/repository"
)

const pollListCacheTTL = 30 * time.Second
const pollListCacheKey = "voter_portal:polls:active"

// PollCacheService wraps the repository with an optional Redis cache for the
// active-polls list. On a cache miss (or if Redis is unavailable) it falls
// through to the database. The cache is intentionally short-lived so that
// scheduler transitions (draft→open, open→closed) surface within seconds.
type PollCacheService struct {
	repo   *repository.VoterRepository
	rdb    *redis.Client // nil if Redis is unavailable
	logger *slog.Logger
}

// NewPollCacheService creates the service. rdb may be nil for cache-disabled mode.
func NewPollCacheService(repo *repository.VoterRepository, rdb *redis.Client, logger *slog.Logger) *PollCacheService {
	return &PollCacheService{repo: repo, rdb: rdb, logger: logger.With("service", "poll_cache")}
}

// GetActivePolls returns the cached active-poll list, falling back to the database.
func (s *PollCacheService) GetActivePolls(ctx context.Context) ([]models.Poll, error) {
	if s.rdb != nil {
		if polls, err := s.fromCache(ctx); err == nil {
			return polls, nil
		}
	}

	polls, err := s.repo.GetActivePolls(ctx)
	if err != nil {
		return nil, err
	}

	if s.rdb != nil {
		s.toCache(ctx, polls)
	}
	return polls, nil
}

// InvalidateCache removes the active-poll list from Redis.
// Call this after any poll status change.
func (s *PollCacheService) InvalidateCache(ctx context.Context) {
	if s.rdb != nil {
		s.rdb.Del(ctx, pollListCacheKey)
	}
}

func (s *PollCacheService) fromCache(ctx context.Context) ([]models.Poll, error) {
	raw, err := s.rdb.Get(ctx, pollListCacheKey).Bytes()
	if err != nil {
		return nil, err
	}
	var polls []models.Poll
	if err := json.Unmarshal(raw, &polls); err != nil {
		return nil, fmt.Errorf("poll_cache: unmarshal: %w", err)
	}
	return polls, nil
}

func (s *PollCacheService) toCache(ctx context.Context, polls []models.Poll) {
	raw, err := json.Marshal(polls)
	if err != nil {
		s.logger.WarnContext(ctx, "poll_cache: marshal failed", "error", err)
		return
	}
	if err := s.rdb.Set(ctx, pollListCacheKey, raw, pollListCacheTTL).Err(); err != nil {
		s.logger.WarnContext(ctx, "poll_cache: set failed", "error", err)
	}
}

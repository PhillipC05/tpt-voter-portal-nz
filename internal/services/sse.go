package services

import (
	"context"
	"sync"

	"github.com/tpt-nz/tpt-voter-portal-nz/internal/models"
)

// TallyHub manages Server-Sent Event subscribers for per-poll live tally updates.
// When a ballot is cast, Publish broadcasts a TallyEvent to all active subscribers
// for that poll. Subscribers are created by Subscribe and removed automatically
// when the client disconnects (context cancellation).
//
// For multi-instance deployments, replace Publish with a NATS publisher and
// run a background subscriber that calls Publish from the NATS message handler.
type TallyHub struct {
	mu   sync.RWMutex
	subs map[string][]chan models.TallyEvent // pollID → subscriber channels
}

// NewTallyHub creates an empty hub.
func NewTallyHub() *TallyHub {
	return &TallyHub{subs: make(map[string][]chan models.TallyEvent)}
}

// Subscribe registers a new subscriber for pollID and returns a read-only channel.
// The channel is closed and the subscription removed when ctx is cancelled.
func (h *TallyHub) Subscribe(ctx context.Context, pollID string) <-chan models.TallyEvent {
	ch := make(chan models.TallyEvent, 4)

	h.mu.Lock()
	h.subs[pollID] = append(h.subs[pollID], ch)
	h.mu.Unlock()

	go func() {
		<-ctx.Done()
		h.remove(pollID, ch)
		close(ch)
	}()

	return ch
}

// Publish sends ev to all active subscribers for the poll. Non-blocking: slow
// consumers are skipped so one stalled browser cannot block the ballot insert path.
func (h *TallyHub) Publish(ev models.TallyEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, ch := range h.subs[ev.PollID] {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (h *TallyHub) remove(pollID string, target chan models.TallyEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	list := h.subs[pollID]
	for i, ch := range list {
		if ch == target {
			h.subs[pollID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	if len(h.subs[pollID]) == 0 {
		delete(h.subs, pollID)
	}
}

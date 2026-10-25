package watcher

import (
	"context"
	"sync"
	"time"
)

// Coalescer aggregates bursts of filesystem events for the same path.
type Coalescer struct {
	debounce time.Duration
	pending  map[string]ChangeEvent
	mu       sync.Mutex
	out      chan ChangeEvent
}

// NewCoalescer creates an event debouncer.
func NewCoalescer(debounce time.Duration) *Coalescer {
	if debounce <= 0 {
		debounce = 80 * time.Millisecond
	}
	return &Coalescer{
		debounce: debounce,
		pending:  make(map[string]ChangeEvent),
		out:      make(chan ChangeEvent, 100),
	}
}

// Push registers a raw event.
func (c *Coalescer) Push(ev ChangeEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prev, exists := c.pending[ev.Path]
	if exists {
		// If created before and now written, keep CREATE
		if prev.Type == ChangeCreate && ev.Type == ChangeWrite {
			ev.Type = ChangeCreate
		}
	}
	c.pending[ev.Path] = ev
}

// Run flushes settled events at the configured interval.
func (c *Coalescer) Run(ctx context.Context) <-chan ChangeEvent {
	ticker := time.NewTicker(c.debounce)
	go func() {
		defer ticker.Stop()
		defer close(c.out)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.mu.Lock()
				if len(c.pending) > 0 {
					for path, ev := range c.pending {
						select {
						case c.out <- ev:
						case <-ctx.Done():
							c.mu.Unlock()
							return
						}
						delete(c.pending, path)
					}
				}
				c.mu.Unlock()
			}
		}
	}()
	return c.out
}

package middleware

import (
	"sync"
	"time"
)

type limiterStore struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string][]time.Time
}

func newLimiterStore(limit int, window time.Duration) *limiterStore {
	return &limiterStore{
		limit:   limit,
		window:  window,
		entries: make(map[string][]time.Time),
	}
}

func (s *limiterStore) allow(key string) (bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-s.window)
	existing := s.entries[key]
	kept := existing[:0]
	for _, at := range existing {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	if len(kept) >= s.limit {
		s.entries[key] = kept
		retry := s.window
		if len(kept) > 0 {
			retry = kept[0].Add(s.window).Sub(now)
		}
		if retry < time.Second {
			retry = time.Second
		}
		return false, retry.Round(time.Second).String()
	}
	s.entries[key] = append(kept, now)
	return true, ""
}

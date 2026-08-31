package smtp

import (
	"sync"
	"time"
)

// RateLimiter spaces deliveries evenly across the configured minute.
type RateLimiter struct {
	mu          sync.Mutex
	interval    time.Duration
	nextAllowed time.Time
}

func NewRateLimiter(perMinute int) *RateLimiter {
	if perMinute < 1 {
		perMinute = 1
	}
	return &RateLimiter{interval: time.Minute / time.Duration(perMinute)}
}

func (r *RateLimiter) Allow(now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.nextAllowed.IsZero() && now.Before(r.nextAllowed) {
		return false
	}
	r.nextAllowed = now.Add(r.interval)
	return true
}

func (r *RateLimiter) NextAvailable(now time.Time) time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.nextAllowed.IsZero() || !now.Before(r.nextAllowed) {
		return now
	}
	return r.nextAllowed
}

func (r *RateLimiter) Wait() {
	for {
		now := time.Now()
		if r.Allow(now) {
			return
		}
		if delay := time.Until(r.NextAvailable(time.Now())); delay > 0 {
			time.Sleep(delay)
		}
	}
}

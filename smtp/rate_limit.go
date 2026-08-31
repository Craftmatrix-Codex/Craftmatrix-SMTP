package smtp

import (
	"sync"
	"time"
)

// RateLimiter is a token bucket for outbound delivery attempts.
type RateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	capacity   float64
	refill     float64
	lastRefill time.Time
}

func NewRateLimiter(perMinute int) *RateLimiter {
	if perMinute < 1 {
		perMinute = 1
	}
	return &RateLimiter{
		tokens:   float64(perMinute),
		capacity: float64(perMinute),
		refill:   float64(perMinute) / 60,
	}
}

func (r *RateLimiter) Allow(now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lastRefill.IsZero() {
		r.lastRefill = now
	} else {
		elapsed := now.Sub(r.lastRefill).Seconds()
		if elapsed > 0 {
			r.tokens += elapsed * r.refill
			if r.tokens > r.capacity {
				r.tokens = r.capacity
			}
			r.lastRefill = now
		}
	}
	if r.tokens < 1 {
		return false
	}
	r.tokens--
	return true
}

func (r *RateLimiter) Wait() {
	for {
		if r.Allow(time.Now()) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

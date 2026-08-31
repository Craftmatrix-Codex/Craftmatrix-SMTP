package smtp

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsConfiguredDeliveriesPerMinute(t *testing.T) {
	limiter := NewRateLimiter(2)
	start := time.Unix(0, 0)

	if !limiter.Allow(start) {
		t.Fatal("expected the first delivery to be available")
	}
	if limiter.Allow(start.Add(29 * time.Second)) {
		t.Fatal("expected the second delivery to wait for the configured interval")
	}
	if !limiter.Allow(start.Add(30 * time.Second)) {
		t.Fatal("expected the second delivery after thirty seconds")
	}
}

func TestRateLimiterNextAvailableAddsCooldown(t *testing.T) {
	limiter := NewRateLimiter(25)
	start := time.Unix(0, 0)
	if !limiter.Allow(start) {
		t.Fatal("expected first delivery")
	}
	if got := limiter.NextAvailable(start); got != start.Add(2400*time.Millisecond) {
		t.Fatalf("next available = %v, want %v", got, start.Add(2400*time.Millisecond))
	}
}

package smtp

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsConfiguredDeliveriesPerMinute(t *testing.T) {
	limiter := NewRateLimiter(2)
	start := time.Unix(0, 0)

	if !limiter.Allow(start) || !limiter.Allow(start) {
		t.Fatal("expected the initial capacity to be available")
	}
	if limiter.Allow(start) {
		t.Fatal("expected the configured per-minute limit to reject the third delivery")
	}
	if !limiter.Allow(start.Add(30 * time.Second)) {
		t.Fatal("expected one token to refill after thirty seconds")
	}
}

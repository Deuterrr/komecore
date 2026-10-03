package ratelimit

import (
	"testing"
	"time"
)

func TestInMemorySlidingWindowLimiter_Allow(t *testing.T) {
	// Window 100ms, max 2 requests
	lim := NewInMemorySlidingWindowLimiter(100*time.Millisecond, 2)
	ip := "192.168.1.1"

	// First request - should be allowed
	if !lim.Allow(ip) {
		t.Error("expected first request to be allowed")
	}

	// Second request - should be allowed
	if !lim.Allow(ip) {
		t.Error("expected second request to be allowed")
	}

	// Third request - should be rate limited (not allowed)
	if lim.Allow(ip) {
		t.Error("expected third request to be rate limited")
	}

	// Wait for window to expire
	time.Sleep(110 * time.Millisecond)

	// Fourth request - should be allowed after expiration
	if !lim.Allow(ip) {
		t.Error("expected request after window expiration to be allowed")
	}
}

func TestInMemorySlidingWindowLimiter_DifferentIPs(t *testing.T) {
	lim := NewInMemorySlidingWindowLimiter(100*time.Millisecond, 1)
	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	if !lim.Allow(ip1) {
		t.Error("expected ip1 request to be allowed")
	}
	if lim.Allow(ip1) {
		t.Error("expected ip1 to be rate limited on second request")
	}

	// ip2 should still be allowed as it is a different IP
	if !lim.Allow(ip2) {
		t.Error("expected ip2 request to be allowed")
	}
}

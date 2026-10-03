package ratelimit

import (
	"sync"
	"time"

	"komecore/pkg/clock"
)

// Limiter defines the contract for rate limiting clients.
type Limiter interface {
	// Allow checks if the client IP is allowed to proceed.
	// Returns true if allowed, false if rate limited.
	Allow(ip string) bool
}

// InMemorySlidingWindowLimiter implements the Limiter interface using
// an in-memory map and a sliding window of request timestamps.
type InMemorySlidingWindowLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	window   time.Duration
	maxLimit int
}

func NewInMemorySlidingWindowLimiter(window time.Duration, maxLimit int) *InMemorySlidingWindowLimiter {
	return &InMemorySlidingWindowLimiter{
		requests: make(map[string][]time.Time),
		window:   window,
		maxLimit: maxLimit,
	}
}

// Allow returns true if the client IP has not exceeded the maxLimit
// requests in the specified sliding window duration.
func (l *InMemorySlidingWindowLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := clock.Now()
	cutoff := now.Add(-l.window)

	// Retrieve current timestamps for the IP
	times := l.requests[ip]
	var validTimes []time.Time
	for _, t := range times {
		if t.After(cutoff) {
			validTimes = append(validTimes, t)
		}
	}

	if len(validTimes) >= l.maxLimit {
		l.requests[ip] = validTimes
		return false
	}

	validTimes = append(validTimes, now)
	l.requests[ip] = validTimes

	return true
}

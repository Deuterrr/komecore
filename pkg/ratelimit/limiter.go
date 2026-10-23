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

const numShards = 32

type shard struct {
	mu       sync.Mutex
	requests map[string][]time.Time
}

// InMemorySlidingWindowLimiter implements the Limiter interface using
// 32 partitioned lock shards to eliminate global mutex contention (MC-C6).
type InMemorySlidingWindowLimiter struct {
	shards   [numShards]*shard
	window   time.Duration
	maxLimit int
}

func NewInMemorySlidingWindowLimiter(window time.Duration, maxLimit int) *InMemorySlidingWindowLimiter {
	lim := &InMemorySlidingWindowLimiter{
		window:   window,
		maxLimit: maxLimit,
	}
	for i := range lim.shards {
		lim.shards[i] = &shard{
			requests: make(map[string][]time.Time),
		}
	}
	return lim
}

const (
	offset32 = 2166136261
	prime32  = 16777619
)

func hashKey(s string) uint32 {
	var h uint32 = offset32
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime32
	}
	return h
}

// Allow returns true if the client IP has not exceeded the maxLimit
// requests in the specified sliding window duration.
func (l *InMemorySlidingWindowLimiter) Allow(ip string) bool {
	shardIdx := hashKey(ip) & (numShards - 1)
	s := l.shards[shardIdx]
	s.mu.Lock()
	defer s.mu.Unlock()

	now := clock.Now()
	cutoff := now.Add(-l.window)

	times := s.requests[ip]
	validTimes := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			validTimes = append(validTimes, t)
		}
	}

	if len(validTimes) >= l.maxLimit {
		s.requests[ip] = validTimes
		return false
	}

	validTimes = append(validTimes, now)
	s.requests[ip] = validTimes

	return true
}

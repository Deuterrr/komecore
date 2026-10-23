package ratelimit

import (
	"fmt"
	"sync"
	"sync/atomic"
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

func TestInMemorySlidingWindowLimiter_ConcurrentAccess(t *testing.T) {
	lim := NewInMemorySlidingWindowLimiter(1*time.Second, 10)
	const numWorkers = 100
	const numDistinctIPs = 50

	var wg sync.WaitGroup
	var allowedCount int32
	var blockedCount int32

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ip := fmt.Sprintf("10.0.%d.%d", workerID%numDistinctIPs, workerID%255)
			if lim.Allow(ip) {
				atomic.AddInt32(&allowedCount, 1)
			} else {
				atomic.AddInt32(&blockedCount, 1)
			}
		}(i)
	}

	wg.Wait()

	total := atomic.LoadInt32(&allowedCount) + atomic.LoadInt32(&blockedCount)
	if total != numWorkers {
		t.Fatalf("expected total requests %d, got %d", numWorkers, total)
	}
}

func TestInMemorySlidingWindowLimiter_Distribution(t *testing.T) {
	lim := NewInMemorySlidingWindowLimiter(1*time.Minute, 100)
	const numIPs = 200

	// Send 1 request for each of 200 distinct IPs
	for i := 0; i < numIPs; i++ {
		ip := fmt.Sprintf("192.168.%d.%d", i/256, i%256)
		lim.Allow(ip)
	}

	// Check how many of the 32 shards received requests
	activeShards := 0
	for i := range lim.shards {
		lim.shards[i].mu.Lock()
		if len(lim.shards[i].requests) > 0 {
			activeShards++
		}
		lim.shards[i].mu.Unlock()
	}

	// With 200 IPs and 32 shards, FNV-1a hash must distribute across virtually all shards (> 25)
	if activeShards < 25 {
		t.Errorf("expected at least 25 active shards, got %d", activeShards)
	}
}

func BenchmarkInMemorySlidingWindowLimiter_Parallel(b *testing.B) {
	lim := NewInMemorySlidingWindowLimiter(1*time.Minute, 1000000)
	var ipSeq int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		id := atomic.AddInt64(&ipSeq, 1)
		ip := fmt.Sprintf("10.0.%d.%d", (id/256)%256, id%256)
		for pb.Next() {
			lim.Allow(ip)
		}
	})
}

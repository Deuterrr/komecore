package bootstrap

import (
	"context"
	"sync"
	"testing"
	"time"
)

type mockJob struct {
	started bool
	mu      sync.Mutex
}

func (m *mockJob) Start(ctx context.Context) {
	m.mu.Lock()
	m.started = true
	m.mu.Unlock()

	<-ctx.Done()
}

func TestScheduler_RegisterAndStart(t *testing.T) {
	s := &Scheduler{}

	mock1 := &mockJob{}
	mock2 := &mockJob{}

	s.Register(mock1)
	s.Register(mock2)

	if len(s.jobs) != 2 {
		t.Fatalf("expected 2 registered jobs, got %d", len(s.jobs))
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	// Allow goroutines time to spin up
	time.Sleep(50 * time.Millisecond)

	mock1.mu.Lock()
	started1 := mock1.started
	mock1.mu.Unlock()

	mock2.mu.Lock()
	started2 := mock2.started
	mock2.mu.Unlock()

	if !started1 || !started2 {
		t.Errorf("expected both jobs to be started, got job1=%v, job2=%v", started1, started2)
	}

	cancel()

	waitCtx, waitCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer waitCancel()

	if err := s.Wait(waitCtx); err != nil {
		t.Errorf("expected clean drain of scheduler jobs, got %v", err)
	}
}

func TestScheduler_NilRegistration(t *testing.T) {
	s := &Scheduler{}
	s.Register(nil)

	if len(s.jobs) != 0 {
		t.Errorf("expected 0 jobs after registering nil, got %d", len(s.jobs))
	}
}

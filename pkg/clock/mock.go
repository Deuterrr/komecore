package clock

import (
	"sync"
	"time"
)

// MockClock is a thread-safe clock implementation for tests.
type MockClock struct {
	mu        sync.RWMutex
	fixedTime time.Time
}

// NewMockClock creates a MockClock with the given fixed time.
func NewMockClock(fixedTime time.Time) *MockClock {
	return &MockClock{
		fixedTime: InAppLocation(fixedTime),
	}
}

func (m *MockClock) Now() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.fixedTime
}

// SetTime sets the current mock time.
func (m *MockClock) SetTime(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fixedTime = InAppLocation(t)
}

// Advance moves the mock time forward by the specified duration.
func (m *MockClock) Advance(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fixedTime = m.fixedTime.Add(d)
}

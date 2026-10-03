package clock

import (
	"sync"
	"time"
	_ "time/tzdata"
)

const (
	LOCATION_NAME      = "Asia/Jakarta"
	ZONE_NAME          = "WIB"
	ZONE_OFFSET_SECOND = 7 * 3600
)

var (
	jakartaLocation *time.Location
	once            sync.Once
	mu              sync.RWMutex
	defaultClock    Clock = RealClock{}
)

// Location returns the application's canonical timezone.
func Location() *time.Location {
	once.Do(func() {
		loc, err := time.LoadLocation(LOCATION_NAME)
		if err != nil {
			loc = time.FixedZone(ZONE_NAME, ZONE_OFFSET_SECOND)
		}
		jakartaLocation = loc
	})
	return jakartaLocation
}

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

// Now returns the current time in Asia/Jakarta timezone.
func (RealClock) Now() time.Time {
	return time.Now().In(Location())
}

// SetDefault replaces the global clock, primarily for tests.
func SetDefault(c Clock) {
	mu.Lock()
	defer mu.Unlock()
	if c != nil {
		defaultClock = c
	}
}

// ResetDefault restores the default RealClock.
func ResetDefault() {
	mu.Lock()
	defer mu.Unlock()
	defaultClock = RealClock{}
}

func GetDefault() Clock {
	mu.RLock()
	defer mu.RUnlock()
	return defaultClock
}

// Now returns the current time using
// the configured application clock.
func Now() time.Time {
	return GetDefault().Now()
}

// InAppLocation normalizes a time to the application's canonical timezone.
func InAppLocation(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.In(Location())
}

// ToUTC converts a time to UTC for external integrations.
func ToUTC(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.UTC()
}

func BeginningOfDay(t time.Time) time.Time {
	t = InAppLocation(t)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, Location())
}

func EndOfDay(t time.Time) time.Time {
	t = InAppLocation(t)
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, Location())
}

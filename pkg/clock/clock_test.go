package clock_test

import (
	"testing"
	"time"

	"komecore/pkg/clock"
)

func TestLocation(t *testing.T) {
	loc := clock.Location()
	if loc == nil {
		t.Fatal("expected location not to be nil")
	}

	now := clock.Now()
	if now.Location().String() != clock.LOCATION_NAME && now.Location().String() != clock.ZONE_NAME {
		t.Errorf("expected timezone %s or %s, got %s", clock.LOCATION_NAME, clock.ZONE_NAME, now.Location().String())
	}
}

func TestInAppLocation(t *testing.T) {
	utcTime := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	wibTime := clock.InAppLocation(utcTime)

	if wibTime.Hour() != 19 {
		t.Errorf("expected hour 19 in WIB (12 UTC + 7), got %d", wibTime.Hour())
	}
}

func TestMockClock(t *testing.T) {
	initial := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	mc := clock.NewMockClock(initial)

	if mc.Now().Hour() != 17 { // 10 UTC = 17 WIB
		t.Errorf("expected initial hour 17 WIB, got %d", mc.Now().Hour())
	}

	mc.Advance(2 * time.Hour)
	if mc.Now().Hour() != 19 {
		t.Errorf("expected advanced hour 19 WIB, got %d", mc.Now().Hour())
	}
}

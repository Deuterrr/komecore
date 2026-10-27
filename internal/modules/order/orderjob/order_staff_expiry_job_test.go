package orderjob

import (
	"context"
	"testing"
	"time"

	"komecore/internal/modules/order"
)

func TestOrderStaffExpiryJob_NilLogger_DoesNotPanic(t *testing.T) {
	// Initialize job with nil logger
	job := NewOrderStaffExpiryJob(&order.Service{}, 10*time.Millisecond, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	// Starting with nil logger should not panic
	done := make(chan struct{})
	go func() {
		defer close(done)
		job.Start(ctx)
	}()

	select {
	case <-done:
		// success: did not panic
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for job to stop")
	}
}

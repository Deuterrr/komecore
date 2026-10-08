package orderdomain

import (
	"testing"
	"time"

	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

func TestOrder_Confirm(t *testing.T) {
	now := appclock.Now()
	order := Order{
		ID:     uuid.New(),
		Status: OrderStatusPending,
	}

	err := order.Confirm(now, DefaultHandlingSLAWindow)
	if err != nil {
		t.Fatalf("unexpected error on Confirm: %v", err)
	}

	if order.Status != OrderStatusConfirmed {
		t.Errorf("expected status %s, got %s", OrderStatusConfirmed, order.Status)
	}
	if order.ConfirmedAt == nil || !order.ConfirmedAt.Equal(now) {
		t.Errorf("expected ConfirmedAt %v, got %v", now, order.ConfirmedAt)
	}
	expectedExpiresAt := now.Add(DefaultHandlingSLAWindow)
	if order.HandlingExpiresAt == nil || !order.HandlingExpiresAt.Equal(expectedExpiresAt) {
		t.Errorf("expected HandlingExpiresAt %v, got %v", expectedExpiresAt, order.HandlingExpiresAt)
	}
}

func TestOrder_Validate(t *testing.T) {
	tests := []struct {
		name    string
		order   Order
		wantErr bool
	}{
		{
			name: "pending order without SLA timestamps is valid",
			order: Order{
				Status: OrderStatusPending,
			},
			wantErr: false,
		},
		{
			name: "confirmed order without SLA timestamps is invalid",
			order: Order{
				Status: OrderStatusConfirmed,
			},
			wantErr: true,
		},
		{
			name: "processing order without SLA timestamps is invalid",
			order: Order{
				Status: OrderStatusProcessing,
			},
			wantErr: true,
		},
		{
			name: "confirmed order with SLA timestamps is valid",
			order: func() Order {
				now := appclock.Now()
				expires := now.Add(72 * time.Hour)
				return Order{
					Status:            OrderStatusConfirmed,
					ConfirmedAt:       &now,
					HandlingExpiresAt: &expires,
				}
			}(),
			wantErr: false,
		},
		{
			name: "expired order without SLA timestamps is valid",
			order: Order{
				Status: OrderStatusExpired,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.order.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

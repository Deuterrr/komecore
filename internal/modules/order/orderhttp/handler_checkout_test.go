package orderhttp

import (
	"testing"

	"github.com/google/uuid"
)

func TestParseCheckoutInput_CartItemWithoutProductID(t *testing.T) {
	h := &orderHandler{}

	shopID := uuid.New().String()
	cartItemID := uuid.New().String()

	req := checkoutCalculateRequest{
		Shops: []checkoutShopRequest{
			{
				ShopID: shopID,
				Items: []checkoutItemRequest{
					{
						CartItemID: &cartItemID,
						ProductID:  nil,
						Quantity:   1,
					},
				},
			},
		},
	}

	input, err := h.parseCheckoutInput(req)
	if err != nil {
		t.Fatalf("expected parseCheckoutInput to succeed for cart item, got error: %v", err)
	}

	if len(input.ShopInput) != 1 {
		t.Fatalf("expected 1 shop input, got %d", len(input.ShopInput))
	}

	item := input.ShopInput[0].Items[0]
	if item.CartItemID == nil || item.CartItemID.String() != cartItemID {
		t.Errorf("expected CartItemID %s, got %v", cartItemID, item.CartItemID)
	}
}

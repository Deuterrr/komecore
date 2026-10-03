package http

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestParseCheckoutInput_CartItemWithoutProductID(t *testing.T) {
	h := &CartHandler{}

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

func TestUpdateItemRequest_UnmarshalOptions(t *testing.T) {
	body := `{"quantity": 3, "item_options": {"size": "large", "color": "black"}}`
	var req updateItemRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unexpected error unmarshaling updateItemRequest: %v", err)
	}
	if req.Quantity != 3 {
		t.Errorf("expected quantity 3, got %d", req.Quantity)
	}
	if req.ItemOptions["size"] != "large" || req.ItemOptions["color"] != "black" {
		t.Errorf("expected size large and color black, got %+v", req.ItemOptions)
	}
}

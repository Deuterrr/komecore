package carthttp

import (
	"encoding/json"
	"testing"
)

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

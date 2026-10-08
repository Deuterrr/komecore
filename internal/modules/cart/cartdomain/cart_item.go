package cartdomain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// CartItem represents one line in a customer's cart.
type CartItem struct {
	ID          uuid.UUID
	ProductID   uuid.UUID
	ShopID      uuid.UUID
	Quantity    int
	ItemOptions ItemOptions
	DeletedAt   *time.Time
}

// ItemOptions captures extensible key-value variant specifications stored as JSONB.
type ItemOptions map[string]string

func (o ItemOptions) Normalized() ItemOptions {
	if o == nil {
		return make(ItemOptions)
	}
	norm := make(ItemOptions, len(o))
	for k, v := range o {
		kClean := strings.ToLower(strings.TrimSpace(k))
		vClean := strings.TrimSpace(v)
		if kClean != "" && vClean != "" {
			norm[kClean] = vClean
		}
	}
	return norm
}

func (o ItemOptions) Equals(other ItemOptions) bool {
	n1 := o.Normalized()
	n2 := other.Normalized()
	if len(n1) != len(n2) {
		return false
	}
	for k, v := range n1 {
		if n2[k] != v {
			return false
		}
	}
	return true
}

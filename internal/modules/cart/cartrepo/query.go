package cartrepo

import "komecore/internal/modules/cart/cartdomain"

type CartWithItems struct {
	*cartdomain.Cart
}

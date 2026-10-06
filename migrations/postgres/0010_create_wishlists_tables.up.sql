CREATE TABLE wishlists (
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (customer_id, product_id)
);

CREATE INDEX idx_wishlists_customer_id ON wishlists(customer_id);

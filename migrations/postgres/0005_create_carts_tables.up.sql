CREATE TABLE carts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    CONSTRAINT fk_carts_customer
        FOREIGN KEY(customer_id)
        REFERENCES customers(id)
        ON DELETE CASCADE
);

CREATE TABLE cart_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cart_id UUID NOT NULL,
    product_id UUID NOT NULL,
    shop_id UUID NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    item_options JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_cart_items_cart
        FOREIGN KEY(cart_id)
        REFERENCES carts(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_cart_items_product
        FOREIGN KEY(product_id)
        REFERENCES products(id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_cart_items_shop
        FOREIGN KEY(shop_id)
        REFERENCES shops(id)
        ON DELETE RESTRICT
);

CREATE UNIQUE INDEX unique_product_with_options_per_cart
ON cart_items(cart_id, product_id, shop_id, item_options)
WHERE deleted_at IS NULL;

CREATE INDEX idx_cart_items_cart_id
ON cart_items(cart_id);

CREATE INDEX idx_cart_items_shop_id
ON cart_items(shop_id);

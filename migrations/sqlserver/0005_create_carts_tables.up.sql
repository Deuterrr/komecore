CREATE TABLE carts (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    customer_id UNIQUEIDENTIFIER NOT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    CONSTRAINT pk_carts PRIMARY KEY (id),
    CONSTRAINT uq_carts_customer_id UNIQUE (customer_id),
    CONSTRAINT fk_carts_customer
        FOREIGN KEY (customer_id)
        REFERENCES customers(id)
        ON DELETE CASCADE
);

CREATE TABLE cart_items (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    cart_id UNIQUEIDENTIFIER NOT NULL,
    product_id UNIQUEIDENTIFIER NOT NULL,
    shop_id UNIQUEIDENTIFIER NOT NULL,
    quantity INTEGER NOT NULL,
    item_options NVARCHAR(MAX) NOT NULL DEFAULT '{}',
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    deleted_at DATETIMEOFFSET,
    CONSTRAINT pk_cart_items PRIMARY KEY (id),
    CONSTRAINT chk_cart_items_quantity CHECK (quantity > 0),
    CONSTRAINT fk_cart_items_cart
        FOREIGN KEY (cart_id)
        REFERENCES carts(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_cart_items_product
        FOREIGN KEY (product_id)
        REFERENCES products(id),
    CONSTRAINT fk_cart_items_shop
        FOREIGN KEY (shop_id)
        REFERENCES shops(id)
);

CREATE UNIQUE INDEX unique_product_with_options_per_cart
ON cart_items(cart_id, product_id, shop_id, item_options)
WHERE deleted_at IS NULL;

CREATE INDEX idx_cart_items_cart_id
ON cart_items(cart_id);

CREATE INDEX idx_cart_items_shop_id
ON cart_items(shop_id);

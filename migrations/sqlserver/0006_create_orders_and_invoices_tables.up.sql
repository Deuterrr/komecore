CREATE TABLE orders (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    number NVARCHAR(50) NOT NULL,
    customer_id UNIQUEIDENTIFIER NOT NULL,
    address_id UNIQUEIDENTIFIER NOT NULL,
    status NVARCHAR(50) NOT NULL,
    subtotal BIGINT NOT NULL,
    shipping_fee BIGINT NOT NULL,
    total BIGINT NOT NULL,
    confirmed_at DATETIMEOFFSET,
    handling_expires_at DATETIMEOFFSET,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    CONSTRAINT pk_orders PRIMARY KEY (id),
    CONSTRAINT uq_orders_number UNIQUE (number),
    CONSTRAINT fk_orders_customer
        FOREIGN KEY (customer_id)
        REFERENCES customers(id),
    CONSTRAINT fk_orders_address
        FOREIGN KEY (address_id)
        REFERENCES customer_addresses(id),
    CONSTRAINT chk_orders_status
        CHECK (status IN ('pending', 'confirmed', 'processing', 'shipped', 'delivered', 'cancelled', 'expired')),
    CONSTRAINT chk_orders_subtotal
        CHECK (subtotal >= 0),
    CONSTRAINT chk_orders_shipping_fee
        CHECK (shipping_fee >= 0),
    CONSTRAINT chk_orders_total
        CHECK (total >= 0),
    CONSTRAINT chk_orders_handling_sla_timestamps
        CHECK (
            (status NOT IN ('confirmed', 'processing')) OR
            (confirmed_at IS NOT NULL AND handling_expires_at IS NOT NULL)
        )
);

CREATE INDEX idx_orders_status_handling_expires_at
    ON orders(status, handling_expires_at)
    WHERE status IN ('confirmed', 'processing');

CREATE TABLE order_items (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    order_id UNIQUEIDENTIFIER NOT NULL,
    shop_id UNIQUEIDENTIFIER NOT NULL,
    shop_name NVARCHAR(255) NOT NULL,
    product_id UNIQUEIDENTIFIER NOT NULL,
    product_name NVARCHAR(255) NOT NULL,
    shipment_id UNIQUEIDENTIFIER,
    courier_code NVARCHAR(100),
    courier_service NVARCHAR(100),
    shipping_fee_total BIGINT NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price BIGINT NOT NULL,
    subtotal BIGINT NOT NULL,
    item_options NVARCHAR(MAX) NOT NULL DEFAULT '{}',
    CONSTRAINT pk_order_items PRIMARY KEY (id),
    CONSTRAINT fk_order_items_order
        FOREIGN KEY (order_id)
        REFERENCES orders(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_order_items_shop
        FOREIGN KEY (shop_id)
        REFERENCES shops(id),
    CONSTRAINT fk_order_items_product
        FOREIGN KEY (product_id)
        REFERENCES products(id),
    CONSTRAINT chk_order_items_quantity
        CHECK (quantity > 0),
    CONSTRAINT chk_order_items_unit_price
        CHECK (unit_price >= 0),
    CONSTRAINT chk_order_items_subtotal
        CHECK (subtotal >= 0)
);

CREATE INDEX idx_order_items_shipment_id ON order_items(shipment_id);

CREATE TABLE invoices (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    number NVARCHAR(50) NOT NULL,
    order_id UNIQUEIDENTIFIER NOT NULL,
    status NVARCHAR(50) NOT NULL,
    subtotal BIGINT NOT NULL,
    shipping_fee BIGINT NOT NULL,
    total BIGINT NOT NULL,
    issued_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_invoices PRIMARY KEY (id),
    CONSTRAINT uq_invoices_number UNIQUE (number),
    CONSTRAINT uq_invoices_order_id UNIQUE (order_id),
    CONSTRAINT fk_invoices_order
        FOREIGN KEY (order_id)
        REFERENCES orders(id)
        ON DELETE CASCADE,
    CONSTRAINT chk_invoices_status
        CHECK (status IN ('issued', 'void')),
    CONSTRAINT chk_invoices_subtotal
        CHECK (subtotal >= 0),
    CONSTRAINT chk_invoices_shipping_fee
        CHECK (shipping_fee >= 0),
    CONSTRAINT chk_invoices_total
        CHECK (total >= 0)
);

CREATE TABLE invoice_items (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    invoice_id UNIQUEIDENTIFIER NOT NULL,
    shop_id UNIQUEIDENTIFIER NOT NULL,
    shop_name NVARCHAR(255) NOT NULL,
    product_id UNIQUEIDENTIFIER NOT NULL,
    product_name NVARCHAR(255) NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price BIGINT NOT NULL,
    subtotal BIGINT NOT NULL,
    courier_code NVARCHAR(100),
    courier_service NVARCHAR(100),
    shipping_fee_total BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT pk_invoice_items PRIMARY KEY (id),
    CONSTRAINT fk_invoice_items_invoice
        FOREIGN KEY (invoice_id)
        REFERENCES invoices(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_invoice_items_shop
        FOREIGN KEY (shop_id)
        REFERENCES shops(id),
    CONSTRAINT fk_invoice_items_product
        FOREIGN KEY (product_id)
        REFERENCES products(id),
    CONSTRAINT chk_invoice_items_quantity
        CHECK (quantity > 0),
    CONSTRAINT chk_invoice_items_unit_price
        CHECK (unit_price >= 0),
    CONSTRAINT chk_invoice_items_subtotal
        CHECK (subtotal >= 0)
);

CREATE TYPE order_status AS ENUM (
    'pending',
    'confirmed',
    'processing',
    'shipped',
    'delivered',
    'cancelled',
    'expired'
);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number VARCHAR(50) UNIQUE NOT NULL,
    customer_id UUID NOT NULL,
    address_id UUID NOT NULL,
    status order_status NOT NULL,
    subtotal BIGINT NOT NULL,
    shipping_fee BIGINT NOT NULL,
    total BIGINT NOT NULL,
    confirmed_at TIMESTAMPTZ,
    handling_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    CONSTRAINT fk_orders_customer
        FOREIGN KEY (customer_id)
        REFERENCES customers(id),
    CONSTRAINT fk_orders_address
        FOREIGN KEY (address_id)
        REFERENCES customer_addresses(id),
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
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL,
    shop_id UUID NOT NULL,
    shop_name VARCHAR(255) NOT NULL,
    product_id UUID NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    shipment_id UUID,
    courier_code VARCHAR(100),
    courier_service VARCHAR(100),
    shipping_fee_total BIGINT NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price BIGINT NOT NULL,
    subtotal BIGINT NOT NULL,
    item_options JSONB NOT NULL DEFAULT '{}',
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

CREATE TYPE invoice_status AS ENUM (
    'issued',
    'void'
);

CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number VARCHAR(50) UNIQUE NOT NULL,
    order_id UUID UNIQUE NOT NULL,
    status invoice_status NOT NULL,
    subtotal BIGINT NOT NULL,
    shipping_fee BIGINT NOT NULL,
    total BIGINT NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_invoices_order
        FOREIGN KEY (order_id)
        REFERENCES orders(id)
        ON DELETE CASCADE,
    CONSTRAINT chk_invoices_subtotal
        CHECK (subtotal >= 0),
    CONSTRAINT chk_invoices_shipping_fee
        CHECK (shipping_fee >= 0),
    CONSTRAINT chk_invoices_total
        CHECK (total >= 0)
);

CREATE TABLE invoice_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL,
    shop_id UUID NOT NULL,
    shop_name VARCHAR(255) NOT NULL,
    product_id UUID NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price BIGINT NOT NULL,
    subtotal BIGINT NOT NULL,
    courier_code VARCHAR(100),
    courier_service VARCHAR(100),
    shipping_fee_total BIGINT NOT NULL DEFAULT 0,
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

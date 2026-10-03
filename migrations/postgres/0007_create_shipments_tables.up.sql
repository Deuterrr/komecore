CREATE TYPE shipment_status_enum AS ENUM (
    'created',
    'packed',
    'labelled',
    'picked_up',
    'in_transit',
    'out_for_delivery',
    'delivered',
    'failed',
    'returned',
    'cancelled'
);

CREATE TYPE fulfillment_method_enum AS ENUM (
    'courier',
    'self_delivery'
);

CREATE TABLE shipments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL UNIQUE,
    status shipment_status_enum NOT NULL DEFAULT 'created',
    tracking_number TEXT UNIQUE,
    fulfillment_method fulfillment_method_enum NOT NULL DEFAULT 'self_delivery',
    courier_name TEXT NOT NULL,
    service TEXT NOT NULL,
    shipping_cost BIGINT NOT NULL,
    weight INTEGER NOT NULL,
    origin_id TEXT NOT NULL,
    destination_id TEXT NOT NULL,
    shipped_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    CONSTRAINT fk_shipments_order
        FOREIGN KEY(order_id)
        REFERENCES orders(id)
        ON DELETE CASCADE
);

CREATE TABLE shipments (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    order_id UNIQUEIDENTIFIER NOT NULL,
    status NVARCHAR(50) NOT NULL DEFAULT 'created',
    tracking_number NVARCHAR(MAX),
    fulfillment_method NVARCHAR(50) NOT NULL DEFAULT 'self_delivery',
    courier_name NVARCHAR(MAX) NOT NULL,
    service NVARCHAR(MAX) NOT NULL,
    shipping_cost BIGINT NOT NULL,
    weight INTEGER NOT NULL,
    origin_id NVARCHAR(MAX) NOT NULL,
    destination_id NVARCHAR(MAX) NOT NULL,
    shipped_at DATETIMEOFFSET,
    delivered_at DATETIMEOFFSET,
    cancelled_at DATETIMEOFFSET,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    CONSTRAINT pk_shipments PRIMARY KEY (id),
    CONSTRAINT uq_shipments_order_id UNIQUE (order_id),
    CONSTRAINT uq_shipments_tracking_number UNIQUE (tracking_number),
    CONSTRAINT fk_shipments_order
        FOREIGN KEY (order_id)
        REFERENCES orders(id)
        ON DELETE CASCADE,
    CONSTRAINT chk_shipments_status
        CHECK (status IN ('created', 'packed', 'labelled', 'picked_up', 'in_transit', 'out_for_delivery', 'delivered', 'failed', 'returned', 'cancelled')),
    CONSTRAINT chk_shipments_fulfillment_method
        CHECK (fulfillment_method IN ('courier', 'self_delivery'))
);

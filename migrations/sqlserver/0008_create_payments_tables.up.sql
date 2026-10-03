CREATE TABLE payment_methods (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    name NVARCHAR(MAX) NOT NULL,
    code NVARCHAR(MAX) NOT NULL,
    provider NVARCHAR(MAX) NOT NULL,
    type NVARCHAR(50) NOT NULL,
    is_active BIT NOT NULL DEFAULT 1,
    description NVARCHAR(MAX),
    fee_type NVARCHAR(50) NOT NULL,
    fee_amount NUMERIC(15, 2) DEFAULT 0,
    fee_rate NUMERIC(5, 4) DEFAULT 0,
    fee_max NUMERIC(15, 2) DEFAULT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    deleted_at DATETIMEOFFSET,
    CONSTRAINT pk_payment_methods PRIMARY KEY (id),
    CONSTRAINT payment_methods_code_provider_type_key
        UNIQUE (code, provider, type),
    CONSTRAINT chk_payment_methods_type
        CHECK (type IN ('bank_transfer', 'ewallet', 'qr_code')),
    CONSTRAINT chk_payment_methods_fee_type
        CHECK (fee_type IN ('flat', 'percentage', 'mixed')),
    CONSTRAINT payment_methods_fee_check
        CHECK (
            (fee_type = 'flat' AND fee_rate = 0)
         OR (fee_type = 'percentage' AND fee_amount = 0)
         OR (fee_type = 'mixed')
        ),
    CONSTRAINT payment_methods_fee_amount_non_negative
        CHECK (fee_amount >= 0),
    CONSTRAINT payment_methods_fee_rate_range
        CHECK (fee_rate >= 0 AND fee_rate <= 1),
    CONSTRAINT payment_methods_fee_max_non_negative
        CHECK (fee_max IS NULL OR fee_max >= 0)
);

CREATE TABLE payment_instructions (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    payment_method_id UNIQUEIDENTIFIER NOT NULL,
    content NVARCHAR(MAX) NOT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_payment_instructions PRIMARY KEY (id),
    CONSTRAINT uq_payment_instructions_method_id UNIQUE (payment_method_id),
    CONSTRAINT fk_payment_instructions_payment_method
        FOREIGN KEY (payment_method_id)
        REFERENCES payment_methods(id)
        ON DELETE CASCADE
);

CREATE TABLE payments (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    order_id UNIQUEIDENTIFIER NOT NULL,
    method_id UNIQUEIDENTIFIER NOT NULL,
    provider NVARCHAR(MAX) NOT NULL,
    provider_payment_id NVARCHAR(MAX),
    provider_order_id NVARCHAR(MAX),
    amount BIGINT NOT NULL,
    status NVARCHAR(50) NOT NULL DEFAULT 'pending',
    expires_at DATETIMEOFFSET,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    paid_at DATETIMEOFFSET,
    CONSTRAINT pk_payments PRIMARY KEY (id),
    CONSTRAINT fk_payments_order
        FOREIGN KEY (order_id)
        REFERENCES orders(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_payments_method
        FOREIGN KEY (method_id)
        REFERENCES payment_methods(id),
    CONSTRAINT chk_payments_status
        CHECK (status IN ('pending', 'paid', 'failed', 'expired', 'cancelled', 'refund_pending', 'refund_failed', 'refunded'))
);

CREATE INDEX idx_payments_order_id ON payments(order_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_provider_payment_id ON payments(provider_payment_id);

CREATE TABLE payment_events (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    payment_id UNIQUEIDENTIFIER NOT NULL,
    event_name NVARCHAR(MAX) NOT NULL,
    payload NVARCHAR(MAX) NOT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_payment_events PRIMARY KEY (id),
    CONSTRAINT fk_payment_events_payment
        FOREIGN KEY (payment_id)
        REFERENCES payments(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_payment_events_payment_id ON payment_events(payment_id);
CREATE INDEX idx_payment_events_event_name ON payment_events(event_name);

CREATE TABLE payment_channel_data (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    payment_id UNIQUEIDENTIFIER NOT NULL,
    channel_type NVARCHAR(50) NOT NULL,
    display_name NVARCHAR(MAX) NOT NULL,
    account_number NVARCHAR(MAX),
    qr_string NVARCHAR(MAX),
    redirect_url NVARCHAR(MAX),
    metadata NVARCHAR(MAX) DEFAULT '{}',
    action_url NVARCHAR(MAX),
    expires_at DATETIMEOFFSET,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_payment_channel_data PRIMARY KEY (id),
    CONSTRAINT uq_payment_channel_data_payment_id UNIQUE (payment_id),
    CONSTRAINT fk_payment_channel_data_payment
        FOREIGN KEY (payment_id)
        REFERENCES payments(id)
        ON DELETE CASCADE,
    CONSTRAINT chk_payment_channel_data_channel_type
        CHECK (channel_type IN ('bank_transfer', 'ewallet', 'qr_code'))
);

CREATE INDEX idx_payment_channel_data_payment_id ON payment_channel_data(payment_id);

CREATE TABLE payment_webhook_events (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    gateway_order_id NVARCHAR(MAX) NOT NULL,
    transaction_status NVARCHAR(MAX) NOT NULL,
    payload NVARCHAR(MAX) NOT NULL,
    status NVARCHAR(MAX) NOT NULL DEFAULT 'received',
    error NVARCHAR(MAX),
    received_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    processed_at DATETIMEOFFSET,
    CONSTRAINT pk_payment_webhook_events PRIMARY KEY (id),
    CONSTRAINT uq_payment_webhook_events_order_status
        UNIQUE (gateway_order_id, transaction_status)
);

CREATE INDEX idx_pwe_status ON payment_webhook_events(status);
CREATE INDEX idx_pwe_gateway_order_id ON payment_webhook_events(gateway_order_id);

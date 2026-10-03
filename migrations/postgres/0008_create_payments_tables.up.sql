CREATE TYPE fee_type AS ENUM ('flat', 'percentage', 'mixed');
CREATE TYPE method_type AS ENUM ('bank_transfer', 'ewallet', 'qr_code');

CREATE TABLE payment_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    provider TEXT NOT NULL,
    type method_type NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    description TEXT,
    fee_type fee_type NOT NULL,
    fee_amount NUMERIC(15,2) DEFAULT 0,
    fee_rate NUMERIC(5,4) DEFAULT 0,
    fee_max NUMERIC(15,2) DEFAULT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT payment_methods_code_provider_type_key
        UNIQUE (code, provider, type),
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
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_method_id UUID UNIQUE NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_payment_instructions_payment_method
        FOREIGN KEY (payment_method_id)
        REFERENCES payment_methods(id)
        ON DELETE CASCADE
);

CREATE TYPE payment_status AS ENUM (
    'pending',
    'paid',
    'failed',
    'expired',
    'cancelled',
    'refund_pending',
    'refund_failed',
    'refunded'
);

CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL,
    method_id UUID NOT NULL,
    provider TEXT NOT NULL,
    provider_payment_id TEXT,
    provider_order_id TEXT,
    amount BIGINT NOT NULL,
    status payment_status NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    CONSTRAINT fk_payments_order
        FOREIGN KEY (order_id)
        REFERENCES orders(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_payments_method
        FOREIGN KEY (method_id)
        REFERENCES payment_methods(id)
);

CREATE INDEX idx_payments_order_id ON payments(order_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_provider_payment_id ON payments(provider_payment_id);

CREATE TABLE payment_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL,
    event_name TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_payment_events_payment
        FOREIGN KEY (payment_id)
        REFERENCES payments(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_payment_events_payment_id ON payment_events(payment_id);
CREATE INDEX idx_payment_events_event_name ON payment_events(event_name);

CREATE TABLE payment_channel_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID UNIQUE NOT NULL,
    channel_type method_type NOT NULL,
    display_name TEXT NOT NULL,
    account_number TEXT,
    qr_string TEXT,
    redirect_url TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    action_url TEXT,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_payment_channel_data_payment
        FOREIGN KEY (payment_id)
        REFERENCES payments(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_payment_channel_data_payment_id ON payment_channel_data(payment_id);

CREATE TABLE payment_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gateway_order_id TEXT NOT NULL,
    transaction_status TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'received',
    error TEXT,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    CONSTRAINT uq_payment_webhook_events_order_status
        UNIQUE (gateway_order_id, transaction_status)
);

CREATE INDEX idx_pwe_status ON payment_webhook_events(status);
CREATE INDEX idx_pwe_gateway_order_id ON payment_webhook_events(gateway_order_id);

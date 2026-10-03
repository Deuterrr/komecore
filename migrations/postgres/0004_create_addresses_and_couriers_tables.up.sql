CREATE TABLE customer_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    recipient_name TEXT NOT NULL,
    phone TEXT NOT NULL,
    is_default BOOLEAN NOT NULL,
    province TEXT NOT NULL,
    city TEXT NOT NULL,
    district TEXT NOT NULL,
    village TEXT NOT NULL,
    full_address TEXT NOT NULL,
    postal_code TEXT NOT NULL,
    latitude NUMERIC(10, 7),
    longitude NUMERIC(10, 7),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_customer_addresses_customer
        FOREIGN KEY(customer_id)
        REFERENCES customers(id)
        ON DELETE CASCADE,
    CONSTRAINT chk_customer_addresses_latitude
        CHECK (latitude IS NULL OR (latitude >= -90.0 AND latitude <= 90.0)),
    CONSTRAINT chk_customer_addresses_longitude
        CHECK (longitude IS NULL OR (longitude >= -180.0 AND longitude <= 180.0))
);

CREATE UNIQUE INDEX uq_idx_one_default_address_per_customer
ON customer_addresses(customer_id)
WHERE is_default = true;

CREATE TABLE couriers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

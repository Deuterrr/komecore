CREATE TABLE customer_addresses (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    customer_id UNIQUEIDENTIFIER NOT NULL,
    recipient_name NVARCHAR(MAX) NOT NULL,
    phone NVARCHAR(MAX) NOT NULL,
    is_default BIT NOT NULL,
    province NVARCHAR(MAX) NOT NULL,
    city NVARCHAR(MAX) NOT NULL,
    district NVARCHAR(MAX) NOT NULL,
    village NVARCHAR(MAX) NOT NULL,
    full_address NVARCHAR(MAX) NOT NULL,
    postal_code NVARCHAR(MAX) NOT NULL,
    latitude NUMERIC(10, 7),
    longitude NUMERIC(10, 7),
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    deleted_at DATETIMEOFFSET,
    CONSTRAINT pk_customer_addresses PRIMARY KEY (id),
    CONSTRAINT fk_customer_addresses_customer
        FOREIGN KEY (customer_id)
        REFERENCES customers(id)
        ON DELETE CASCADE,
    CONSTRAINT chk_customer_addresses_latitude
        CHECK (latitude IS NULL OR (latitude >= -90.0 AND latitude <= 90.0)),
    CONSTRAINT chk_customer_addresses_longitude
        CHECK (longitude IS NULL OR (longitude >= -180.0 AND longitude <= 180.0))
);

CREATE UNIQUE INDEX uq_idx_one_default_address_per_customer
ON customer_addresses(customer_id)
WHERE is_default = 1;

CREATE TABLE couriers (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    code NVARCHAR(50) NOT NULL,
    name NVARCHAR(100) NOT NULL,
    is_active BIT NOT NULL DEFAULT 1,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    CONSTRAINT pk_couriers PRIMARY KEY (id),
    CONSTRAINT uq_couriers_code UNIQUE (code)
);

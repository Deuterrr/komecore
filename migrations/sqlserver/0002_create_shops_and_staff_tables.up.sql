CREATE TABLE shops (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    name NVARCHAR(MAX) NOT NULL,
    slug NVARCHAR(MAX) NOT NULL,
    description NVARCHAR(MAX),
    is_active BIT NOT NULL,
    approval_status NVARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    deleted_at DATETIMEOFFSET,
    CONSTRAINT pk_shops PRIMARY KEY (id),
    CONSTRAINT uq_shops_slug UNIQUE (slug),
    CONSTRAINT chk_shops_approval_status
        CHECK (approval_status IN ('pending', 'approved', 'rejected'))
);

CREATE TABLE shop_addresses (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    shop_id UNIQUEIDENTIFIER NOT NULL,
    label NVARCHAR(MAX) NOT NULL,
    phone NVARCHAR(MAX) NOT NULL,
    is_active BIT NOT NULL,
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
    CONSTRAINT pk_shop_addresses PRIMARY KEY (id),
    CONSTRAINT fk_shop_addresses_shop
        FOREIGN KEY (shop_id)
        REFERENCES shops(id)
        ON DELETE CASCADE,
    CONSTRAINT chk_shop_addresses_latitude
        CHECK (latitude IS NULL OR (latitude >= -90.0 AND latitude <= 90.0)),
    CONSTRAINT chk_shop_addresses_longitude
        CHECK (longitude IS NULL OR (longitude >= -180.0 AND longitude <= 180.0))
);

CREATE UNIQUE INDEX uq_idx_one_active_address_per_shop
ON shop_addresses(shop_id)
WHERE is_active = 1;

CREATE TABLE roles (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    code NVARCHAR(MAX) NOT NULL,
    name NVARCHAR(MAX) NOT NULL,
    CONSTRAINT pk_roles PRIMARY KEY (id),
    CONSTRAINT uq_roles_code UNIQUE (code)
);

CREATE TABLE staff (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    user_id UNIQUEIDENTIFIER NOT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    deleted_at DATETIMEOFFSET,
    CONSTRAINT pk_staff PRIMARY KEY (id),
    CONSTRAINT uq_staff_user_id UNIQUE (user_id),
    CONSTRAINT fk_staff_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE TABLE staff_memberships (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    staff_id UNIQUEIDENTIFIER NOT NULL,
    account_id UNIQUEIDENTIFIER NOT NULL,
    role_id UNIQUEIDENTIFIER NOT NULL,
    created_by UNIQUEIDENTIFIER NOT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_staff_memberships PRIMARY KEY (id),
    CONSTRAINT fk_staff_memberships_staff
        FOREIGN KEY (staff_id)
        REFERENCES staff(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_staff_memberships_account
        FOREIGN KEY (account_id)
        REFERENCES accounts(id),
    CONSTRAINT fk_staff_memberships_role
        FOREIGN KEY (role_id)
        REFERENCES roles(id),
    CONSTRAINT uq_staff_memberships_account
        UNIQUE (account_id)
);

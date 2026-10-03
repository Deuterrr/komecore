CREATE TYPE shop_approval_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE shops (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,
    is_active BOOLEAN NOT NULL,
    approval_status shop_approval_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE shop_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL,
    label TEXT NOT NULL,
    phone TEXT NOT NULL,
    is_active BOOLEAN NOT NULL,
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
    CONSTRAINT fk_shop_addresses_shop
        FOREIGN KEY(shop_id)
        REFERENCES shops(id)
        ON DELETE CASCADE,
    CONSTRAINT chk_shop_addresses_latitude
        CHECK (latitude IS NULL OR (latitude >= -90.0 AND latitude <= 90.0)),
    CONSTRAINT chk_shop_addresses_longitude
        CHECK (longitude IS NULL OR (longitude >= -180.0 AND longitude <= 180.0))
);

CREATE UNIQUE INDEX uq_idx_one_active_address_per_shop
ON shop_addresses(shop_id)
WHERE is_active = true;

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL
);

CREATE TABLE staff (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_staff_user
        FOREIGN KEY(user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE TABLE staff_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    staff_id UUID NOT NULL,
    account_id UUID NOT NULL,
    role_id UUID NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_staff_memberships_staff
        FOREIGN KEY (staff_id)
        REFERENCES staff(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_staff_memberships_account
        FOREIGN KEY (account_id)
        REFERENCES accounts(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_staff_memberships_role
        FOREIGN KEY (role_id)
        REFERENCES roles(id),
    CONSTRAINT uq_staff_memberships_account
        UNIQUE (account_id)
);

CREATE TYPE product_status AS ENUM ('active', 'inactive', 'archived');

CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,
    status product_status DEFAULT 'active',
    base_price BIGINT NOT NULL,
    weight NUMERIC(10, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    archived_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE product_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL,
    thumbnail_url TEXT NOT NULL,
    preview_url TEXT NOT NULL,
    detail_url TEXT NOT NULL,
    thumbnail_key TEXT NOT NULL,
    preview_key TEXT NOT NULL,
    detail_key TEXT NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    display_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_product_images_product
        FOREIGN KEY (product_id)
        REFERENCES products(id)
        ON DELETE CASCADE,
    CONSTRAINT uq_product_images_display_order
        UNIQUE(product_id, display_order)
);

CREATE TABLE product_performance (
    product_id UUID PRIMARY KEY,
    cost_price BIGINT,
    supplier_lead_time_days INTEGER,
    gross_margin_pct NUMERIC(6, 2),
    view_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    CONSTRAINT fk_product_performance_product
        FOREIGN KEY (product_id)
        REFERENCES products(id)
        ON DELETE CASCADE
);

CREATE TABLE product_stock_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL,
    shop_id UUID NOT NULL,
    available INTEGER NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_psh_product
        FOREIGN KEY (product_id)
        REFERENCES products(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_psh_shop
        FOREIGN KEY (shop_id)
        REFERENCES shops(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_product_stock_history_product_id
    ON product_stock_history(product_id);

CREATE INDEX idx_product_stock_history_recorded_at
    ON product_stock_history(recorded_at DESC);

CREATE TABLE inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL,
    shop_id UUID NOT NULL,
    stock INTEGER NOT NULL DEFAULT 0,
    reserved_stock INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    CONSTRAINT fk_inventory_product
        FOREIGN KEY(product_id)
        REFERENCES products(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_inventory_shop
        FOREIGN KEY(shop_id)
        REFERENCES shops(id)
        ON DELETE CASCADE,
    CONSTRAINT unique_inventory_product_shop
        UNIQUE(product_id, shop_id)
);

CREATE INDEX idx_inventory_shop_id
    ON inventory(shop_id);

CREATE TABLE products (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    sku NVARCHAR(MAX) NOT NULL,
    name NVARCHAR(MAX) NOT NULL,
    slug NVARCHAR(MAX) NOT NULL,
    description NVARCHAR(MAX),
    status NVARCHAR(50) NOT NULL DEFAULT 'active',
    base_price BIGINT NOT NULL,
    weight NUMERIC(10, 2),
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    archived_at DATETIMEOFFSET,
    deleted_at DATETIMEOFFSET,
    CONSTRAINT pk_products PRIMARY KEY (id),
    CONSTRAINT uq_products_sku UNIQUE (sku),
    CONSTRAINT uq_products_slug UNIQUE (slug),
    CONSTRAINT chk_products_status
        CHECK (status IN ('active', 'inactive', 'archived'))
);

CREATE TABLE product_images (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    product_id UNIQUEIDENTIFIER NOT NULL,
    thumbnail_url NVARCHAR(MAX) NOT NULL,
    preview_url NVARCHAR(MAX) NOT NULL,
    detail_url NVARCHAR(MAX) NOT NULL,
    thumbnail_key NVARCHAR(MAX) NOT NULL,
    preview_key NVARCHAR(MAX) NOT NULL,
    detail_key NVARCHAR(MAX) NOT NULL,
    is_primary BIT NOT NULL DEFAULT 0,
    display_order INT NOT NULL DEFAULT 0,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    deleted_at DATETIMEOFFSET,
    CONSTRAINT pk_product_images PRIMARY KEY (id),
    CONSTRAINT fk_product_images_product
        FOREIGN KEY (product_id)
        REFERENCES products(id)
        ON DELETE CASCADE,
    CONSTRAINT uq_product_images_display_order
        UNIQUE (product_id, display_order)
);

CREATE TABLE product_performance (
    product_id UNIQUEIDENTIFIER NOT NULL,
    cost_price BIGINT,
    supplier_lead_time_days INTEGER,
    gross_margin_pct NUMERIC(6, 2),
    view_count BIGINT NOT NULL DEFAULT 0,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    CONSTRAINT pk_product_performance PRIMARY KEY (product_id),
    CONSTRAINT fk_product_performance_product
        FOREIGN KEY (product_id)
        REFERENCES products(id)
        ON DELETE CASCADE
);

CREATE TABLE product_stock_history (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    product_id UNIQUEIDENTIFIER NOT NULL,
    shop_id UNIQUEIDENTIFIER NOT NULL,
    available INTEGER NOT NULL,
    recorded_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_product_stock_history PRIMARY KEY (id),
    CONSTRAINT fk_psh_product
        FOREIGN KEY (product_id)
        REFERENCES products(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_psh_shop
        FOREIGN KEY (shop_id)
        REFERENCES shops(id)
);

CREATE INDEX idx_product_stock_history_product_id
    ON product_stock_history(product_id);

CREATE INDEX idx_product_stock_history_recorded_at
    ON product_stock_history(recorded_at DESC);

CREATE TABLE inventory (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    product_id UNIQUEIDENTIFIER NOT NULL,
    shop_id UNIQUEIDENTIFIER NOT NULL,
    stock INTEGER NOT NULL DEFAULT 0,
    reserved_stock INTEGER NOT NULL DEFAULT 0,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    CONSTRAINT pk_inventory PRIMARY KEY (id),
    CONSTRAINT fk_inventory_product
        FOREIGN KEY (product_id)
        REFERENCES products(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_inventory_shop
        FOREIGN KEY (shop_id)
        REFERENCES shops(id),
    CONSTRAINT unique_inventory_product_shop
        UNIQUE (product_id, shop_id)
);

CREATE INDEX idx_inventory_shop_id
    ON inventory(shop_id);

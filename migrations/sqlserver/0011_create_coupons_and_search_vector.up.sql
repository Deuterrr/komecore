CREATE TABLE coupons (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    code NVARCHAR(50) NOT NULL,
    description NVARCHAR(MAX),
    type NVARCHAR(20) NOT NULL,
    discount_value BIGINT NOT NULL,
    min_spend BIGINT NOT NULL CONSTRAINT df_coupons_min_spend DEFAULT 0,
    max_discount BIGINT,
    quota_total INT NOT NULL,
    quota_remaining INT NOT NULL,
    starts_at DATETIMEOFFSET NOT NULL,
    expires_at DATETIMEOFFSET NOT NULL,
    is_active BIT NOT NULL CONSTRAINT df_coupons_is_active DEFAULT 1,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    CONSTRAINT pk_coupons PRIMARY KEY (id),
    CONSTRAINT uq_coupons_code UNIQUE (code),
    CONSTRAINT chk_coupons_type CHECK (type IN ('percentage', 'fixed_amount', 'free_shipping'))
);

CREATE INDEX idx_coupons_code ON coupons(code);

CREATE TABLE coupon_redemptions (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    coupon_id UNIQUEIDENTIFIER NOT NULL,
    customer_id UNIQUEIDENTIFIER NOT NULL,
    order_id UNIQUEIDENTIFIER NOT NULL,
    discount_amount BIGINT NOT NULL,
    redeemed_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_coupon_redemptions PRIMARY KEY (id),
    CONSTRAINT fk_coupon_redemptions_coupon FOREIGN KEY (coupon_id) REFERENCES coupons(id) ON DELETE CASCADE,
    CONSTRAINT fk_coupon_redemptions_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE,
    CONSTRAINT fk_coupon_redemptions_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    CONSTRAINT uq_coupon_redemption_order UNIQUE (order_id)
);

CREATE INDEX idx_coupon_redemptions_coupon_id ON coupon_redemptions(coupon_id);
CREATE INDEX idx_coupon_redemptions_customer_id ON coupon_redemptions(customer_id);

ALTER TABLE products
    ADD search_vector NVARCHAR(MAX);

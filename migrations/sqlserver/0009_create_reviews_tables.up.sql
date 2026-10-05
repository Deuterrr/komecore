CREATE TABLE product_reviews (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    product_id UNIQUEIDENTIFIER NOT NULL,
    customer_id UNIQUEIDENTIFIER NOT NULL,
    order_id UNIQUEIDENTIFIER NOT NULL,
    rating SMALLINT NOT NULL,
    title NVARCHAR(120),
    comment NVARCHAR(MAX),
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    deleted_at DATETIMEOFFSET,
    CONSTRAINT pk_product_reviews PRIMARY KEY (id),
    CONSTRAINT fk_product_reviews_product FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT fk_product_reviews_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE,
    CONSTRAINT fk_product_reviews_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    CONSTRAINT chk_product_reviews_rating CHECK (rating >= 1 AND rating <= 5)
);

CREATE UNIQUE INDEX uq_review_customer_product_order ON product_reviews(customer_id, product_id, order_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_reviews_product_id ON product_reviews(product_id) WHERE deleted_at IS NULL;

ALTER TABLE products
    ADD average_rating NUMERIC(3, 2) NOT NULL CONSTRAINT df_products_average_rating DEFAULT 0.00,
        review_count INT NOT NULL CONSTRAINT df_products_review_count DEFAULT 0;

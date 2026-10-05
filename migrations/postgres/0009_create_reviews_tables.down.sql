ALTER TABLE products
    DROP COLUMN IF EXISTS review_count,
    DROP COLUMN IF EXISTS average_rating;

DROP TABLE IF EXISTS product_reviews;

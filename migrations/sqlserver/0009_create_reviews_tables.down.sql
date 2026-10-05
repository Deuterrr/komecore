IF COL_LENGTH('products', 'average_rating') IS NOT NULL
BEGIN
    ALTER TABLE products DROP CONSTRAINT df_products_average_rating;
    ALTER TABLE products DROP COLUMN average_rating;
END;

IF COL_LENGTH('products', 'review_count') IS NOT NULL
BEGIN
    ALTER TABLE products DROP CONSTRAINT df_products_review_count;
    ALTER TABLE products DROP COLUMN review_count;
END;

IF OBJECT_ID('product_reviews', 'U') IS NOT NULL DROP TABLE product_reviews;

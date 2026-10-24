-- 1. Insert customer user
IF NOT EXISTS (SELECT 1 FROM users WHERE CAST(username AS NVARCHAR(255)) = 'budisantoso' AND deleted_at IS NULL)
BEGIN
    INSERT INTO users (id, name, username, phone, created_at)
    VALUES (
        'E5F6A7B8-C9D0-4E1F-A2B3-4C5D6E7F8A9B',
        'Budi Santoso',
        'budisantoso',
        '081298765432',
        GETUTCDATE()
    );
END;

-- 2. Insert customer account
IF NOT EXISTS (SELECT 1 FROM accounts WHERE CAST(email AS NVARCHAR(255)) = 'customer@komecore.com' AND deleted_at IS NULL)
BEGIN
    INSERT INTO accounts (id, user_id, email, password, status, type, created_at)
    SELECT
        NEWID(),
        u.id,
        'customer@komecore.com',
        '$2a$10$EDuZ91Yt4FjvVH7qTuTzauYWv01.QlJKvQUHfgZ1SvQrUGvPQPVwa',
        'active',
        'customer',
        GETUTCDATE()
    FROM users u
    WHERE CAST(u.username AS NVARCHAR(255)) = 'budisantoso' AND u.deleted_at IS NULL;
END;

-- 3. Insert customer entity record
IF NOT EXISTS (SELECT 1 FROM customers WHERE id = 'F6A7B8C9-D0E1-4F2A-B3C4-5D6E7F8A9B0C')
BEGIN
    INSERT INTO customers (id, user_id, created_at)
    SELECT
        'F6A7B8C9-D0E1-4F2A-B3C4-5D6E7F8A9B0C',
        u.id,
        GETUTCDATE()
    FROM users u
    WHERE CAST(u.username AS NVARCHAR(255)) = 'budisantoso' AND u.deleted_at IS NULL;
END;

-- 4. Insert default shipping address for customer
IF NOT EXISTS (
    SELECT 1 FROM customer_addresses ca
    WHERE ca.customer_id = 'F6A7B8C9-D0E1-4F2A-B3C4-5D6E7F8A9B0C'
)
BEGIN
    INSERT INTO customer_addresses (id, customer_id, recipient_name, phone, is_default, province, city, district, village, full_address, postal_code, latitude, longitude, created_at)
    VALUES (
        NEWID(),
        'F6A7B8C9-D0E1-4F2A-B3C4-5D6E7F8A9B0C',
        'Budi Santoso',
        '081298765432',
        1,
        'DKI Jakarta',
        'Jakarta Pusat',
        'Menteng',
        'Gondangdia',
        'Jl. Teuku Umar No. 15',
        '10350',
        -6.1955321,
        106.8312456,
        GETUTCDATE()
    );
END;

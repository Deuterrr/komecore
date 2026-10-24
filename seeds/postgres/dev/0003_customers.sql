-- 1. Insert customer user
INSERT INTO users (id, name, username, phone, created_at)
VALUES (
    'e5f6a7b8-c9d0-4e1f-a2b3-4c5d6e7f8a9b',
    'Budi Santoso',
    'budisantoso',
    '081298765432',
    NOW()
)
ON CONFLICT (username) WHERE deleted_at IS NULL DO NOTHING;

-- 2. Insert customer account
INSERT INTO accounts (id, user_id, email, password, status, type, created_at)
SELECT
    gen_random_uuid(),
    u.id,
    'customer@komecore.com',
    crypt('CustomerSecret123!', gen_salt('bf', 10)),
    'active',
    'customer',
    NOW()
FROM users u
WHERE u.username = 'budisantoso' AND u.deleted_at IS NULL
ON CONFLICT (email) WHERE deleted_at IS NULL DO NOTHING;

-- 3. Insert customer entity record
INSERT INTO customers (id, user_id, created_at)
SELECT
    'f6a7b8c9-d0e1-4f2a-b3c4-5d6e7f8a9b0c',
    u.id,
    NOW()
FROM users u
WHERE u.username = 'budisantoso' AND u.deleted_at IS NULL
ON CONFLICT (user_id) DO NOTHING;

-- 4. Insert default shipping address for customer
INSERT INTO customer_addresses (id, customer_id, recipient_name, phone, is_default, province, city, district, village, full_address, postal_code, latitude, longitude, created_at)
SELECT
    gen_random_uuid(),
    c.id,
    'Budi Santoso',
    '081298765432',
    true,
    'DKI Jakarta',
    'Jakarta Pusat',
    'Menteng',
    'Gondangdia',
    'Jl. Teuku Umar No. 15',
    '10350',
    -6.1955321,
    106.8312456,
    NOW()
FROM customers c
WHERE c.id = 'f6a7b8c9-d0e1-4f2a-b3c4-5d6e7f8a9b0c'
ON CONFLICT DO NOTHING;

-- 1. Insert flagship central store
INSERT INTO shops (id, name, slug, description, is_active, approval_status, created_at)
VALUES (
    'c0000000-0000-0000-0000-000000000001',
    'Kome Central Store',
    'kome-central-store',
    'Official flagship store for premium artisanal rice and specialty agricultural grains.',
    true,
    'approved',
    NOW()
)
ON CONFLICT (slug) DO NOTHING;

-- 2. Insert regional organic specialty shop
INSERT INTO shops (id, name, slug, description, is_active, approval_status, created_at)
VALUES (
    'c0000000-0000-0000-0000-000000000002',
    'Kome Farm Organics',
    'kome-farm-organics',
    'Direct farm-to-table organic grains, heirloom brown rice, and specialty crops.',
    true,
    'approved',
    NOW()
)
ON CONFLICT (slug) DO NOTHING;

-- 3. Insert active fulfillment address for flagship store
INSERT INTO shop_addresses (id, shop_id, label, phone, is_active, province, city, district, village, full_address, postal_code, latitude, longitude, created_at)
SELECT
    'e0000000-0000-0000-0000-000000000001',
    s.id,
    'Central Fulfillment Hub',
    '081234567890',
    true,
    'DKI Jakarta',
    'Jakarta Selatan',
    'Kebayoran Baru',
    'Senayan',
    'Jl. Jenderal Sudirman No. 1',
    '12190',
    -6.2297465,
    106.8074743,
    NOW()
FROM shops s
WHERE s.slug = 'kome-central-store'
ON CONFLICT DO NOTHING;

-- 4. Insert active fulfillment address for organic farm shop
INSERT INTO shop_addresses (id, shop_id, label, phone, is_active, province, city, district, village, full_address, postal_code, latitude, longitude, created_at)
SELECT
    gen_random_uuid(),
    s.id,
    'West Java Farm Warehouse',
    '081298765432',
    true,
    'Jawa Barat',
    'Cianjur',
    'Cianjur',
    'Sukamaju',
    'Jl. Raya Cianjur No. 88',
    '43215',
    -6.8174523,
    107.1396481,
    NOW()
FROM shops s
WHERE s.slug = 'kome-farm-organics'
ON CONFLICT DO NOTHING;

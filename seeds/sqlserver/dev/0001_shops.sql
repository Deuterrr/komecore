-- 1. Insert flagship central store
IF NOT EXISTS (SELECT 1 FROM shops WHERE slug = 'kome-central-store')
BEGIN
    INSERT INTO shops (id, name, slug, description, is_active, approval_status, created_at)
    VALUES (
        'C0000000-0000-0000-0000-000000000001',
        'Kome Central Store',
        'kome-central-store',
        'Official flagship store for premium artisanal rice and specialty agricultural grains.',
        1,
        'approved',
        GETUTCDATE()
    );
END;

-- 2. Insert regional organic specialty shop
IF NOT EXISTS (SELECT 1 FROM shops WHERE slug = 'kome-farm-organics')
BEGIN
    INSERT INTO shops (id, name, slug, description, is_active, approval_status, created_at)
    VALUES (
        'C0000000-0000-0000-0000-000000000002',
        'Kome Farm Organics',
        'kome-farm-organics',
        'Direct farm-to-table organic grains, heirloom brown rice, and specialty crops.',
        1,
        'approved',
        GETUTCDATE()
    );
END;

-- 3. Insert active fulfillment address for flagship store
IF NOT EXISTS (
    SELECT 1 FROM shop_addresses sa
    JOIN shops s ON s.id = sa.shop_id
    WHERE s.slug = 'kome-central-store'
)
BEGIN
    INSERT INTO shop_addresses (id, shop_id, label, phone, is_active, province, city, district, village, full_address, postal_code, latitude, longitude, created_at)
    SELECT
        'E0000000-0000-0000-0000-000000000001',
        s.id,
        'Central Fulfillment Hub',
        '081234567890',
        1,
        'DKI Jakarta',
        'Jakarta Selatan',
        'Kebayoran Baru',
        'Senayan',
        'Jl. Jenderal Sudirman No. 1',
        '12190',
        -6.2297465,
        106.8074743,
        GETUTCDATE()
    FROM shops s
    WHERE s.slug = 'kome-central-store';
END;

-- 4. Insert active fulfillment address for organic farm shop
IF NOT EXISTS (
    SELECT 1 FROM shop_addresses sa
    JOIN shops s ON s.id = sa.shop_id
    WHERE s.slug = 'kome-farm-organics'
)
BEGIN
    INSERT INTO shop_addresses (id, shop_id, label, phone, is_active, province, city, district, village, full_address, postal_code, latitude, longitude, created_at)
    SELECT
        NEWID(),
        s.id,
        'West Java Farm Warehouse',
        '081298765432',
        1,
        'Jawa Barat',
        'Cianjur',
        'Cianjur',
        'Sukamaju',
        'Jl. Raya Cianjur No. 88',
        '43215',
        -6.8174523,
        107.1396481,
        GETUTCDATE()
    FROM shops s
    WHERE s.slug = 'kome-farm-organics';
END;

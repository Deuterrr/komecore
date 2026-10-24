-- Insert catalog products
IF NOT EXISTS (SELECT 1 FROM products WHERE sku = 'PRD-001')
BEGIN
    INSERT INTO products (id, sku, name, slug, description, status, base_price, weight, created_at)
    VALUES (
        'B0000000-0000-0000-0000-000000000001',
        'PRD-001',
        'Organic Japonica Rice 5kg',
        'organic-japonica-rice-5kg',
        'Premium certified organic short-grain Japonica rice, naturally sweet with superior sticky texture ideal for everyday meals, sushi, and donburi.',
        'active',
        75000,
        5000.00,
        GETUTCDATE()
    );
END;

IF NOT EXISTS (SELECT 1 FROM products WHERE sku = 'PRD-002')
BEGIN
    INSERT INTO products (id, sku, name, slug, description, status, base_price, weight, created_at)
    VALUES (
        'B0000000-0000-0000-0000-000000000002',
        'PRD-002',
        'Premium Pandan Wangi Rice 5kg',
        'premium-pandan-wangi-rice-5kg',
        'Authentic Cianjur highland Pandan Wangi long-grain white rice with natural pandan fragrance, tender fluffiness, and unpolished grain integrity.',
        'active',
        85000,
        5000.00,
        GETUTCDATE()
    );
END;

IF NOT EXISTS (SELECT 1 FROM products WHERE sku = 'PRD-003')
BEGIN
    INSERT INTO products (id, sku, name, slug, description, status, base_price, weight, created_at)
    VALUES (
        'B0000000-0000-0000-0000-000000000003',
        'PRD-003',
        'Organic Red Brown Rice 2kg',
        'organic-red-brown-rice-2kg',
        'High-fiber heirloom red brown rice rich in anthocyanins and complex minerals with a nutty, wholesome profile for balanced low-GI diets.',
        'active',
        48000,
        2000.00,
        GETUTCDATE()
    );
END;

IF NOT EXISTS (SELECT 1 FROM products WHERE sku = 'PRD-004')
BEGIN
    INSERT INTO products (id, sku, name, slug, description, status, base_price, weight, created_at)
    VALUES (
        'B0000000-0000-0000-0000-000000000004',
        'PRD-004',
        'Koshihikari Grain Rice 1kg',
        'koshihikari-grain-rice-1kg',
        'Export-grade Koshihikari round-grain rice known for glistening luster, rich umami aroma, and firm elasticity even after cooling.',
        'active',
        65000,
        1000.00,
        GETUTCDATE()
    );
END;

-- Insert inventory stock counts for Kome Central Store
IF NOT EXISTS (
    SELECT 1 FROM inventory i
    JOIN products p ON p.id = i.product_id
    JOIN shops s ON s.id = i.shop_id
    WHERE p.sku = 'PRD-001' AND s.slug = 'kome-central-store'
)
BEGIN
    INSERT INTO inventory (id, product_id, shop_id, stock, reserved_stock, created_at)
    SELECT
        NEWID(),
        p.id,
        s.id,
        100,
        0,
        GETUTCDATE()
    FROM products p
    CROSS JOIN shops s
    WHERE s.slug = 'kome-central-store' AND p.sku IN ('PRD-001', 'PRD-002', 'PRD-003', 'PRD-004');
END;

-- Insert inventory stock counts for Kome Farm Organics
IF NOT EXISTS (
    SELECT 1 FROM inventory i
    JOIN products p ON p.id = i.product_id
    JOIN shops s ON s.id = i.shop_id
    WHERE p.sku = 'PRD-001' AND s.slug = 'kome-farm-organics'
)
BEGIN
    INSERT INTO inventory (id, product_id, shop_id, stock, reserved_stock, created_at)
    SELECT
        NEWID(),
        p.id,
        s.id,
        50,
        0,
        GETUTCDATE()
    FROM products p
    CROSS JOIN shops s
    WHERE s.slug = 'kome-farm-organics' AND p.sku IN ('PRD-001', 'PRD-002', 'PRD-003', 'PRD-004');
END;

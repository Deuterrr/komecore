-- Insert catalog products
INSERT INTO products (id, sku, name, slug, description, status, base_price, weight, created_at)
VALUES
    (
        'b0000000-0000-0000-0000-000000000001',
        'PRD-001',
        'Organic Japonica Rice 5kg',
        'organic-japonica-rice-5kg',
        'Premium certified organic short-grain Japonica rice, naturally sweet with superior sticky texture ideal for everyday meals, sushi, and donburi.',
        'active',
        75000,
        5000.00,
        NOW()
    ),
    (
        'b0000000-0000-0000-0000-000000000002',
        'PRD-002',
        'Premium Pandan Wangi Rice 5kg',
        'premium-pandan-wangi-rice-5kg',
        'Authentic Cianjur highland Pandan Wangi long-grain white rice with natural pandan fragrance, tender fluffiness, and unpolished grain integrity.',
        'active',
        85000,
        5000.00,
        NOW()
    ),
    (
        'b0000000-0000-0000-0000-000000000003',
        'PRD-003',
        'Organic Red Brown Rice 2kg',
        'organic-red-brown-rice-2kg',
        'High-fiber heirloom red brown rice rich in anthocyanins and complex minerals with a nutty, wholesome profile for balanced low-GI diets.',
        'active',
        48000,
        2000.00,
        NOW()
    ),
    (
        'b0000000-0000-0000-0000-000000000004',
        'PRD-004',
        'Koshihikari Grain Rice 1kg',
        'koshihikari-grain-rice-1kg',
        'Export-grade Koshihikari round-grain rice known for glistening luster, rich umami aroma, and firm elasticity even after cooling.',
        'active',
        65000,
        1000.00,
        NOW()
    )
ON CONFLICT (sku) DO NOTHING;

-- Insert inventory stock counts for Kome Central Store
INSERT INTO inventory (id, product_id, shop_id, stock, reserved_stock, created_at)
SELECT
    gen_random_uuid(),
    p.id,
    s.id,
    100,
    0,
    NOW()
FROM products p
CROSS JOIN shops s
WHERE s.slug = 'kome-central-store'
  AND p.sku IN ('PRD-001', 'PRD-002', 'PRD-003', 'PRD-004')
ON CONFLICT (product_id, shop_id) DO NOTHING;

-- Insert inventory stock counts for Kome Farm Organics
INSERT INTO inventory (id, product_id, shop_id, stock, reserved_stock, created_at)
SELECT
    gen_random_uuid(),
    p.id,
    s.id,
    50,
    0,
    NOW()
FROM products p
CROSS JOIN shops s
WHERE s.slug = 'kome-farm-organics'
  AND p.sku IN ('PRD-001', 'PRD-002', 'PRD-003', 'PRD-004')
ON CONFLICT (product_id, shop_id) DO NOTHING;

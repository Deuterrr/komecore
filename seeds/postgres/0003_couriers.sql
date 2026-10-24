INSERT INTO couriers (id, code, name, is_active)
VALUES
    (gen_random_uuid(), 'jne', 'Jalur Nugraha Ekakurir', true),
    (gen_random_uuid(), 'sicepat', 'SiCepat', true),
    (gen_random_uuid(), 'ide', 'IDExpress', true),
    (gen_random_uuid(), 'sap', 'SAP Express', true),
    (gen_random_uuid(), 'ninja', 'Ninja Express', true),
    (gen_random_uuid(), 'jnt', 'J&T Express', true),
    (gen_random_uuid(), 'tiki', 'TIKI', true),
    (gen_random_uuid(), 'wahana', 'Wahana Express', true),
    (gen_random_uuid(), 'pos', 'POS Indonesia', true),
    (gen_random_uuid(), 'sentral', 'Sentral Cargo', true),
    (gen_random_uuid(), 'lion', 'Lion Parcel', true),
    (gen_random_uuid(), 'rex', 'Royal Express Indonesia', true)
ON CONFLICT (code) DO NOTHING;

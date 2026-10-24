IF NOT EXISTS (SELECT 1 FROM couriers WHERE code = 'jne')
    INSERT INTO couriers (id, code, name, is_active) VALUES (NEWID(), 'jne', 'Jalur Nugraha Ekakurir', 1);

IF NOT EXISTS (SELECT 1 FROM couriers WHERE code = 'sicepat')
    INSERT INTO couriers (id, code, name, is_active) VALUES (NEWID(), 'sicepat', 'SiCepat', 1);

IF NOT EXISTS (SELECT 1 FROM couriers WHERE code = 'ide')
    INSERT INTO couriers (id, code, name, is_active) VALUES (NEWID(), 'ide', 'IDExpress', 1);

IF NOT EXISTS (SELECT 1 FROM couriers WHERE code = 'sap')
    INSERT INTO couriers (id, code, name, is_active) VALUES (NEWID(), 'sap', 'SAP Express', 1);

IF NOT EXISTS (SELECT 1 FROM couriers WHERE code = 'ninja')
    INSERT INTO couriers (id, code, name, is_active) VALUES (NEWID(), 'ninja', 'Ninja Express', 1);

IF NOT EXISTS (SELECT 1 FROM couriers WHERE code = 'jnt')
    INSERT INTO couriers (id, code, name, is_active) VALUES (NEWID(), 'jnt', 'J&T Express', 1);

IF NOT EXISTS (SELECT 1 FROM couriers WHERE code = 'tiki')
    INSERT INTO couriers (id, code, name, is_active) VALUES (NEWID(), 'tiki', 'TIKI', 1);

IF NOT EXISTS (SELECT 1 FROM couriers WHERE code = 'wahana')
    INSERT INTO couriers (id, code, name, is_active) VALUES (NEWID(), 'wahana', 'Wahana Express', 1);

IF NOT EXISTS (SELECT 1 FROM couriers WHERE code = 'pos')
    INSERT INTO couriers (id, code, name, is_active) VALUES (NEWID(), 'pos', 'POS Indonesia', 1);

IF NOT EXISTS (SELECT 1 FROM couriers WHERE code = 'sentral')
    INSERT INTO couriers (id, code, name, is_active) VALUES (NEWID(), 'sentral', 'Sentral Cargo', 1);

IF NOT EXISTS (SELECT 1 FROM couriers WHERE code = 'lion')
    INSERT INTO couriers (id, code, name, is_active) VALUES (NEWID(), 'lion', 'Lion Parcel', 1);

IF NOT EXISTS (SELECT 1 FROM couriers WHERE code = 'rex')
    INSERT INTO couriers (id, code, name, is_active) VALUES (NEWID(), 'rex', 'Royal Express Indonesia', 1);

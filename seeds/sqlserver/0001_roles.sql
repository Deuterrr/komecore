IF NOT EXISTS (SELECT 1 FROM roles WHERE code = 'staff_admin')
    INSERT INTO roles (id, code, name) VALUES (NEWID(), 'staff_admin', 'Staff Admin');

IF NOT EXISTS (SELECT 1 FROM roles WHERE code = 'staff')
    INSERT INTO roles (id, code, name) VALUES (NEWID(), 'staff', 'Staff');

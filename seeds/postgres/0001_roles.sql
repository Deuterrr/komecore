INSERT INTO roles (id, code, name)
VALUES
    (gen_random_uuid(), 'staff_admin', 'Staff Admin'),
    (gen_random_uuid(), 'staff', 'Staff')
ON CONFLICT (code) DO NOTHING;

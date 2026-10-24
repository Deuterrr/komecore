-- 1. Insert admin user
INSERT INTO users (id, name, username, created_at)
VALUES (
    gen_random_uuid(),
    'System Administrator',
    'admin',
    NOW()
)
ON CONFLICT (username) WHERE deleted_at IS NULL DO NOTHING;

-- 2. Insert admin account with staff type and bcrypt password
-- Default credentials: admin@komecore.com / AdminSecret123!
INSERT INTO accounts (id, user_id, email, password, status, type, created_at)
SELECT
    gen_random_uuid(),
    u.id,
    'admin@komecore.com',
    crypt('AdminSecret123!', gen_salt('bf', 10)),
    'active',
    'staff',
    NOW()
FROM users u
WHERE u.username = 'admin' AND u.deleted_at IS NULL
ON CONFLICT (email) WHERE deleted_at IS NULL DO NOTHING;

-- 3. Insert staff record
INSERT INTO staff (id, user_id, created_at)
SELECT
    gen_random_uuid(),
    u.id,
    NOW()
FROM users u
WHERE u.username = 'admin' AND u.deleted_at IS NULL
ON CONFLICT (user_id) DO NOTHING;

-- 4. Bind staff membership with staff_admin role
-- Role 'staff_admin' automatically bestows all platform permissions via Actor.IsSuperAdmin
INSERT INTO staff_memberships (id, staff_id, account_id, role_id, created_by, created_at)
SELECT
    gen_random_uuid(),
    s.id,
    a.id,
    r.id,
    a.id,
    NOW()
FROM accounts a
JOIN staff s ON s.user_id = a.user_id
JOIN roles r ON r.code = 'staff_admin'
WHERE a.email = 'admin@komecore.com'
ON CONFLICT (account_id) DO NOTHING;

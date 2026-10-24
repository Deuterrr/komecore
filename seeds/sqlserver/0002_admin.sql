-- 1. Insert admin user if missing
IF NOT EXISTS (SELECT 1 FROM users WHERE CAST(username AS NVARCHAR(255)) = 'admin' AND deleted_at IS NULL)
BEGIN
    INSERT INTO users (id, name, username, created_at)
    VALUES (NEWID(), 'System Administrator', 'admin', GETUTCDATE());
END;

-- 2. Insert admin account with staff type and bcrypt password (AdminSecret123!)
IF NOT EXISTS (SELECT 1 FROM accounts WHERE CAST(email AS NVARCHAR(255)) = 'admin@komecore.com' AND deleted_at IS NULL)
BEGIN
    INSERT INTO accounts (id, user_id, email, password, status, type, created_at)
    SELECT
        NEWID(),
        u.id,
        'admin@komecore.com',
        '$2a$10$wN7KjQk5W6v0M.wA1x/Q5OM1k0yv9VjR6jZq0mE2N8n3L2H/P8N.e',
        'active',
        'staff',
        GETUTCDATE()
    FROM users u
    WHERE CAST(u.username AS NVARCHAR(255)) = 'admin' AND u.deleted_at IS NULL;
END;

-- 3. Insert staff record
IF NOT EXISTS (
    SELECT 1 FROM staff s
    JOIN users u ON u.id = s.user_id
    WHERE CAST(u.username AS NVARCHAR(255)) = 'admin' AND u.deleted_at IS NULL
)
BEGIN
    INSERT INTO staff (id, user_id, created_at)
    SELECT NEWID(), u.id, GETUTCDATE()
    FROM users u
    WHERE CAST(u.username AS NVARCHAR(255)) = 'admin' AND u.deleted_at IS NULL;
END;

-- 4. Bind staff membership with staff_admin role
IF NOT EXISTS (
    SELECT 1 FROM staff_memberships sm
    JOIN accounts a ON a.id = sm.account_id
    WHERE CAST(a.email AS NVARCHAR(255)) = 'admin@komecore.com'
)
BEGIN
    INSERT INTO staff_memberships (id, staff_id, account_id, role_id, created_by, created_at)
    SELECT
        NEWID(),
        s.id,
        a.id,
        r.id,
        a.id,
        GETUTCDATE()
    FROM accounts a
    JOIN staff s ON s.user_id = a.user_id
    JOIN roles r ON r.code = 'staff_admin'
    WHERE CAST(a.email AS NVARCHAR(255)) = 'admin@komecore.com';
END;

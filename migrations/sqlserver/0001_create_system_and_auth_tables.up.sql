CREATE TABLE seed_versions (
    name NVARCHAR(255) NOT NULL,
    version NVARCHAR(100) NOT NULL,
    applied_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_seed_versions PRIMARY KEY (name)
);

CREATE TABLE users (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    name NVARCHAR(MAX) NOT NULL,
    username NVARCHAR(MAX) NOT NULL,
    phone NVARCHAR(MAX),
    avatar_url NVARCHAR(MAX),
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    deleted_at DATETIMEOFFSET,
    CONSTRAINT pk_users PRIMARY KEY (id)
);

CREATE UNIQUE INDEX unique_username_per_users ON users(username)
WHERE deleted_at IS NULL;

CREATE TABLE accounts (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    user_id UNIQUEIDENTIFIER NOT NULL,
    email NVARCHAR(MAX) NOT NULL,
    password NVARCHAR(MAX) NOT NULL,
    status NVARCHAR(50) NOT NULL DEFAULT 'pending',
    type NVARCHAR(50) NOT NULL,
    last_login_at DATETIMEOFFSET,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    deleted_at DATETIMEOFFSET,
    CONSTRAINT pk_accounts PRIMARY KEY (id),
    CONSTRAINT fk_accounts_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,
    CONSTRAINT chk_accounts_status
        CHECK (status IN ('pending', 'active', 'suspended', 'locked')),
    CONSTRAINT chk_accounts_type
        CHECK (type IN ('customer', 'staff'))
);

CREATE UNIQUE INDEX unique_user_per_account ON accounts(user_id)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX unique_email_per_account ON accounts(email)
WHERE deleted_at IS NULL;

CREATE TABLE customers (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    user_id UNIQUEIDENTIFIER NOT NULL,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIMEOFFSET,
    deleted_at DATETIMEOFFSET,
    CONSTRAINT pk_customers PRIMARY KEY (id),
    CONSTRAINT uq_customers_user_id UNIQUE (user_id),
    CONSTRAINT fk_customers_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE TABLE verification_challenges (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    user_id UNIQUEIDENTIFIER,
    type NVARCHAR(50) NOT NULL,
    channel NVARCHAR(50) NOT NULL,
    purpose NVARCHAR(50) NOT NULL,
    target NVARCHAR(MAX) NOT NULL,
    code_hash NVARCHAR(MAX) NOT NULL,
    attempt_count INT NOT NULL DEFAULT 0,
    expires_at DATETIMEOFFSET NOT NULL,
    verified_at DATETIMEOFFSET,
    consumed_at DATETIMEOFFSET,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_verification_challenges PRIMARY KEY (id),
    CONSTRAINT fk_verification_challenges_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,
    CONSTRAINT chk_vc_type CHECK (type IN ('numeric', 'magic_link')),
    CONSTRAINT chk_vc_channel CHECK (channel IN ('email', 'sms')),
    CONSTRAINT chk_vc_purpose CHECK (purpose IN ('register', 'login', 'password_reset', 'email_verification'))
);

CREATE INDEX idx_verification_challenges_user_id ON verification_challenges(user_id);
CREATE INDEX idx_verification_challenges_target ON verification_challenges(target);
CREATE INDEX idx_verification_challenges_purpose ON verification_challenges(purpose);
CREATE INDEX idx_verification_challenges_expires_at ON verification_challenges(expires_at);

CREATE TABLE sessions (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    user_id UNIQUEIDENTIFIER NOT NULL,
    user_agent NVARCHAR(MAX),
    ip_address NVARCHAR(MAX),
    expires_at DATETIMEOFFSET NOT NULL,
    revoked_at DATETIMEOFFSET,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    last_activity_at DATETIMEOFFSET,
    CONSTRAINT pk_sessions PRIMARY KEY (id),
    CONSTRAINT fk_sessions_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE TABLE refresh_tokens (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    session_id UNIQUEIDENTIFIER NOT NULL,
    token_hash NVARCHAR(MAX) NOT NULL,
    expires_at DATETIMEOFFSET NOT NULL,
    revoked_at DATETIMEOFFSET,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_refresh_tokens PRIMARY KEY (id),
    CONSTRAINT uq_refresh_tokens_session_id UNIQUE (session_id),
    CONSTRAINT fk_refresh_tokens_session
        FOREIGN KEY (session_id)
        REFERENCES sessions(id)
        ON DELETE CASCADE
);

CREATE TABLE oauth_connections (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    user_id UNIQUEIDENTIFIER NOT NULL,
    provider NVARCHAR(MAX) NOT NULL,
    subject NVARCHAR(MAX) NOT NULL,
    email NVARCHAR(MAX),
    last_login_at DATETIMEOFFSET,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    deleted_at DATETIMEOFFSET,
    CONSTRAINT pk_oauth_connections PRIMARY KEY (id),
    CONSTRAINT uq_oauth_connections_user_id UNIQUE (user_id),
    CONSTRAINT fk_oauth_connections_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE UNIQUE INDEX unique_provider_subject ON oauth_connections(provider, subject)
WHERE deleted_at IS NULL;

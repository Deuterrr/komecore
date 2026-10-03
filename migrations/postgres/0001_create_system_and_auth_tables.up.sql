CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS seed_versions (
    name TEXT PRIMARY KEY,
    version TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    username TEXT NOT NULL,
    phone TEXT,
    avatar_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX unique_username_per_users ON users(username)
WHERE deleted_at IS NULL;

CREATE TYPE account_status AS ENUM ('pending', 'active', 'suspended', 'locked');
CREATE TYPE account_type AS ENUM ('customer', 'staff');

CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    email TEXT NOT NULL,
    password TEXT NOT NULL,
    status account_status NOT NULL DEFAULT 'pending',
    type account_type NOT NULL,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_accounts_user
        FOREIGN KEY(user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE UNIQUE INDEX unique_user_per_account ON accounts(user_id)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX unique_email_per_account ON accounts(email)
WHERE deleted_at IS NULL;

CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_customers_user
        FOREIGN KEY(user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE TYPE otp_type AS ENUM ('numeric', 'magic_link');
CREATE TYPE otp_channel AS ENUM ('email', 'sms');
CREATE TYPE otp_purpose AS ENUM ('register', 'login', 'password_reset', 'email_verification');

CREATE TABLE verification_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID,
    type otp_type NOT NULL,
    channel otp_channel NOT NULL,
    purpose otp_purpose NOT NULL,
    target TEXT NOT NULL,
    code_hash TEXT NOT NULL,
    attempt_count INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    verified_at TIMESTAMPTZ,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_verification_challenges_user
        FOREIGN KEY(user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_verification_challenges_user_id ON verification_challenges(user_id);
CREATE INDEX idx_verification_challenges_target ON verification_challenges(target);
CREATE INDEX idx_verification_challenges_purpose ON verification_challenges(purpose);
CREATE INDEX idx_verification_challenges_expires_at ON verification_challenges(expires_at);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    user_agent TEXT,
    ip_address TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ,
    CONSTRAINT fk_sessions_user
        FOREIGN KEY(user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL UNIQUE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_refresh_tokens_session
        FOREIGN KEY(session_id)
        REFERENCES sessions(id)
        ON DELETE CASCADE
);

CREATE TABLE oauth_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    provider TEXT NOT NULL,
    subject TEXT NOT NULL,
    email TEXT,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_oauth_connections_user
        FOREIGN KEY(user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE UNIQUE INDEX unique_provider_subject ON oauth_connections(provider, subject)
WHERE deleted_at IS NULL;

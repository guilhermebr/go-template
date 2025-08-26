-- Create account type enum
CREATE TYPE account_type AS ENUM ('user', 'admin', 'super_admin');


CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    auth_provider VARCHAR(50) NOT NULL DEFAULT 'supabase',
    auth_provider_id VARCHAR(255),
    account_type account_type NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(auth_provider, auth_provider_id)
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_auth_provider_id ON users(auth_provider_id);
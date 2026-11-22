CREATE TABLE IF NOT EXISTS users (
    id            text PRIMARY KEY,
    email         text,
    password_hash text,
    role          text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS auth_users_email_idx ON users (email);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         text PRIMARY KEY,
    user_id    text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash text NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS auth_refresh_tokens_hash_idx ON refresh_tokens (token_hash);

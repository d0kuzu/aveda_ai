CREATE TABLE IF NOT EXISTS assistants (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    name text NOT NULL,
    configuration text NOT NULL DEFAULT '',
    api_token text NOT NULL DEFAULT '',
    telegram_bot_token text NOT NULL DEFAULT '',
    type varchar(20) NOT NULL DEFAULT 'api',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ai_assistants_owner ON assistants (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS ai_assistants_api_token ON assistants (api_token) WHERE api_token <> '';

CREATE TABLE IF NOT EXISTS chats (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    assistant_id uuid NOT NULL REFERENCES assistants(id),
    customer_id text,
    message_count integer NOT NULL DEFAULT 0,
    is_end boolean NOT NULL DEFAULT false,
    started_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ai_chats_assistant_created ON chats (assistant_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS ai_chats_customer ON chats (assistant_id, customer_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_chats_assistant_started ON chats (assistant_id, started_at);

CREATE TABLE IF NOT EXISTS messages (
    id bigserial PRIMARY KEY,
    chat_id uuid NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    role varchar(20) NOT NULL,
    content text NOT NULL,
    time timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ai_messages_history ON messages (chat_id, time, id);

CREATE TABLE IF NOT EXISTS twilio_configs (
    assistant_id uuid PRIMARY KEY REFERENCES assistants(id) ON DELETE CASCADE,
    user_id uuid NOT NULL,
    twilio_number text NOT NULL,
    account_sid text NOT NULL,
    auth_token text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS blocked_customers (user_id text PRIMARY KEY);

CREATE INDEX IF NOT EXISTS idx_chats_started_at ON chats (started_at);
CREATE INDEX IF NOT EXISTS idx_chats_assistant_started ON chats (assistant_id, started_at);

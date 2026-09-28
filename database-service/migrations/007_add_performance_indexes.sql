-- Ensure composite index on messages(chat_id, time) exists for fast lookups
CREATE INDEX IF NOT EXISTS idx_messages_chat_id_time ON messages (chat_id, "time" ASC);

-- Index on chats for period metrics queries
CREATE INDEX IF NOT EXISTS idx_chats_started_at ON chats (started_at);
CREATE INDEX IF NOT EXISTS idx_chats_assistant_started ON chats (assistant_id, started_at);

-- Index on chats for followup/unreviewed queries
CREATE INDEX IF NOT EXISTS idx_chats_is_end_is_reviewed ON chats (is_end, is_reviewed) WHERE is_end = false;

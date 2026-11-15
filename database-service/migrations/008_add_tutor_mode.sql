-- Add is_tutor flag to chats table
ALTER TABLE chats ADD COLUMN IF NOT EXISTS is_tutor BOOLEAN NOT NULL DEFAULT FALSE;

-- Create tutor_messages table for tutor mode chat history
CREATE TABLE IF NOT EXISTS tutor_messages (
    id SERIAL PRIMARY KEY,
    chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    "time" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Ensure composite index on tutor_messages(chat_id, time) exists for fast lookups
CREATE INDEX IF NOT EXISTS idx_tutor_messages_chat_id_time ON tutor_messages (chat_id, "time" ASC);

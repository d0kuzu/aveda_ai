-- Targeted Migration: Create twilio_configs table
-- This table stores dynamic Twilio configurations for different assistants.

CREATE TABLE IF NOT EXISTS twilio_configs (
    assistant_id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    twilio_number VARCHAR(20) NOT NULL,
    account_sid VARCHAR(100) NOT NULL,
    auth_token VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add foreign key constraint if assistants table exists
-- ALTER TABLE twilio_configs ADD CONSTRAINT fk_twilio_configs_assistant FOREIGN KEY (assistant_id) REFERENCES assistants(id) ON DELETE CASCADE;

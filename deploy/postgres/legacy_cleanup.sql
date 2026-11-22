-- Run explicitly on a backed-up legacy database to discard the retired data.
-- This maintenance script targets only the original shared schema.
BEGIN;

-- Tutor chats keep their regular messages and use the regular message count.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_attribute
        WHERE attrelid = to_regclass('chats') AND attname = 'is_tutor' AND NOT attisdropped
    ) THEN
        UPDATE chats AS c
        SET message_count = (SELECT count(*) FROM messages AS m WHERE m.chat_id = c.id)
        WHERE c.is_tutor;
    END IF;
END;
$$;

DROP TABLE IF EXISTS tutor_messages, appointments, google_syncs, campuslogins, analytics;
ALTER TABLE IF EXISTS chats
    DROP COLUMN IF EXISTS is_tutor,
    DROP COLUMN IF EXISTS is_tuting,
    DROP COLUMN IF EXISTS tuting_program,
    DROP COLUMN IF EXISTS tutor_count,
    DROP COLUMN IF EXISTS tutor_count_per_day,
    DROP COLUMN IF EXISTS followup_stage,
    DROP COLUMN IF EXISTS is_reviewed,
    DROP COLUMN IF EXISTS is_booked,
    DROP COLUMN IF EXISTS user_id;

ALTER TABLE IF EXISTS users
    DROP COLUMN IF EXISTS telegram_id,
    DROP COLUMN IF EXISTS username,
    DROP COLUMN IF EXISTS first_name,
    DROP COLUMN IF EXISTS last_name;

COMMIT;

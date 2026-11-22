package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

const chatFields = `c.id::text, c.assistant_id::text, COALESCE(c.customer_id, ''), COALESCE(c.message_count, 0),
	COALESCE(c.is_end, false), c.started_at, c.created_at, c.updated_at`

func scanChat(row pgx.Row) (*Chat, error) {
	var chat Chat
	var started, created, updated time.Time
	if err := row.Scan(&chat.Id, &chat.AssistantId, &chat.CustomerId, &chat.MessageCount, &chat.IsEnd,
		&started, &created, &updated); err != nil {
		return nil, err
	}
	chat.StartedAt, chat.CreatedAt, chat.UpdatedAt = timestamp(started), timestamp(created), timestamp(updated)
	return &chat, nil
}

func collectChats(rows pgx.Rows, err error) ([]*Chat, error) {
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Chat, error) { return scanChat(row) })
}

func (s *Store) GetChat(ctx context.Context, chatID string) (*Chat, error) {
	return scanChat(s.pool.QueryRow(ctx, "SELECT "+chatFields+" FROM chats c WHERE c.id = $1", chatID))
}

func (s *Store) GetChatForUser(ctx context.Context, chatID, userID string) (*Chat, error) {
	// The previous schema stored assistants.id as text and chats.assistant_id as uuid.
	return scanChat(s.pool.QueryRow(ctx, "SELECT "+chatFields+` FROM chats c JOIN assistants a
		ON c.assistant_id = a.id::uuid WHERE c.id = $1 AND a.user_id = $2`, chatID, userID))
}

const latestChat = ` FROM chats c WHERE c.assistant_id = $1 AND c.customer_id = $2
	ORDER BY c.started_at DESC, c.id DESC LIMIT 1`

func (s *Store) GetLatestChatByCustomer(ctx context.Context, assistantID, customerID string) (*Chat, error) {
	chat, err := scanChat(s.pool.QueryRow(ctx, "SELECT "+chatFields+latestChat, assistantID, customerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return chat, err
}

func (s *Store) GetChatPagesCountByUserID(ctx context.Context, assistantIDs []string, perPage int32) (int32, error) {
	if perPage < 1 {
		perPage = 10
	}
	var pages int32
	err := s.pool.QueryRow(ctx, `SELECT (count(*) + $2::bigint - 1) / $2::bigint FROM chats
		WHERE assistant_id = ANY($1::text[]::uuid[])`, assistantIDs, perPage).Scan(&pages)
	return pages, err
}

func (s *Store) GetChatPageByUserID(ctx context.Context, assistantIDs []string, page, perPage int32) ([]*Chat, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	offset := int64(page-1) * int64(perPage)
	return collectChats(s.pool.Query(ctx, "SELECT "+chatFields+` FROM chats c
		WHERE c.assistant_id = ANY($1::text[]::uuid[]) ORDER BY c.created_at DESC, c.id DESC LIMIT $2 OFFSET $3`,
		assistantIDs, perPage, offset))
}

func (s *Store) SearchChatsByCustomer(ctx context.Context, assistantIDs []string, search string) ([]*Chat, int32, error) {
	chats, err := collectChats(s.pool.Query(ctx, "SELECT "+chatFields+` FROM chats c
		WHERE c.assistant_id = ANY($1::text[]::uuid[]) AND COALESCE(c.customer_id, '') LIKE '%' || $2 || '%'
		ORDER BY c.created_at DESC, c.id DESC`, assistantIDs, search))
	return chats, int32(len(chats)), err
}

func scanMessage(row pgx.Row) (*Message, error) {
	var msg Message
	var created time.Time
	if err := row.Scan(&msg.Id, &msg.ChatId, &msg.Role, &msg.Content, &created); err != nil {
		return nil, err
	}
	msg.CreatedAt = timestamp(created)
	return &msg, nil
}

const messageFields = `id::text, chat_id::text, role, content, time`

func (s *Store) GetChatMessages(ctx context.Context, chatID string, limit, offset int32) ([]*Message, error) {
	rows, err := s.pool.Query(ctx, "SELECT "+messageFields+` FROM messages WHERE chat_id = $1
		ORDER BY messages.time, messages.id LIMIT CASE WHEN $2::integer < 0 THEN NULL ELSE $2 END OFFSET GREATEST($3::integer, 0)`,
		chatID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Message, error) { return scanMessage(row) })
}

func (s *Store) GetAllChatMessages(ctx context.Context, chatID string) ([]*Message, error) {
	return s.GetChatMessages(ctx, chatID, -1, 0)
}

func (s *Store) SaveMessage(ctx context.Context, chatID, role, content, platform string) (*Message, error) {
	// Updating the parent first serializes appends and works for imported tables without FKs.
	msg, err := scanMessage(s.pool.QueryRow(ctx, `WITH parent AS (
		UPDATE chats SET message_count = COALESCE(message_count, 0) + 1, updated_at = clock_timestamp()
		WHERE id = $1 RETURNING id, updated_at)
		INSERT INTO messages (chat_id, role, content, time, created_at, updated_at)
		SELECT id, $2, $3, updated_at, updated_at, updated_at FROM parent RETURNING `+messageFields, chatID, role, content))
	if err != nil {
		return nil, err
	}
	msg.Platform = platform
	return msg, nil
}

// SaveConversationTurn appends both messages only after a successful model response.
func (s *Store) SaveConversationTurn(ctx context.Context, assistantID, customerID, question, answer string) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// Concurrent first turns for the same customer must choose the same chat.
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1::text || ':' || $2::text, 0))", assistantID, customerID); err != nil {
			return err
		}
		var chatID string
		err := tx.QueryRow(ctx, "SELECT c.id::text"+latestChat, assistantID, customerID).Scan(&chatID)
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, `INSERT INTO chats (id, assistant_id, customer_id, started_at, created_at, updated_at)
				SELECT gen_random_uuid(), id::uuid, $2, now(), now(), now() FROM assistants WHERE id = $1
				RETURNING id::text`, assistantID, customerID).Scan(&chatID)
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `WITH parent AS (
			UPDATE chats SET message_count = COALESCE(message_count, 0) + 2, updated_at = clock_timestamp()
			WHERE id = $1 RETURNING id, updated_at)
			INSERT INTO messages (chat_id, role, content, time, created_at, updated_at)
			SELECT parent.id, entry.role, entry.content, parent.updated_at, parent.updated_at, parent.updated_at
			FROM parent CROSS JOIN (VALUES ('user', $2::text), ('assistant', $3::text)) AS entry(role, content)`,
			chatID, question, answer)
		return err
	})
}

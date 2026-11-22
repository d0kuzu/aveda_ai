package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

const assistantFields = `id::text, user_id::text, COALESCE(name, ''), COALESCE(configuration, ''),
	COALESCE(api_token, ''), COALESCE(telegram_bot_token, ''), COALESCE(type, 'api'), created_at, updated_at`

func scanAssistant(row pgx.Row) (*Assistant, error) {
	var a Assistant
	var created, updated time.Time
	if err := row.Scan(&a.Id, &a.UserId, &a.Name, &a.Configuration, &a.ApiToken,
		&a.TelegramBotToken, &a.Type, &created, &updated); err != nil {
		return nil, err
	}
	a.CreatedAt, a.UpdatedAt = timestamp(created), timestamp(updated)
	return &a, nil
}

// Explicit IDs and timestamps also support assistants imported from the earlier schema.
const insertAssistant = `INSERT INTO assistants
	(id, name, api_token, user_id, telegram_bot_token, type, configuration, created_at, updated_at)
	VALUES (gen_random_uuid(), $1, $2, $3, $4, COALESCE(NULLIF($5, ''), 'api'), '', now(), now())
	RETURNING ` + assistantFields

func (s *Store) CreateAssistant(ctx context.Context, name, token, userID, telegramBotToken, assistantType string) (*Assistant, error) {
	return scanAssistant(s.pool.QueryRow(ctx, insertAssistant, name, token, userID, telegramBotToken, assistantType))
}

func (s *Store) GetAssistant(ctx context.Context, id string) (*Assistant, error) {
	return scanAssistant(s.pool.QueryRow(ctx, "SELECT "+assistantFields+" FROM assistants WHERE id = $1", id))
}

func (s *Store) GetAssistantForUser(ctx context.Context, id, userID string) (*Assistant, error) {
	return scanAssistant(s.pool.QueryRow(ctx, "SELECT "+assistantFields+" FROM assistants WHERE id = $1 AND user_id = $2", id, userID))
}

func (s *Store) GetAssistantByAPIToken(ctx context.Context, token string) (*Assistant, error) {
	if token == "" {
		return nil, ErrNotFound
	}
	return scanAssistant(s.pool.QueryRow(ctx, "SELECT "+assistantFields+" FROM assistants WHERE api_token = $1 AND api_token <> ''", token))
}

func (s *Store) GetAssistantsByUserID(ctx context.Context, userID string) ([]*Assistant, error) {
	rows, err := s.pool.Query(ctx, "SELECT "+assistantFields+" FROM assistants WHERE user_id = $1 ORDER BY created_at, id", userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Assistant, error) { return scanAssistant(row) })
}

func (s *Store) UpdateAssistantForUser(ctx context.Context, id, userID, name, configuration string) (*Assistant, error) {
	return scanAssistant(s.pool.QueryRow(ctx, `UPDATE assistants SET name = $3, configuration = $4, updated_at = now()
		WHERE id = $1 AND user_id = $2 RETURNING `+assistantFields, id, userID, name, configuration))
}

func (s *Store) CreateTwilioAssistant(ctx context.Context, name, userID, number, accountSID, authToken string) (*Assistant, error) {
	var assistant *Assistant
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		assistant, err = scanAssistant(tx.QueryRow(ctx, insertAssistant, name, "", userID, "", "twilio"))
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO twilio_configs
			(assistant_id, user_id, twilio_number, account_sid, auth_token, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, now(), now())`, assistant.Id, userID, number, accountSID, authToken)
		return err
	})
	if err != nil {
		return nil, err
	}
	return assistant, nil
}

func (s *Store) GetTwilioConfig(ctx context.Context, assistantID string) (*TwilioConfig, error) {
	var result TwilioConfig
	err := s.pool.QueryRow(ctx, `SELECT assistant_id::text, user_id::text, twilio_number, account_sid, auth_token
		FROM twilio_configs WHERE assistant_id = $1`, assistantID).Scan(&result.AssistantId, &result.UserId, &result.TwilioNumber, &result.AccountSid, &result.AuthToken)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Store) IsCustomerBlocked(ctx context.Context, customerID string) (bool, error) {
	var blocked bool
	err := s.pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM blocked_customers WHERE user_id = $1)", customerID).Scan(&blocked)
	return blocked, err
}

func (s *Store) BlockCustomer(ctx context.Context, customerID string) error {
	_, err := s.pool.Exec(ctx, "INSERT INTO blocked_customers (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING", customerID)
	return err
}

func (s *Store) GetAllBlockedCustomers(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, "SELECT user_id FROM blocked_customers ORDER BY user_id")
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

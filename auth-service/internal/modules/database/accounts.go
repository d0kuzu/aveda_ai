package database

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           string
	Role         string
	PasswordHash string
}

func (q *Queries) CreateUser(ctx context.Context, email, hash string) (User, error) {
	var user User
	err := q.conn.QueryRow(ctx, `
		INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, 'user', now(), now())
		RETURNING id, role`, uuid.NewString(), email, hash).Scan(&user.ID, &user.Role)
	return user, err
}

func (q *Queries) UserByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := q.conn.QueryRow(ctx, `
		SELECT id, COALESCE(role, ''), COALESCE(password_hash, '')
		FROM users WHERE email = $1`, email).Scan(&user.ID, &user.Role, &user.PasswordHash)
	return user, err
}

func (q *Queries) UserByID(ctx context.Context, id string) (User, error) {
	var user User
	err := q.conn.QueryRow(ctx, "SELECT id, COALESCE(role, '') FROM users WHERE id = $1", id).Scan(&user.ID, &user.Role)
	return user, err
}

func (q *Queries) SaveRefresh(ctx context.Context, userID, digest string, expires time.Time) error {
	_, err := q.conn.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, now())`, uuid.NewString(), userID, digest, expires)
	return err
}

// Older databases store the signed JWT directly in token_hash. A rotation
// consumes all matching legacy rows and stores only the new token's digest.
func (q *Queries) ConsumeRefresh(ctx context.Context, userID, digest, original string) (bool, error) {
	result, err := q.conn.Exec(ctx, `
		DELETE FROM refresh_tokens
		WHERE user_id = $1 AND token_hash IN ($2, $3) AND expires_at > now()`, userID, digest, original)
	return result.RowsAffected() > 0, err
}

func (q *Queries) RevokeRefresh(ctx context.Context, userID, digest, original string) error {
	_, err := q.conn.Exec(ctx, `
		DELETE FROM refresh_tokens WHERE user_id = $1 AND token_hash IN ($2, $3)`, userID, digest, original)
	return err
}

package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

var ErrNotFound = pgx.ErrNoRows

//go:embed migrations/*.sql
var migrations embed.FS

func Open(ctx context.Context, dsn string) (*Store, error) {
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("configure AI database: %w", err)
	}
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error { return migrate(ctx, tx) }); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate AI database: %w", err)
	}
	return &Store{pool: pool}, nil
}

func migrate(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended('ai-service migrations', 0))"); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "CREATE TABLE IF NOT EXISTS ai_schema_migrations (version text PRIMARY KEY)"); err != nil {
		return err
	}
	// embed.FS.ReadDir returns entries sorted by filename.
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		var applied bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM ai_schema_migrations WHERE version = $1)", name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		sql, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO ai_schema_migrations (version) VALUES ($1)", name); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func timestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

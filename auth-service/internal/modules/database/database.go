package database

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

type connection interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Queries struct {
	conn connection
}

type Store struct {
	Queries
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		return migrate(ctx, tx)
	}); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate auth database: %w", err)
	}
	return &Store{Queries: Queries{conn: pool}, pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) InTransaction(ctx context.Context, fn func(*Queries) error) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(&Queries{conn: tx})
	})
}

func migrate(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext('auth-service migrations')::bigint)"); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "CREATE TABLE IF NOT EXISTS auth_schema_migrations (version text PRIMARY KEY)"); err != nil {
		return err
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		version := entry.Name()
		var applied bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM auth_schema_migrations WHERE version = $1)", version).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := migrations.ReadFile("migrations/" + version)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("%s: %w", version, err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO auth_schema_migrations (version) VALUES ($1)", version); err != nil {
			return err
		}
	}
	return nil
}

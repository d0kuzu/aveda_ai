package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"auth-service/internal/config"
	"auth-service/internal/modules/database"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Set AUTH_TEST_DATABASE_URL to an isolated PostgreSQL database.
func TestRegisteredIdentityAndLegacyRefreshRotation(t *testing.T) {
	url := os.Getenv("AUTH_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("AUTH_TEST_DATABASE_URL is unset")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	cfg := &config.Config{
		AccessSecret: "test access", RefreshSecret: "test refresh",
		AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour,
	}
	service := New(db, cfg)
	email := uuid.NewString() + "@example.test"
	pair, err := service.Register(ctx, email, "password")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Exec(ctx, "DELETE FROM users WHERE email = $1", email)
	user, err := db.UserByEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	access, err := jwt.Parse(pair.Access, func(*jwt.Token) (any, error) { return []byte(cfg.AccessSecret), nil })
	if err != nil {
		t.Fatal(err)
	}
	if subject, _ := access.Claims.GetSubject(); subject != user.ID {
		t.Fatalf("registered subject %q differs from stored user %q", subject, user.ID)
	}

	// Historic clients could issue the same token twice within one second.
	expiry := time.Now().Add(time.Hour)
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID, "exp": expiry.Unix(), "type": "refresh",
	}).SignedString([]byte(cfg.RefreshSecret))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := db.SaveRefresh(ctx, user.ID, raw, expiry); err != nil {
			t.Fatal(err)
		}
	}
	type result struct {
		pair Tokens
		err  error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			next, err := service.Refresh(ctx, raw)
			results <- result{next, err}
		}()
	}
	var winner Tokens
	successes := 0
	for range 2 {
		got := <-results
		if got.err == nil {
			winner = got.pair
			successes++
		} else if !errors.Is(got.err, errInvalidCredentials) {
			t.Fatal(got.err)
		}
	}
	if successes != 1 {
		t.Fatalf("expected one successful rotation, got %d", successes)
	}
	if err := service.Logout(ctx, winner.Refresh); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Refresh(ctx, winner.Refresh); !errors.Is(err, errInvalidCredentials) {
		t.Fatalf("logged-out refresh accepted or unexpected error: %v", err)
	}
}

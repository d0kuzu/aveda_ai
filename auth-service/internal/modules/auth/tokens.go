package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"auth-service/internal/modules/database"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type refreshClaims struct {
	jwt.RegisteredClaims
	Type string `json:"type"`
}

func tokenDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Service) issue(ctx context.Context, q *database.Queries, user database.User) (Tokens, error) {
	now := time.Now()
	access, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"type": "access",
		"exp":  now.Add(s.cfg.AccessTokenTTL).Unix(),
	}).SignedString([]byte(s.cfg.AccessSecret))
	if err != nil {
		return Tokens{}, err
	}
	expires := jwt.NewNumericDate(now.Add(s.cfg.RefreshTokenTTL))
	refresh, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: user.ID, ID: uuid.NewString(), ExpiresAt: expires},
		Type:             "refresh",
	}).SignedString([]byte(s.cfg.RefreshSecret))
	if err != nil {
		return Tokens{}, err
	}
	if err := q.SaveRefresh(ctx, user.ID, tokenDigest(refresh), expires.Time); err != nil {
		return Tokens{}, err
	}
	return Tokens{Access: access, Refresh: refresh}, nil
}

func (s *Service) refreshSubject(raw string) (string, error) {
	var claims refreshClaims
	_, err := jwt.ParseWithClaims(raw, &claims, func(_ *jwt.Token) (any, error) {
		return []byte(s.cfg.RefreshSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || claims.Type != "refresh" || claims.Subject == "" {
		return "", errInvalidCredentials
	}
	return claims.Subject, nil
}

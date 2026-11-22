package auth

import (
	"context"
	"errors"

	"auth-service/internal/config"
	"auth-service/internal/modules/database"

	"golang.org/x/crypto/bcrypt"
)

var errInvalidCredentials = errors.New("invalid credentials")

type Service struct {
	db  *database.Store
	cfg *config.Config
}

func New(db *database.Store, cfg *config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

type Tokens struct {
	Access  string `json:"access_token"`
	Refresh string `json:"refresh_token"`
}

func (s *Service) Register(ctx context.Context, email, password string) (Tokens, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Tokens{}, err
	}
	var pair Tokens
	err = s.db.InTransaction(ctx, func(q *database.Queries) error {
		user, err := q.CreateUser(ctx, email, string(hash))
		if err != nil {
			return err
		}
		pair, err = s.issue(ctx, q, user)
		return err
	})
	if err != nil {
		return Tokens{}, err
	}
	return pair, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (Tokens, error) {
	user, err := s.db.UserByEmail(ctx, email)
	if err != nil {
		return Tokens{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return Tokens{}, errInvalidCredentials
	}
	return s.issue(ctx, &s.db.Queries, user)
}

func (s *Service) Refresh(ctx context.Context, token string) (Tokens, error) {
	id, err := s.refreshSubject(token)
	if err != nil {
		return Tokens{}, err
	}
	var pair Tokens
	err = s.db.InTransaction(ctx, func(q *database.Queries) error {
		found, err := q.ConsumeRefresh(ctx, id, tokenDigest(token), token)
		if err != nil {
			return err
		}
		if !found {
			return errInvalidCredentials
		}
		user, err := q.UserByID(ctx, id)
		if err != nil {
			return err
		}
		pair, err = s.issue(ctx, q, user)
		return err
	})
	if err != nil {
		return Tokens{}, err
	}
	return pair, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	id, err := s.refreshSubject(token)
	if err != nil {
		return nil
	}
	return s.db.RevokeRefresh(ctx, id, tokenDigest(token), token)
}

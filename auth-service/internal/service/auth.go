package service

import (
	"auth-service/grpc/db"
	"auth-service/internal/config"
	"auth-service/internal/crypto"
	"auth-service/internal/jwt"
	"errors"

	"github.com/google/uuid"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type AuthService struct {
	db  *db.Client
	cfg *config.Config
}

func NewAuthService(
	db *db.Client,
	cfg *config.Config,
) *AuthService {
	return &AuthService{db, cfg}
}

func (s *AuthService) Login(email, password string) (string, string, error) {
	user, err := s.db.GetUserByEmail(email)
	if err != nil {
		return "", "", err
	}

	if !crypto.Compare(user.PasswordHash, password) {
		return "", "", ErrInvalidCredentials
	}

	access, _ := jwt.GenerateAccessToken(
		user.Id,
		user.Role,
		s.cfg.AccessTokenTTL,
		s.cfg.AccessSecret,
	)

	refreshToken, expiresAt, _ := jwt.GenerateRefreshToken(
		user.Id,
		s.cfg.RefreshTokenTTL,
		s.cfg.RefreshSecret,
	)

	// Сохраняем refresh токен через gRPC
	err = s.db.SaveRefreshToken(refreshToken, user.Id, expiresAt)
	if err != nil {
		return "", "", err
	}

	return access, refreshToken, nil
}

func (s *AuthService) Refresh(token string) (string, string, error) {
	userID, err := s.db.GetRefreshToken(token)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	// Получаем данные пользователя для генерации нового access токена
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return "", "", err
	}

	access, _ := jwt.GenerateAccessToken(
		userID,
		user.Role,
		s.cfg.AccessTokenTTL,
		s.cfg.AccessSecret,
	)

	refreshToken, expiresAt, _ := jwt.GenerateRefreshToken(
		userID,
		s.cfg.RefreshTokenTTL,
		s.cfg.RefreshSecret,
	)

	// Удаляем старый токен
	s.db.DeleteRefreshToken(token)
	// Сохраняем новый токен
	s.db.SaveRefreshToken(refreshToken, userID, expiresAt)

	return access, refreshToken, nil
}

func (s *AuthService) Logout(token string) error {
	return s.db.DeleteRefreshToken(token)
}

func (s *AuthService) Register(email, password string) (string, string, error) {
	hash, err := crypto.Hash(password)
	if err != nil {
		return "", "", err
	}

	userID := uuid.NewString()
	role := "user"

	// Создаем пользователя через gRPC
	_, err = s.db.CreateUser(email, hash, role)
	if err != nil {
		return "", "", err
	}

	access, err := jwt.GenerateAccessToken(
		userID,
		role,
		s.cfg.AccessTokenTTL,
		s.cfg.AccessSecret,
	)
	if err != nil {
		return "", "", err
	}

	refresh, expiresAt, err := jwt.GenerateRefreshToken(
		userID,
		s.cfg.RefreshTokenTTL,
		s.cfg.RefreshSecret,
	)
	if err != nil {
		return "", "", err
	}

	// Сохраняем refresh токен
	err = s.db.SaveRefreshToken(refresh, userID, expiresAt)
	if err != nil {
		return "", "", err
	}

	return access, refresh, nil
}

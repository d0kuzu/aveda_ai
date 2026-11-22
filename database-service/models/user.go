package models

import (
	"time"
)

// User - соответствует таблице users в БД
type User struct {
	ID           string    `json:"id" db:"id"`
	Email        *string   `json:"email" db:"email"`
	PasswordHash *string   `json:"password_hash" db:"password_hash"`
	Role         *string   `json:"role" db:"role"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// TableName returns the table name for User model
func (User) TableName() string {
	return "users"
}

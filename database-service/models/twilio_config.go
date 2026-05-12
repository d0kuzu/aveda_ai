package models

import "time"

// TwilioConfig - соответствует таблице twilio_configs в БД
type TwilioConfig struct {
	AssistantID  string    `json:"assistant_id" gorm:"primaryKey;type:uuid"`
	UserID       string    `json:"user_id" gorm:"type:uuid;not null"`
	TwilioNumber string    `json:"twilio_number" gorm:"not null"`
	AccountSID   string    `json:"account_sid" gorm:"not null"`
	AuthToken    string    `json:"auth_token" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// TableName returns the table name for TwilioConfig model
func (TwilioConfig) TableName() string {
	return "twilio_configs"
}

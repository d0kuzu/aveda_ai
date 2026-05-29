package models

import "time"

// Chat - соответствует таблице chats в БД (из AI сервиса)
type Chat struct {
	ID           string    `json:"id" db:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AssistantID  string    `json:"assistant_id" db:"assistant_id" gorm:"type:uuid;not null;references:assistants(id)"`
	CustomerID   *string   `json:"customer_id" db:"customer_id"`
	MessageCount int32     `json:"message_count" db:"message_count" gorm:"default:0"`
	IsEnd        bool      `json:"is_end" db:"is_end" gorm:"default:false"`
	FollowupStage int      `json:"followup_stage" db:"followup_stage" gorm:"default:1"`
	StartedAt    time.Time `json:"started_at" db:"started_at" gorm:"default:now()"`
	CreatedAt    time.Time `json:"created_at" db:"created_at" gorm:"default:now()"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at" gorm:"default:now()"`
}

// TableName returns the table name for Chat model
func (Chat) TableName() string {
	return "chats"
}

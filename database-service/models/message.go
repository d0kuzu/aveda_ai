package models

import "time"

// Message - соответствует таблице messages в БД (из AI сервиса)
type Message struct {
	ID        uint      `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	ChatID    string    `json:"chat_id" db:"chat_id" gorm:"type:uuid;not null;index:idx_chat_time;references:chats(id)"`
	Role      string    `json:"role" db:"role" gorm:"type:varchar(20);not null"`
	Content   string    `json:"content" db:"content" gorm:"type:text;not null"`
	Time      time.Time `json:"time" db:"time" gorm:"default:now();index:idx_chat_time"`
	CreatedAt time.Time `json:"created_at" db:"created_at" gorm:"default:now()"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" gorm:"default:now()"`
}

// TableName returns the table name for Message model
func (Message) TableName() string {
	return "messages"
}

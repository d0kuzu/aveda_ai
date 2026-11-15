package models

import "time"

// TutorMessage - соответствует таблице tutor_messages в БД (сообщения в режиме tutor mode)
type TutorMessage struct {
	ID        uint      `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	ChatID    string    `json:"chat_id" db:"chat_id" gorm:"type:uuid;not null;index:idx_tutor_chat_time;references:chats(id)"`
	Role      string    `json:"role" db:"role" gorm:"type:varchar(20);not null"`
	Content   string    `json:"content" db:"content" gorm:"type:text;not null"`
	Time      time.Time `json:"time" db:"time" gorm:"default:now();index:idx_tutor_chat_time"`
	CreatedAt time.Time `json:"created_at" db:"created_at" gorm:"default:now()"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" gorm:"default:now()"`
}

// TableName returns the table name for TutorMessage model
func (TutorMessage) TableName() string {
	return "tutor_messages"
}

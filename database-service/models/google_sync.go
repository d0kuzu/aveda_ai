package models

import "time"

// GoogleSync — хранит sync_token для инкрементальной синхронизации с Google Calendar
type GoogleSync struct {
	CalendarID string    `json:"calendar_id" gorm:"primaryKey;type:varchar(255)"`
	SyncToken  string    `json:"sync_token" gorm:"type:text"`
	ChannelID  string    `json:"channel_id" gorm:"type:varchar(255)"`
	ResourceID string    `json:"resource_id" gorm:"type:varchar(255)"`
	ExpiresAt  time.Time `json:"expires_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName returns the table name for GoogleSync model
func (GoogleSync) TableName() string {
	return "google_syncs"
}

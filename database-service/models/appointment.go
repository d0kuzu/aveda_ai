package models

import "time"

// Appointment — хранит записи (appointments) из Google Calendar
type Appointment struct {
	ID            string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	GoogleEventID string    `json:"google_event_id" gorm:"uniqueIndex;type:varchar(255);not null"`
	Title         string    `json:"title" gorm:"type:varchar(500)"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Status        string    `json:"status" gorm:"type:varchar(50)"`
	Description   string    `json:"description" gorm:"type:text"`
	CalendarID    string    `json:"calendar_id" gorm:"type:varchar(255)"`
	CampusLogin   bool      `json:"campus_login" gorm:"default:false"`
	CreatedAt     time.Time `json:"created_at"`
}

// TableName returns the table name for Appointment model
func (Appointment) TableName() string {
	return "appointments"
}

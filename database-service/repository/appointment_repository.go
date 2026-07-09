package repository

import (
	"diaxel_zerde/database-service/models"
	"gorm.io/gorm"
)

type AppointmentRepository interface {
	Create(appointment *models.Appointment) error
	GetByGoogleEventID(googleEventID string) (*models.Appointment, error)
}

type appointmentRepository struct {
	db *gorm.DB
}

func NewAppointmentRepository(db *gorm.DB) AppointmentRepository {
	return &appointmentRepository{db: db}
}

func (r *appointmentRepository) Create(appointment *models.Appointment) error {
	return r.db.Create(appointment).Error
}

func (r *appointmentRepository) GetByGoogleEventID(googleEventID string) (*models.Appointment, error) {
	var appointment models.Appointment
	if err := r.db.Where("google_event_id = ?", googleEventID).First(&appointment).Error; err != nil {
		return nil, err
	}
	return &appointment, nil
}

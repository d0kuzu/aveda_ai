package repository

import (
	"context"
	"time"

	"diaxel_zerde/database-service/models"
	"gorm.io/gorm"
)

type AppointmentRepository interface {
	Create(appointment *models.Appointment) error
	GetByGoogleEventID(googleEventID string) (*models.Appointment, error)
	CountByDateRange(ctx context.Context, startTime, endTime time.Time) (int32, error)
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

func (r *appointmentRepository) CountByDateRange(ctx context.Context, startTime, endTime time.Time) (int32, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Appointment{}).
		Where("created_at >= ? AND created_at < ?", startTime, endTime).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

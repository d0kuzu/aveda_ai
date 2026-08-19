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
	CountBySlot(ctx context.Context, calendarID string, startTime time.Time) (int32, error)
	GetUnsyncedCampusloginAppointments(ctx context.Context) ([]*models.Appointment, error)
	UpdateAppointmentCampusloginStatus(ctx context.Context, id string, status bool) (*models.Appointment, error)
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

func (r *appointmentRepository) CountBySlot(ctx context.Context, calendarID string, startTime time.Time) (int32, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Appointment{}).
		Where("calendar_id = ? AND start_time = ?", calendarID, startTime).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

func (r *appointmentRepository) GetUnsyncedCampusloginAppointments(ctx context.Context) ([]*models.Appointment, error) {
	var appointments []*models.Appointment
	err := r.db.WithContext(ctx).Where("campus_login = ?", false).Find(&appointments).Error
	if err != nil {
		return nil, err
	}
	return appointments, nil
}

func (r *appointmentRepository) UpdateAppointmentCampusloginStatus(ctx context.Context, id string, status bool) (*models.Appointment, error) {
	var appointment models.Appointment
	err := r.db.WithContext(ctx).First(&appointment, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	appointment.CampusLogin = status
	err = r.db.WithContext(ctx).Save(&appointment).Error
	if err != nil {
		return nil, err
	}

	return &appointment, nil
}

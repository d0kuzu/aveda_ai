package repository

import (
	"diaxel_zerde/database-service/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GoogleSyncRepository interface {
	Upsert(sync *models.GoogleSync) error
	GetByCalendarID(calendarID string) (*models.GoogleSync, error)
}

type googleSyncRepository struct {
	db *gorm.DB
}

func NewGoogleSyncRepository(db *gorm.DB) GoogleSyncRepository {
	return &googleSyncRepository{db: db}
}

func (r *googleSyncRepository) Upsert(sync *models.GoogleSync) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "calendar_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"sync_token", "channel_id", "resource_id", "updated_at"}),
	}).Create(sync).Error
}

func (r *googleSyncRepository) GetByCalendarID(calendarID string) (*models.GoogleSync, error) {
	var sync models.GoogleSync
	if err := r.db.Where("calendar_id = ?", calendarID).First(&sync).Error; err != nil {
		return nil, err
	}
	return &sync, nil
}

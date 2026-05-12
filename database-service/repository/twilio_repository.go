package repository

import (
	"diaxel_zerde/database-service/models"
	"gorm.io/gorm"
)

type TwilioRepository interface {
	SaveConfig(config *models.TwilioConfig) error
	GetConfigByAssistantID(assistantID string) (*models.TwilioConfig, error)
	DeleteConfig(assistantID string) error
}

type twilioRepository struct {
	db *gorm.DB
}

func NewTwilioRepository(db *gorm.DB) TwilioRepository {
	return &twilioRepository{db: db}
}

func (r *twilioRepository) SaveConfig(config *models.TwilioConfig) error {
	return r.db.Save(config).Error
}

func (r *twilioRepository) GetConfigByAssistantID(assistantID string) (*models.TwilioConfig, error) {
	var config models.TwilioConfig
	if err := r.db.Where("assistant_id = ?", assistantID).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *twilioRepository) DeleteConfig(assistantID string) error {
	return r.db.Where("assistant_id = ?", assistantID).Delete(&models.TwilioConfig{}).Error
}

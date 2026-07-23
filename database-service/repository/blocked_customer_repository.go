package repository

import (
	"context"
	"fmt"

	"diaxel_zerde/database-service/models"

	"gorm.io/gorm"
)

type BlockedCustomerRepository interface {
	IsBlocked(ctx context.Context, userID string) (bool, error)
	Block(ctx context.Context, userID string) error
	GetAll(ctx context.Context) ([]string, error)
}

type blockedCustomerRepository struct {
	db *gorm.DB
}

func NewBlockedCustomerRepository(db *gorm.DB) BlockedCustomerRepository {
	return &blockedCustomerRepository{db: db}
}

func (r *blockedCustomerRepository) IsBlocked(ctx context.Context, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.BlockedCustomer{}).Where("user_id = ?", userID).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check blocked status: %w", err)
	}
	return count > 0, nil
}

func (r *blockedCustomerRepository) Block(ctx context.Context, userID string) error {
	blockedCustomer := models.BlockedCustomer{UserID: userID}
	err := r.db.WithContext(ctx).Create(&blockedCustomer).Error
	if err != nil {
		return fmt.Errorf("failed to block customer: %w", err)
	}
	return nil
}

func (r *blockedCustomerRepository) GetAll(ctx context.Context) ([]string, error) {
	var blockedCustomers []models.BlockedCustomer
	err := r.db.WithContext(ctx).Find(&blockedCustomers).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all blocked customers: %w", err)
	}

	var userIDs []string
	for _, bc := range blockedCustomers {
		userIDs = append(userIDs, bc.UserID)
	}
	return userIDs, nil
}

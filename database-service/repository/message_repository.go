package repository

import (
	"context"
	"fmt"
	"time"

	"diaxel_zerde/database-service/models"

	"gorm.io/gorm"
)

type MessageRepository interface {
	SaveMessage(ctx context.Context, chatID, role, content, platform string, msgTime string) (*models.Message, error)
	GetMessageByID(ctx context.Context, id uint) (*models.Message, error)
	GetMessagesByChatID(ctx context.Context, chatID string, limit, offset int32) ([]*models.Message, error)
	GetMessagePagesCount(ctx context.Context, chatID string, messagesPerPage int32) (int32, error)
	GetAllChatMessages(ctx context.Context, chatID string) ([]*models.Message, error)
	UpdateMessage(ctx context.Context, id uint, role, content string) (*models.Message, error)
	DeleteMessage(ctx context.Context, id uint) error
	DeleteMessagesByChatID(ctx context.Context, chatID string) error
	DeleteAllMessages(ctx context.Context) error
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) SaveMessage(ctx context.Context, chatID, role, content, platform string, msgTime string) (*models.Message, error) {
	parsedTime := time.Now().UTC()
	if msgTime != "" {
		if t, err := time.Parse(time.RFC3339, msgTime); err == nil {
			parsedTime = t
		} else {
			// For debug, but no need to log every fail
		}
	}

	message := models.Message{
		ChatID:    chatID,
		Role:      role,
		Content:   content,
		Time:      parsedTime,
		CreatedAt: parsedTime,
		UpdatedAt: parsedTime,
	}

	if err := r.db.WithContext(ctx).Create(&message).Error; err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	return &message, nil
}

func (r *messageRepository) GetMessagesByChatID(ctx context.Context, chatID string, limit, offset int32) ([]*models.Message, error) {
	var messages []*models.Message
	if err := r.db.WithContext(ctx).Where("chat_id = ?", chatID).Order("time ASC").Limit(int(limit)).Offset(int(offset)).Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	return messages, nil
}

func (r *messageRepository) GetMessagePagesCount(ctx context.Context, chatID string, messagesPerPage int32) (int32, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Message{}).Where("chat_id = ?", chatID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	if messagesPerPage <= 0 {
		messagesPerPage = 10
	}

	pagesCount := int32((count + int64(messagesPerPage) - 1) / int64(messagesPerPage))
	return pagesCount, nil
}

func (r *messageRepository) GetAllChatMessages(ctx context.Context, chatID string) ([]*models.Message, error) {
	var messages []*models.Message
	if err := r.db.WithContext(ctx).Where("chat_id = ?", chatID).Order("time ASC").Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to get all messages: %w", err)
	}

	return messages, nil
}

func (r *messageRepository) GetMessageByID(ctx context.Context, id uint) (*models.Message, error) {
	var message models.Message
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&message).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("message not found")
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return &message, nil
}

func (r *messageRepository) UpdateMessage(ctx context.Context, id uint, role, content string) (*models.Message, error) {
	var message models.Message
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&message).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("message not found")
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	message.Role = role
	message.Content = content
	message.UpdatedAt = time.Now()

	if err := r.db.WithContext(ctx).Save(&message).Error; err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	return &message, nil
}

func (r *messageRepository) DeleteMessage(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Message{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete message: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("message not found")
	}

	return nil
}

func (r *messageRepository) DeleteAllMessages(ctx context.Context) error {
	result := r.db.WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Message{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete all messages: %w", result.Error)
	}
	return nil
}

func (r *messageRepository) DeleteMessagesByChatID(ctx context.Context, chatID string) error {
	result := r.db.WithContext(ctx).Where("chat_id = ?", chatID).Delete(&models.Message{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete messages by chat id: %w", result.Error)
	}
	return nil
}

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"diaxel_zerde/database-service/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatRepository interface {
	CreateChat(ctx context.Context, assistantID, customerID, platform string) (*models.Chat, error)
	GetChatByID(ctx context.Context, id string) (*models.Chat, error)
	GetChatsByCustomerID(ctx context.Context, customerID string) ([]*models.Chat, error)
	GetChatsByUserID(ctx context.Context, assistantIDs []string, limit, offset int32) ([]*models.Chat, error)
	UpdateChat(ctx context.Context, id, assistantID, customerID string) (*models.Chat, error)
	UpdateChatIsEnd(ctx context.Context, id string, isEnd bool) (*models.Chat, error)
	UpdateChatIsBooked(ctx context.Context, id string, isBooked bool) (*models.Chat, error)
	UpdateChatIsReviewed(ctx context.Context, id string, isReviewed bool) (*models.Chat, error)
	GetUnreviewedActiveChats(ctx context.Context) ([]*models.Chat, error)
	DeleteChat(ctx context.Context, id string) error
	DeleteAllChats(ctx context.Context) error
	GetChatPagesCount(ctx context.Context, assistantID string, chatsPerPage int32) (int32, error)
	GetChatPage(ctx context.Context, assistantID string, page, chatsPerPage int32) ([]*models.Chat, error)
	GetChatPagesCountByUserID(ctx context.Context, assistantIDs []string, chatsPerPage int32) (int32, error)
	GetChatPageByUserID(ctx context.Context, assistantIDs []string, page, chatsPerPage int32) ([]*models.Chat, error)
	SearchChatsByCustomer(ctx context.Context, assistantIDs []string, search string) ([]*models.Chat, int32, error)
	GetLatestChatByCustomer(ctx context.Context, assistantID, customerID string) (*models.Chat, error)
	UpdateMessageCount(ctx context.Context, chatID string) error
	GetChatsForFollowup(ctx context.Context) ([]*models.Chat, error)
	UpdateChatFollowupStage(ctx context.Context, id string, stage int) (*models.Chat, error)
	GetPeriodMetrics(ctx context.Context, assistantID string, startTime, endTime time.Time) (int32, int32, int32, error)
	GetWeeklyChatsStarted(ctx context.Context, assistantID string, startTime time.Time, timezone string) ([]DailyCount, error)
}

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) CreateChat(ctx context.Context, assistantID, customerID, platform string) (*models.Chat, error) {
	chat := models.Chat{
		ID:          uuid.New().String(),
		AssistantID: assistantID,
		CustomerID:  &customerID,
		StartedAt:   time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(&chat).Error; err != nil {
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}

	return &chat, nil
}

func (r *chatRepository) GetChatByID(ctx context.Context, id string) (*models.Chat, error) {
	var chat models.Chat
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&chat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("chat not found")
		}
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	return &chat, nil
}

func (r *chatRepository) GetChatsByCustomerID(ctx context.Context, customerID string) ([]*models.Chat, error) {
	var chats []*models.Chat
	if err := r.db.WithContext(ctx).Where("customer_id = ?", customerID).Order("started_at DESC").Find(&chats).Error; err != nil {
		return nil, fmt.Errorf("failed to get chats: %w", err)
	}

	return chats, nil
}

func (r *chatRepository) GetChatPagesCount(ctx context.Context, assistantID string, chatsPerPage int32) (int32, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Chat{}).Where("assistant_id = ?", assistantID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count chats: %w", err)
	}

	if chatsPerPage <= 0 {
		chatsPerPage = 10 // default value
	}

	pagesCount := int32((count + int64(chatsPerPage) - 1) / int64(chatsPerPage))
	return pagesCount, nil
}

func (r *chatRepository) GetChatPage(ctx context.Context, assistantID string, page, chatsPerPage int32) ([]*models.Chat, error) {
	if page <= 0 {
		page = 1
	}
	if chatsPerPage <= 0 {
		chatsPerPage = 10
	}

	offset := (page - 1) * chatsPerPage

	var chats []*models.Chat
	if err := r.db.WithContext(ctx).
		Where("assistant_id = ?", assistantID).
		Order("created_at DESC").
		Limit(int(chatsPerPage)).
		Offset(int(offset)).
		Find(&chats).Error; err != nil {
		return nil, fmt.Errorf("failed to get chat page: %w", err)
	}

	return chats, nil
}

func (r *chatRepository) GetChatPagesCountByUserID(ctx context.Context, assistantIDs []string, chatsPerPage int32) (int32, error) {
	if len(assistantIDs) == 0 {
		return 0, nil
	}
	log.Printf("GetChatPagesCountByUserID: assistantIDs=%v, chatsPerPage=%d", assistantIDs, chatsPerPage)
	var count int64
	query := r.db.WithContext(ctx).Debug().Model(&models.Chat{}).Where("assistant_id IN ?", assistantIDs)

	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count chats by user id: %w", err)
	}

	if chatsPerPage <= 0 {
		chatsPerPage = 10
	}

	pagesCount := int32((count + int64(chatsPerPage) - 1) / int64(chatsPerPage))
	return pagesCount, nil
}

func (r *chatRepository) GetChatPageByUserID(ctx context.Context, assistantIDs []string, page, chatsPerPage int32) ([]*models.Chat, error) {
	if len(assistantIDs) == 0 {
		return []*models.Chat{}, nil
	}
	if page <= 0 {
		page = 1
	}
	if chatsPerPage <= 0 {
		chatsPerPage = 10
	}

	offset := (page - 1) * chatsPerPage

	log.Printf("GetChatPageByUserID: assistantIDs=%v, page=%d, chatsPerPage=%d, offset=%d", assistantIDs, page, chatsPerPage, offset)

	var chats []*models.Chat
	query := r.db.WithContext(ctx).Debug().Where("assistant_id IN ?", assistantIDs)

	if err := query.Order("created_at DESC").Limit(int(chatsPerPage)).Offset(int(offset)).Find(&chats).Error; err != nil {
		return nil, fmt.Errorf("failed to get chat page by user id: %w", err)
	}

	log.Printf("GetChatPageByUserID: found %d chats", len(chats))

	return chats, nil
}

func (r *chatRepository) SearchChatsByCustomer(ctx context.Context, assistantIDs []string, search string) ([]*models.Chat, int32, error) {
	if len(assistantIDs) == 0 {
		return []*models.Chat{}, 0, nil
	}
	var chats []*models.Chat
	var count int64

	query := r.db.WithContext(ctx).Model(&models.Chat{}).Where("assistant_id IN ?", assistantIDs)

	// If search term is provided, search by customer_id
	if search != "" {
		query = query.Where("customer_id LIKE ?", "%"+search+"%")
	}

	// Get total count
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count search results: %w", err)
	}

	// Get the actual chats
	if err := query.Order("created_at DESC").Find(&chats).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to search chats: %w", err)
	}

	return chats, int32(count), nil
}

func (r *chatRepository) GetLatestChatByCustomer(ctx context.Context, assistantID, customerID string) (*models.Chat, error) {
	var chat models.Chat
	result := r.db.WithContext(ctx).
		Where("assistant_id = ? AND customer_id = ?", assistantID, customerID).
		Order("started_at DESC").
		First(&chat)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // Return nil if no chat found
		}
		return nil, fmt.Errorf("failed to get latest chat: %w", result.Error)
	}

	return &chat, nil
}

func (r *chatRepository) UpdateMessageCount(ctx context.Context, chatID string) error {
	// Count messages for this chat
	var messageCount int64
	if err := r.db.WithContext(ctx).Model(&models.Message{}).Where("chat_id = ?", chatID).Count(&messageCount).Error; err != nil {
		return fmt.Errorf("failed to count messages: %w", err)
	}

	// Update the message count in the chat
	if err := r.db.WithContext(ctx).Model(&models.Chat{}).Where("id = ?", chatID).Update("message_count", messageCount).Error; err != nil {
		return fmt.Errorf("failed to update message count: %w", err)
	}

	return nil
}

func (r *chatRepository) GetChatsByUserID(ctx context.Context, assistantIDs []string, limit, offset int32) ([]*models.Chat, error) {
	if len(assistantIDs) == 0 {
		return []*models.Chat{}, nil
	}
	var chats []*models.Chat
	query := r.db.WithContext(ctx).Where("assistant_id IN ?", assistantIDs)
	if limit > 0 {
		query = query.Limit(int(limit))
	}
	if offset > 0 {
		query = query.Offset(int(offset))
	}
	if err := query.Order("created_at DESC").Find(&chats).Error; err != nil {
		return nil, fmt.Errorf("failed to get chats by user: %w", err)
	}

	return chats, nil
}

func (r *chatRepository) UpdateChat(ctx context.Context, id, assistantID, customerID string) (*models.Chat, error) {
	var chat models.Chat
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&chat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("chat not found")
		}
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	if assistantID != "" {
		chat.AssistantID = assistantID
	}
	if customerID != "" {
		chat.CustomerID = &customerID
	}
	chat.UpdatedAt = time.Now()

	if err := r.db.WithContext(ctx).Save(&chat).Error; err != nil {
		return nil, fmt.Errorf("failed to update chat: %w", err)
	}

	return &chat, nil
}

func (r *chatRepository) DeleteChat(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Chat{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete chat: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("chat not found")
	}

	return nil
}

func (r *chatRepository) DeleteAllChats(ctx context.Context) error {
	result := r.db.WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Chat{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete all chats: %w", result.Error)
	}
	return nil
}

func (r *chatRepository) UpdateChatIsEnd(ctx context.Context, id string, isEnd bool) (*models.Chat, error) {
	var chat models.Chat
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&chat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("chat not found")
		}
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	chat.IsEnd = isEnd
	chat.UpdatedAt = time.Now()

	if err := r.db.WithContext(ctx).Save(&chat).Error; err != nil {
		return nil, fmt.Errorf("failed to update chat is_end: %w", err)
	}

	return &chat, nil
}

func (r *chatRepository) UpdateChatIsBooked(ctx context.Context, id string, isBooked bool) (*models.Chat, error) {
	var chat models.Chat
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&chat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("chat not found")
		}
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	chat.IsBooked = isBooked
	chat.UpdatedAt = time.Now()

	if err := r.db.WithContext(ctx).Save(&chat).Error; err != nil {
		return nil, fmt.Errorf("failed to update chat is_booked: %w", err)
	}

	return &chat, nil
}

func (r *chatRepository) UpdateChatIsReviewed(ctx context.Context, id string, isReviewed bool) (*models.Chat, error) {
	var chat models.Chat
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&chat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("chat not found")
		}
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	chat.IsReviewed = isReviewed
	chat.UpdatedAt = time.Now()

	if err := r.db.WithContext(ctx).Save(&chat).Error; err != nil {
		return nil, fmt.Errorf("failed to update chat is_reviewed: %w", err)
	}

	return &chat, nil
}

func (r *chatRepository) GetUnreviewedActiveChats(ctx context.Context) ([]*models.Chat, error) {
	var chats []*models.Chat

	if err := r.db.WithContext(ctx).
		Where("is_end = ? AND is_reviewed = ?", false, false).
		Order("updated_at ASC").
		Find(&chats).Error; err != nil {
		return nil, fmt.Errorf("failed to get unreviewed active chats: %w", err)
	}

	return chats, nil
}

func (r *chatRepository) GetChatsForFollowup(ctx context.Context) ([]*models.Chat, error) {
	var chats []*models.Chat

	if err := r.db.WithContext(ctx).
		Where("is_end = ?", false).
		Order("updated_at ASC").
		Find(&chats).Error; err != nil {
		return nil, fmt.Errorf("failed to get chats for followup: %w", err)
	}

	return chats, nil
}

func (r *chatRepository) UpdateChatFollowupStage(ctx context.Context, id string, stage int) (*models.Chat, error) {
	var chat models.Chat
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&chat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("chat not found")
		}
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	chat.FollowupStage = stage
	if err := r.db.WithContext(ctx).Save(&chat).Error; err != nil {
		return nil, fmt.Errorf("failed to update chat followup stage: %w", err)
	}

	return &chat, nil
}

func (r *chatRepository) GetPeriodMetrics(ctx context.Context, assistantID string, startTime, endTime time.Time) (int32, int32, int32, error) {
	var startedCount int64
	var completedCount int64
	var bookedCount int64

	query := r.db.WithContext(ctx).Model(&models.Chat{}).Where("started_at >= ? AND started_at < ?", startTime, endTime)
	if assistantID != "" {
		query = query.Where("assistant_id = ?", assistantID)
	}

	if err := query.Count(&startedCount).Error; err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get started chats count: %w", err)
	}

	// Use a fresh query for completed Count since the previous Where might mutate
	completedQuery := r.db.WithContext(ctx).Model(&models.Chat{}).Where("started_at >= ? AND started_at < ?", startTime, endTime)
	if assistantID != "" {
		completedQuery = completedQuery.Where("assistant_id = ?", assistantID)
	}
	if err := completedQuery.Where("is_end = ?", true).Count(&completedCount).Error; err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get completed chats count: %w", err)
	}

	bookedQuery := r.db.WithContext(ctx).Model(&models.Chat{}).Where("started_at >= ? AND started_at < ?", startTime, endTime)
	if assistantID != "" {
		bookedQuery = bookedQuery.Where("assistant_id = ?", assistantID)
	}
	if err := bookedQuery.Where("is_booked = ?", true).Count(&bookedCount).Error; err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get booked chats count: %w", err)
	}

	return int32(startedCount), int32(completedCount), int32(bookedCount), nil
}

type DailyCount struct {
	Date  string
	Count int32
}

func (r *chatRepository) GetWeeklyChatsStarted(ctx context.Context, assistantID string, startTime time.Time, timezone string) ([]DailyCount, error) {
	endTime := startTime.AddDate(0, 0, 7)

	sqlDB, err := r.db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	var rows *sql.Rows
	if assistantID != "" {
		rows, err = sqlDB.QueryContext(ctx,
			`SELECT TO_CHAR(started_at AT TIME ZONE $1, 'YYYY-MM-DD') AS day, COUNT(*) AS count
			 FROM chats
			 WHERE started_at >= $2 AND started_at < $3
			   AND assistant_id::text = $4
			 GROUP BY TO_CHAR(started_at AT TIME ZONE $1, 'YYYY-MM-DD')
			 ORDER BY day ASC`,
			timezone, startTime, endTime, assistantID,
		)
	} else {
		rows, err = sqlDB.QueryContext(ctx,
			`SELECT TO_CHAR(started_at AT TIME ZONE $1, 'YYYY-MM-DD') AS day, COUNT(*) AS count
			 FROM chats
			 WHERE started_at >= $2 AND started_at < $3
			 GROUP BY TO_CHAR(started_at AT TIME ZONE $1, 'YYYY-MM-DD')
			 ORDER BY day ASC`,
			timezone, startTime, endTime,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query weekly chats: %w", err)
	}
	defer rows.Close()

	countMap := make(map[string]int32)
	for rows.Next() {
		var day string
		var count int32
		if err := rows.Scan(&day, &count); err != nil {
			return nil, fmt.Errorf("failed to scan weekly chats row: %w", err)
		}
		countMap[day] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("weekly chats rows error: %w", err)
	}

	log.Printf("[DEBUG GetWeeklyChatsStarted] timezone: %s, countMap: %+v", timezone, countMap)

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	days := make([]DailyCount, 7)
	for i := 0; i < 7; i++ {
		day := startTime.In(loc).AddDate(0, 0, i)
		dayStr := day.Format("2006-01-02")
		days[i] = DailyCount{
			Date:  dayStr,
			Count: countMap[dayStr],
		}
	}

	return days, nil
}

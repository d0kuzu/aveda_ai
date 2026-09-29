package analytics

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"diaxel/internal/app"
	"diaxel/internal/constants"

	"github.com/gin-gonic/gin"
)

type PeriodMetrics struct {
	StartedChats     int32   `json:"started_conversations"`
	CompletedChats   int32   `json:"completed_conversations"`
	BookedMeetings   int32   `json:"booked_meetings"`
	ConversionRate   float64 `json:"conversion_rate"`
	StartedChange    float64 `json:"started_change_pct"`
	CompletedChange  float64 `json:"completed_change_pct"`
	BookedChange     float64 `json:"booked_change_pct"`
	ConversionChange float64 `json:"conversion_change_pct"`
}

type DailyCount struct {
	Date  string `json:"date"`
	Count int32  `json:"count"`
}

type AnalyticsResponse struct {
	Today                      PeriodMetrics `json:"today"`
	Days7                      PeriodMetrics `json:"7_days"`
	Days30                     PeriodMetrics `json:"30_days"`
	Days60                     PeriodMetrics `json:"60_days"`
	Days90                     PeriodMetrics `json:"90_days"`
	WeeklyConversationsStarted []DailyCount  `json:"weekly_conversations_started"`
}

type AnalyticsChatJSON struct {
	ID            string `json:"id"`
	AssistantID   string `json:"assistant_id"`
	CustomerID    string `json:"customer_id"`
	Platform      string `json:"platform"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	StartedAt     string `json:"started_at"`
	MessageCount  int32  `json:"message_count"`
	IsEnd         bool   `json:"is_end"`
	FollowupStage int32  `json:"followup_stage"`
	IsReviewed    bool   `json:"is_reviewed"`
	IsBooked      bool   `json:"is_booked"`
}

type AnalyticsChatsResponse struct {
	Category    string              `json:"category"`
	Days        int                 `json:"days"`
	TotalCount  int64               `json:"total_count"`
	TotalPages  int64               `json:"total_pages"`
	CurrentPage int                 `json:"current_page"`
	Limit       int                 `json:"limit"`
	Chats       []AnalyticsChatJSON `json:"chats"`
}

func CalculatePeriodRange(now time.Time, location *time.Location, days int) (startCurrent, endCurrent time.Time) {
	y, m, d := now.Date()
	if days == 1 {
		startCurrent = time.Date(y, m, d, 0, 0, 0, 0, location)
		endCurrent = startCurrent.AddDate(0, 0, 1)
	} else {
		startCurrent = time.Date(y, m, d, 0, 0, 0, 0, location).AddDate(0, 0, -days+1)
		endCurrent = time.Date(y, m, d, 0, 0, 0, 0, location).AddDate(0, 0, 1)
	}
	return startCurrent, endCurrent
}

func GetAnalytics(application *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		assistantID := c.Query("assistant_id")
		if assistantID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "assistant_id is required"})
			return
		}

		location, err := time.LoadLocation(constants.DefaultTimezone)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid timezone"})
			return
		}

		now := time.Now().In(location)

		getMetrics := func(days int) (PeriodMetrics, error) {
			startCurrent, endCurrent := CalculatePeriodRange(now, location, days)
			startPrev := startCurrent.AddDate(0, 0, -days)
			endPrev := startCurrent

			currentResp, err := application.Db.GetPeriodMetrics(assistantID, startCurrent.Format(time.RFC3339), endCurrent.Format(time.RFC3339))
			if err != nil {
				return PeriodMetrics{}, err
			}

			prevResp, err := application.Db.GetPeriodMetrics(assistantID, startPrev.Format(time.RFC3339), endPrev.Format(time.RFC3339))
			if err != nil {
				return PeriodMetrics{}, err
			}

			currBookedChats := currentResp.BookedChats
			prevBookedChats := prevResp.BookedChats

			if assistantID == constants.AvedaSintaAssistantID && application.Calcom != nil {
				var wg sync.WaitGroup
				wg.Add(2)

				go func() {
					defer wg.Done()
					ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
					defer cancel()
					count, err := application.Calcom.GetBookingsCount(ctx, startCurrent, endCurrent)
					if err == nil {
						currBookedChats = int32(count)
					} else {
						log.Printf("Failed to get calcom bookings for current period: %v", err)
					}
				}()

				go func() {
					defer wg.Done()
					ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
					defer cancel()
					count, err := application.Calcom.GetBookingsCount(ctx, startPrev, endPrev)
					if err == nil {
						prevBookedChats = int32(count)
					} else {
						log.Printf("Failed to get calcom bookings for previous period: %v", err)
					}
				}()

				wg.Wait()
			}

			currConversion := 0.0
			if currentResp.StartedChats > 0 {
				currConversion = float64(currBookedChats) / float64(currentResp.StartedChats) * 100
			}

			prevConversion := 0.0
			if prevResp.StartedChats > 0 {
				prevConversion = float64(prevBookedChats) / float64(prevResp.StartedChats) * 100
			}

			calcChange := func(curr, prev float64) float64 {
				if prev == 0 {
					if curr > 0 {
						return 100.0
					}
					return 0.0
				}
				return ((curr - prev) / prev) * 100
			}

			return PeriodMetrics{
				StartedChats:     currentResp.StartedChats,
				CompletedChats:   currentResp.CompletedChats,
				BookedMeetings:   currBookedChats,
				ConversionRate:   currConversion,
				StartedChange:    calcChange(float64(currentResp.StartedChats), float64(prevResp.StartedChats)),
				CompletedChange:  calcChange(float64(currentResp.CompletedChats), float64(prevResp.CompletedChats)),
				BookedChange:     calcChange(float64(currBookedChats), float64(prevBookedChats)),
				ConversionChange: currConversion - prevConversion,
			}, nil
		}

		type metricsResult struct {
			metrics PeriodMetrics
			err     error
		}

		periods := []int{1, 7, 30, 60, 90}
		results := make([]metricsResult, len(periods))

		var wgPeriods sync.WaitGroup
		wgPeriods.Add(len(periods))
		for i, days := range periods {
			go func(idx, d int) {
				defer wgPeriods.Done()
				m, err := getMetrics(d)
				results[idx] = metricsResult{metrics: m, err: err}
			}(i, days)
		}
		wgPeriods.Wait()

		for i, r := range results {
			if r.err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": r.err.Error(), "period": periods[i]})
				return
			}
		}

		today := results[0].metrics
		d7 := results[1].metrics
		d30 := results[2].metrics
		d60 := results[3].metrics
		d90 := results[4].metrics

		// Weekly chart data — last 7 days starting from 7 days ago
		y, m, d := now.Date()
		weekStart := time.Date(y, m, d, 0, 0, 0, 0, location).AddDate(0, 0, -6)
		weeklyResp, err := application.Db.GetWeeklyChatsStarted(assistantID, weekStart.Format(time.RFC3339), constants.DefaultTimezone)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		weeklyDays := make([]DailyCount, len(weeklyResp.Days))
		for i, day := range weeklyResp.Days {
			weeklyDays[i] = DailyCount{Date: day.Date, Count: day.Count}
		}
		log.Printf("Weekly days: %v\n", weeklyDays)
		c.JSON(http.StatusOK, AnalyticsResponse{
			Today:                      today,
			Days7:                      d7,
			Days30:                     d30,
			Days60:                     d60,
			Days90:                     d90,
			WeeklyConversationsStarted: weeklyDays,
		})
	}
}

func GetAnalyticsChatsByCategory(application *app.App, forcedCategory string) gin.HandlerFunc {
	return func(c *gin.Context) {
		assistantID := c.Query("assistant_id")
		if assistantID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "assistant_id is required"})
			return
		}

		daysStr := c.Query("days")
		if daysStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "days parameter is required (1, 7, 30, 60, or 90)"})
			return
		}
		days, err := strconv.Atoi(daysStr)
		if err != nil || (days != 1 && days != 7 && days != 30 && days != 60 && days != 90) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid days: must be 1, 7, 30, 60 or 90"})
			return
		}

		category := forcedCategory
		if category == "" {
			category = c.Query("category")
		}
		if category != "started" && category != "completed" && category != "booked" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category: must be started, completed, or booked"})
			return
		}

		pageStr := c.DefaultQuery("page", "1")
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		limitStr := c.DefaultQuery("limit", "20")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}
		offset := (page - 1) * limit

		location, err := time.LoadLocation(constants.DefaultTimezone)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid timezone"})
			return
		}

		now := time.Now().In(location)
		startCurrent, endCurrent := CalculatePeriodRange(now, location, days)

		resp, err := application.Db.GetPeriodChats(
			assistantID,
			startCurrent.Format(time.RFC3339),
			endCurrent.Format(time.RFC3339),
			category,
			int32(limit),
			int32(offset),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get period chats: " + err.Error()})
			return
		}

		chatsJSON := make([]AnalyticsChatJSON, len(resp.Chats))
		for i, chat := range resp.Chats {
			chatsJSON[i] = AnalyticsChatJSON{
				ID:            chat.Id,
				AssistantID:   chat.AssistantId,
				CustomerID:    chat.CustomerId,
				Platform:      chat.Platform,
				CreatedAt:     chat.CreatedAt,
				UpdatedAt:     chat.UpdatedAt,
				StartedAt:     chat.StartedAt,
				MessageCount:  chat.MessageCount,
				IsEnd:         chat.IsEnd,
				FollowupStage: chat.FollowupStage,
				IsReviewed:    chat.IsReviewed,
				IsBooked:      chat.IsBooked,
			}
		}

		var totalPages int64 = 0
		if resp.TotalCount > 0 {
			totalPages = (resp.TotalCount + int64(limit) - 1) / int64(limit)
		}

		c.JSON(http.StatusOK, AnalyticsChatsResponse{
			Category:    category,
			Days:        days,
			TotalCount:  resp.TotalCount,
			TotalPages:  totalPages,
			CurrentPage: page,
			Limit:       limit,
			Chats:       chatsJSON,
		})
	}
}

func GetStartedChats(application *app.App) gin.HandlerFunc {
	return GetAnalyticsChatsByCategory(application, "started")
}

func GetCompletedChats(application *app.App) gin.HandlerFunc {
	return GetAnalyticsChatsByCategory(application, "completed")
}

func GetBookedChats(application *app.App) gin.HandlerFunc {
	return GetAnalyticsChatsByCategory(application, "booked")
}


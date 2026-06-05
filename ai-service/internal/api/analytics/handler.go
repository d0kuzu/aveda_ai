package analytics

import (
	"log"
	"net/http"
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
			var startCurrent, endCurrent time.Time

			if days == 1 {
				// today
				y, m, d := now.Date()
				startCurrent = time.Date(y, m, d, 0, 0, 0, 0, location)
				endCurrent = startCurrent.AddDate(0, 0, 1)
			} else {
				y, m, d := now.Date()
				startCurrent = time.Date(y, m, d, 0, 0, 0, 0, location).AddDate(0, 0, -days+1)
				endCurrent = time.Date(y, m, d, 0, 0, 0, 0, location).AddDate(0, 0, 1)
			}

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

			currConversion := 0.0
			if currentResp.StartedChats > 0 {
				currConversion = float64(currentResp.CompletedChats) / float64(currentResp.StartedChats) * 100
			}

			prevConversion := 0.0
			if prevResp.StartedChats > 0 {
				prevConversion = float64(prevResp.CompletedChats) / float64(prevResp.StartedChats) * 100
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
				BookedMeetings:   currentResp.CompletedChats, // Currently the same as completed chats per logic
				ConversionRate:   currConversion,
				StartedChange:    calcChange(float64(currentResp.StartedChats), float64(prevResp.StartedChats)),
				CompletedChange:  calcChange(float64(currentResp.CompletedChats), float64(prevResp.CompletedChats)),
				BookedChange:     calcChange(float64(currentResp.CompletedChats), float64(prevResp.CompletedChats)),
				ConversionChange: currConversion - prevConversion,
			}, nil
		}

		today, err := getMetrics(1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		d7, err := getMetrics(7)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		d30, err := getMetrics(30)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		d60, err := getMetrics(60)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		d90, err := getMetrics(90)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Weekly chart data — last 7 days starting from 7 days ago
		y, m, d := now.Date()
		weekStart := time.Date(y, m, d, 0, 0, 0, 0, location).AddDate(0, 0, -6)
		weeklyResp, err := application.Db.GetWeeklyChatsStarted(assistantID, weekStart.Format(time.RFC3339))
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

package google

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/campuslogin"
	"diaxel/internal/modules/googlecalendar"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/calendar/v3"
)

const defaultCalendarID = "primary"

type GoogleHandler struct {
	gc *googlecalendar.Client
	db *db.Client
	cl *campuslogin.Client
}

func NewGoogleHandler(gc *googlecalendar.Client, db *db.Client, cl *campuslogin.Client) *GoogleHandler {
	return &GoogleHandler{gc: gc, db: db, cl: cl}
}

// HandleWebhook обрабатывает push-нотификацию от Google Calendar.
// Сразу отвечает 200 OK и асинхронно подгружает новые события.
func (h *GoogleHandler) HandleWebhook(c *gin.Context) {
	resourceID := c.GetHeader("X-Goog-Resource-ID")
	channelID := c.GetHeader("X-Goog-Channel-ID")

	log.Printf("[GoogleWebhook] received notification: channelID=%s, resourceID=%s", channelID, resourceID)

	// Сразу отвечаем 200 OK — Google требует быстрый ответ
	c.Status(http.StatusOK)

	// go h.processEvents(channelID, resourceID)
}

func (h *GoogleHandler) processEvents(channelID, resourceID string) {
	log.Printf("[GoogleWebhook] processEvents: calendar processing started")
	log.Printf("[GoogleWebhook] processEvents: channelID=%s", channelID)
	log.Printf("[GoogleWebhook] processEvents: resourceID=%s", resourceID)

	calendarID := defaultCalendarID
	log.Printf("[GoogleWebhook] processEvents: calendarID=%s", calendarID)

	// Получаем текущий sync_token из БД
	var syncToken string
	syncData, err := h.db.GetGoogleSyncToken(calendarID)
	if err != nil {
		log.Printf("[GoogleWebhook] sync token not found for calendar %s, performing full sync", calendarID)
	} else {
		// Игнорируем запросы от старых вебхуков, чтобы избежать дублирования
		if syncData.ChannelId != "" && syncData.ChannelId != channelID {
			log.Printf("[GoogleWebhook] Ignoring webhook from old channelID=%s (current channelID=%s)", channelID, syncData.ChannelId)
			return
		}
		syncToken = syncData.SyncToken
	}

	// Получаем список событий через Google Calendar API
	events, nextSyncToken, err := h.gc.ListEvents(calendarID, syncToken)
	if err != nil {
		log.Printf("[GoogleWebhook] error fetching events: %v", err)
		return
	}

	log.Printf("[GoogleWebhook] processEvents: events=%v", events)

	// Сохраняем новый sync_token
	if nextSyncToken != "" {
		expiresAt := ""
		if syncData != nil {
			expiresAt = syncData.ExpiresAt
		}
		_, err := h.db.UpsertGoogleSyncToken(calendarID, nextSyncToken, channelID, resourceID, expiresAt)
		if err != nil {
			log.Printf("[GoogleWebhook] error saving sync token: %v", err)
		}
	}

	// Обрабатываем полученные события
	for _, event := range events {
		// Пропускаем если статус не "confirmed"
		if event.Status != "confirmed" {
			log.Printf("[GoogleWebhook] processEvents: event %s has status %s", event.Id, event.Status)
			continue
		}

		// Пропускаем если название не содержит "Campus Tour"
		if !strings.Contains(event.Summary, "Campus Tour") {
			log.Printf("[GoogleWebhook] processEvents: event %s has summary %s", event.Id, event.Summary)
			continue
		}

		// Проверяем, есть ли уже такая запись в БД
		_, err := h.db.GetAppointmentByGoogleEventID(event.Id)
		if err == nil {
			// Запись уже существует — пропускаем
			continue
		}

		parseTime := func(eDateTime *calendar.EventDateTime) time.Time {
			if eDateTime == nil {
				return time.Time{}
			}
			if eDateTime.DateTime != "" {
				if t, err := time.Parse(time.RFC3339, eDateTime.DateTime); err == nil {
					return t
				}
			}
			if eDateTime.Date != "" {
				if t, err := time.Parse("2006-01-02", eDateTime.Date); err == nil {
					return t
				}
			}
			return time.Time{}
		}

		startT := parseTime(event.Start)
		endT := parseTime(event.End)

		eventText := event.Summary + " " + event.Description
		campusLoginSent := trySendCampusLogin(context.Background(), h.db, h.cl, eventText, startT, endT, event.Description)

		// Формируем время в RFC3339 для базы данных
		startTimeDB := ""
		endTimeDB := ""
		if event.Start != nil {
			if event.Start.DateTime != "" {
				startTimeDB = event.Start.DateTime
			} else if event.Start.Date != "" {
				startTimeDB = event.Start.Date + "T00:00:00Z"
			}
		}
		if event.End != nil {
			if event.End.DateTime != "" {
				endTimeDB = event.End.DateTime
			} else if event.End.Date != "" {
				endTimeDB = event.End.Date + "T00:00:00Z"
			}
		}

		// Сохраняем новую запись в БД
		_, err = h.db.CreateAppointment(
			event.Id,
			event.Summary,
			startTimeDB,
			endTimeDB,
			event.Status,
			event.Description,
			calendarID,
			campusLoginSent, // CampusLogin default value
			"",              // CreatedAt
		)
		if err != nil {
			log.Printf("[GoogleWebhook] error creating appointment for event %s: %v", event.Id, err)
			continue
		}

		fmt.Printf("[GoogleWebhook] saved new appointment: %s (%s)\n", event.Summary, event.Id)
	}

	log.Printf("[GoogleWebhook] processed %d events, sync complete", len(events))
}

// StopWebhook отключает активный вебхук для текущего календаря
func (h *GoogleHandler) StopWebhook(c *gin.Context) {
	calendarID := defaultCalendarID

	syncData, err := h.db.GetGoogleSyncToken(calendarID)
	if err != nil || syncData == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active webhook found in DB"})
		return
	}

	if syncData.ChannelId == "" || syncData.ResourceId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "webhook channel or resource ID is empty"})
		return
	}

	err = h.gc.StopWatch(syncData.ChannelId, syncData.ResourceId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to stop webhook: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "webhook stopped successfully", "channel_id": syncData.ChannelId})
}

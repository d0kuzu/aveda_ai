package google

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/campuslogin"
	"diaxel/internal/modules/googlecalendar"

	"github.com/gin-gonic/gin"
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

	// Асинхронно обрабатываем события
	go h.processEvents(channelID, resourceID)
}

func (h *GoogleHandler) processEvents(channelID, resourceID string) {
	calendarID := defaultCalendarID

	// Получаем текущий sync_token из БД
	var syncToken string
	syncData, err := h.db.GetGoogleSyncToken(calendarID)
	if err != nil {
		log.Printf("[GoogleWebhook] sync token not found for calendar %s, performing full sync", calendarID)
	} else {
		syncToken = syncData.SyncToken
	}

	// Получаем список событий через Google Calendar API
	events, nextSyncToken, err := h.gc.ListEvents(calendarID, syncToken)
	if err != nil {
		log.Printf("[GoogleWebhook] error fetching events: %v", err)
		return
	}

	// Сохраняем новый sync_token
	if nextSyncToken != "" {
		_, err := h.db.UpsertGoogleSyncToken(calendarID, nextSyncToken, channelID, resourceID)
		if err != nil {
			log.Printf("[GoogleWebhook] error saving sync token: %v", err)
		}
	}

	// Обрабатываем полученные события
	for _, event := range events {
		// Пропускаем если статус не "confirmed"
		if event.Status != "confirmed" {
			continue
		}

		// Пропускаем если название не начинается с "Campus Tour"
		if !strings.HasPrefix(event.Summary, "Campus Tour") {
			continue
		}

		// Проверяем, есть ли уже такая запись в БД
		_, err := h.db.GetAppointmentByGoogleEventID(event.Id)
		if err == nil {
			// Запись уже существует — пропускаем
			continue
		}

		// Извлекаем время начала и окончания
		startTime := ""
		endTime := ""
		if event.Start != nil {
			if event.Start.DateTime != "" {
				startTime = event.Start.DateTime
			} else {
				startTime = event.Start.Date
			}
		}
		if event.End != nil {
			if event.End.DateTime != "" {
				endTime = event.End.DateTime
			} else {
				endTime = event.End.Date
			}
		}

		// Попытка отправить appointment в CampusLogin
		campusLoginSent := false
		
		// Извлекаем номер телефона
		eventText := event.Summary + " " + event.Description
		phoneRegex := regexp.MustCompile(`\+?[1]?[-\s\.]?\(?\d{3}\)?[-\s\.]?\d{3}[-\s\.]?\d{4}`)
		phoneStr := phoneRegex.FindString(eventText)
		
		if phoneStr != "" {
			// Очищаем от нецифровых символов
			digits := regexp.MustCompile(`\D`).ReplaceAllString(phoneStr, "")
			if len(digits) >= 10 {
				phoneSuffix := digits[len(digits)-10:]
				
				// Ищем пользователя в БД по суффиксу телефона
				campusRecord, err := h.db.GetCampusloginByPhone(phoneSuffix)
				if err == nil {
					contactID := int(campusRecord.ContactId)
					programID := int(campusRecord.ProgramId)
					
					// Отправляем Appointment
					err = h.cl.SendAppointment(context.Background(), startTime, endTime, contactID, programID, " ")
					if err == nil {
						campusLoginSent = true
						log.Printf("[GoogleWebhook] successfully sent appointment to CampusLogin for phone %s", phoneSuffix)
					} else {
						log.Printf("[GoogleWebhook] failed to send appointment to CampusLogin for phone %s: %v", phoneSuffix, err)
					}
				} else {
					log.Printf("[GoogleWebhook] user not found in CampusLogin by phone %s: %v", phoneSuffix, err)
				}
			}
		}

		// Сохраняем новую запись в БД
		_, err = h.db.CreateAppointment(
			event.Id,
			event.Summary,
			startTime,
			endTime,
			event.Status,
			event.Description,
			calendarID,
			campusLoginSent, // CampusLogin default value
		)
		if err != nil {
			log.Printf("[GoogleWebhook] error creating appointment for event %s: %v", event.Id, err)
			continue
		}

		fmt.Printf("[GoogleWebhook] saved new appointment: %s (%s)\n", event.Summary, event.Id)
	}

	log.Printf("[GoogleWebhook] processed %d events, sync complete", len(events))
}

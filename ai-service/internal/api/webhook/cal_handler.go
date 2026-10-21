package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"diaxel/internal/api/google"
	"diaxel/internal/constants"
	"diaxel/internal/modules/tgnotifier"

	"github.com/gin-gonic/gin"
)

var (
	phoneRegex    = regexp.MustCompile(`\+?[1]?[-\s\.]?\(?\d{3}\)?[-\s\.]?\d{3}[-\s\.]?\d{4}`)
	nonDigitRegex = regexp.MustCompile(`\D`)
)

// CalWebhookEvent represents the webhook event envelope from Cal.com
type CalWebhookEvent struct {
	TriggerEvent string            `json:"triggerEvent"` // e.g. "BOOKING_CREATED"
	CreatedAt    string            `json:"createdAt"`
	Payload      CalBookingPayload `json:"payload"`
}

// CalBookingPayload represents all necessary booking fields from Cal.com
type CalBookingPayload struct {
	UID                 string                 `json:"uid"`
	BookingID           interface{}            `json:"bookingId"`
	Title               string                 `json:"title"`
	StartTime           string                 `json:"startTime"`
	EndTime             string                 `json:"endTime"`
	Type                string                 `json:"type"`
	EventTypeTitle      string                 `json:"eventTypeTitle,omitempty"`
	Description         string                 `json:"description"`
	AdditionalNotes     string                 `json:"additionalNotes"`
	Location            string                 `json:"location"`
	Status              string                 `json:"status"`
	Organizer           CalOrganizer           `json:"organizer"`
	Attendees           []CalAttendee          `json:"attendees"`
	Responses           map[string]interface{} `json:"responses"`
	UserFieldsResponses map[string]interface{} `json:"userFieldsResponses,omitempty"`
	Metadata            map[string]interface{} `json:"metadata"`
}

// CalOrganizer contains details about the event host
type CalOrganizer struct {
	ID       interface{} `json:"id"`
	Name     string      `json:"name"`
	Email    string      `json:"email"`
	Username string      `json:"username"`
	TimeZone string      `json:"timeZone"`
}

// CalAttendee contains information about an attendee/booker
type CalAttendee struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	TimeZone    string `json:"timeZone"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
	Phone       string `json:"phone,omitempty"`
}

// BookingNotificationInfo stores key booking info for Telegram alerts
type BookingNotificationInfo struct {
	Title       string
	DisplayTime string
	Name        string
	Phone       string
	Email       string
	Location    string
	BookingUID  string
	Notes       string
	ContactID   int
	ProgramID   int
}

// HandleCalWebhook processes Cal.com webhooks, syncing bookings to CampusLogin and sending Telegram notifications
func (h *AIHandler) HandleCalWebhook(c *gin.Context) {
	assistantID := c.Param("assistant_id")

	// Проверяем ассистента — должен быть только Aveda Sinta
	if assistantID != constants.AvedaSintaAssistantID {
		log.Printf("[CalCom Webhook] Request ignored: assistant %s is not Aveda Sinta (%s)", assistantID, constants.AvedaSintaAssistantID)
		c.JSON(http.StatusOK, gin.H{"status": "ignored", "reason": "assistant not supported"})
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[CalCom Webhook] Error reading request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var event CalWebhookEvent
	if err := json.Unmarshal(bodyBytes, &event); err != nil {
		log.Printf("[CalCom Webhook] Failed to unmarshal JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json payload"})
		return
	}

	// Обрабатываем только создание новой записи
	if event.TriggerEvent != "BOOKING_CREATED" {
		log.Printf("[CalCom Webhook] Skipping trigger event: %q (only BOOKING_CREATED is handled for CampusLogin)", event.TriggerEvent)
		c.JSON(http.StatusOK, gin.H{"status": "ignored", "reason": "unsupported trigger event"})
		return
	}

	// Извлекаем ключевые поля записи
	title := event.Payload.Title
	if title == "" {
		title = event.Payload.EventTypeTitle
	}
	if title == "" {
		title = "Campus Tour"
	}

	phone := extractPhone(&event)
	attendeeName, attendeeEmail := extractGuestInfo(&event)
	notes := event.Payload.AdditionalNotes
	if notes == "" {
		notes = getResponseString(event.Payload.Responses, "notes")
	}

	// Парсим время начала и окончания в часовом поясе America/Winnipeg
	loc, err := time.LoadLocation("America/Winnipeg")
	if err != nil {
		loc = time.UTC
	}

	startTime, err := time.Parse(time.RFC3339, event.Payload.StartTime)
	if err != nil {
		startTime, _ = time.Parse(time.RFC3339Nano, event.Payload.StartTime)
	}
	endTime, err := time.Parse(time.RFC3339, event.Payload.EndTime)
	if err != nil {
		endTime, _ = time.Parse(time.RFC3339Nano, event.Payload.EndTime)
	}

	startTimeFormatted := startTime.In(loc).Format("2006-01-02T15:04:05")
	endTimeFormatted := endTime.In(loc).Format("2006-01-02T15:04:05")
	displayTime := fmt.Sprintf("%s - %s", startTime.In(loc).Format("02.01.2006 15:04"), endTime.In(loc).Format("15:04"))

	notifInfo := BookingNotificationInfo{
		Title:       title,
		DisplayTime: displayTime,
		Name:        attendeeName,
		Phone:       phone,
		Email:       attendeeEmail,
		Location:    event.Payload.Location,
		BookingUID:  event.Payload.UID,
		Notes:       notes,
	}

	log.Printf("[CalCom Webhook] New booking %s received | Phone: %s | Guest: %s | Time: %s",
		event.Payload.UID, phone, attendeeName, displayTime)

	// Валидация номера телефона
	digits := nonDigitRegex.ReplaceAllString(phone, "")
	if len(digits) < 10 {
		failReason := fmt.Sprintf("Номер телефона не указан или некорректен (получено: %q)", phone)
		log.Printf("[CalCom Webhook] %s for booking %s", failReason, event.Payload.UID)
		h.notifyCalcomBooking(false, failReason, notifInfo)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "campus_status": "invalid_phone"})
		return
	}
	phoneSuffix := digits[len(digits)-10:]

	if h.db == nil || h.campusLogin == nil {
		failReason := "Сервис CampusLogin или БД не инициализированы"
		log.Printf("[CalCom Webhook] %s", failReason)
		h.notifyCalcomBooking(false, failReason, notifInfo)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "campus_status": "service_unavailable"})
		return
	}

	// Ищем клиента в CampusLogin по 10-значному номеру телефона
	campusRecord, err := h.db.GetCampusloginByPhone(phoneSuffix)
	if err != nil || campusRecord == nil {
		failReason := fmt.Sprintf("Клиент с номером %s не найден в базе CampusLogin", phoneSuffix)
		log.Printf("[CalCom Webhook] %s (err: %v)", failReason, err)
		h.notifyCalcomBooking(false, failReason, notifInfo)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "campus_status": "contact_not_found"})
		return
	}

	contactID := int(campusRecord.ContactId)
	programID := int(campusRecord.ProgramId)
	notifInfo.ContactID = contactID
	notifInfo.ProgramID = programID

	tourTitle := "Campus Tour"
	if campusRecord.FirstName != "" {
		tourTitle = "Campus Tour for " + campusRecord.FirstName
		if notifInfo.Name == "" {
			notifInfo.Name = campusRecord.FirstName
		}
	} else if attendeeName != "" {
		tourTitle = "Campus Tour for " + attendeeName
	}
	if notifInfo.Email == "" && campusRecord.Email != "" {
		notifInfo.Email = campusRecord.Email
	}

	description := event.Payload.Description
	if notes != "" {
		if description != "" {
			description += "\nNotes: " + notes
		} else {
			description = "Notes: " + notes
		}
	}

	// Отправляем встречу в CampusLogin
	err = h.campusLogin.SendAppointment(c.Request.Context(), tourTitle, startTimeFormatted, endTimeFormatted, contactID, programID, description)
	if err != nil {
		failReason := fmt.Sprintf("Ошибка отправки в API CampusLogin: %v", err)
		log.Printf("[CalCom Webhook] %s for phone %s", failReason, phoneSuffix)
		h.notifyCalcomBooking(false, failReason, notifInfo)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "campus_status": "send_failed", "error": err.Error()})
		return
	}

	log.Printf("[CalCom Webhook] Successfully sent appointment to CampusLogin for phone %s (ContactID: %d, ProgramID: %d)",
		phoneSuffix, contactID, programID)

	// Помечаем чаты клиента как забронированные, чтобы прекратить follow-up
	google.MarkChatsBookedByPhone(h.db, phoneSuffix)

	// Отправляем успешное уведомление в Telegram группу
	h.notifyCalcomBooking(true, "", notifInfo)

	c.JSON(http.StatusOK, gin.H{
		"status":        "ok",
		"campus_status": "success",
		"booking_uid":   event.Payload.UID,
		"contact_id":    contactID,
	})
}

// extractPhone извлекает телефон из ответов Cal.com
func extractPhone(event *CalWebhookEvent) string {
	if p := getResponseString(event.Payload.Responses, "phone"); p != "" {
		return p
	}
	if p := getResponseString(event.Payload.UserFieldsResponses, "phone"); p != "" {
		return p
	}
	if p := getResponseString(event.Payload.Responses, "attendeePhoneNumber"); p != "" {
		return p
	}
	for _, att := range event.Payload.Attendees {
		if att.PhoneNumber != "" {
			return att.PhoneNumber
		}
		if att.Phone != "" {
			return att.Phone
		}
	}
	combined := event.Payload.AdditionalNotes + " " + event.Payload.Description + " " + event.Payload.Title
	return phoneRegex.FindString(combined)
}

// extractGuestInfo извлекает имя и email гостя
func extractGuestInfo(event *CalWebhookEvent) (name string, email string) {
	name = getResponseString(event.Payload.Responses, "name")
	if name == "" && len(event.Payload.Attendees) > 0 {
		name = event.Payload.Attendees[0].Name
	}

	email = getResponseString(event.Payload.Responses, "email")
	if email == "" && len(event.Payload.Attendees) > 0 {
		email = event.Payload.Attendees[0].Email
	}
	return
}

// getResponseString извлекает строковое значение из объекта ответа формы Cal.com
func getResponseString(responses map[string]interface{}, key string) string {
	if responses == nil {
		return ""
	}
	entry, ok := responses[key]
	if !ok || entry == nil {
		return ""
	}
	if obj, ok := entry.(map[string]interface{}); ok {
		if val, exists := obj["value"]; exists && val != nil {
			return fmt.Sprintf("%v", val)
		}
	}
	if str, ok := entry.(string); ok {
		return str
	}
	return ""
}

// getTGNotifier возвращает инициализированный tgnotifier.Client или пробует создать его из конфига/переменных окружения
func (h *AIHandler) getTGNotifier() *tgnotifier.Client {
	if h.tgNotifier != nil {
		return h.tgNotifier
	}
	token := h.cfg.CalcomNotifyBotToken
	if token == "" {
		token = os.Getenv("CALCOM_NOTIFY_BOT_TOKEN")
		if token == "" {
			token = os.Getenv("CAL_NOTIFY_BOT_TOKEN")
		}
	}
	chatID := h.cfg.CalcomNotifyChatID
	if chatID == "" {
		chatID = os.Getenv("CALCOM_NOTIFY_CHAT_ID")
		if chatID == "" {
			chatID = os.Getenv("CAL_NOTIFY_CHAT_ID")
		}
	}
	if token != "" && chatID != "" {
		h.tgNotifier = tgnotifier.New(token, chatID)
		return h.tgNotifier
	}
	return nil
}

// notifyCalcomBooking формирует и отправляет уведомление в Telegram группу
func (h *AIHandler) notifyCalcomBooking(success bool, failReason string, b BookingNotificationInfo) {
	notifier := h.getTGNotifier()
	if notifier == nil {
		log.Println("[CalCom Webhook] Telegram notifier not configured (CALCOM_NOTIFY_BOT_TOKEN / CALCOM_NOTIFY_CHAT_ID not set)")
		return
	}

	var sb strings.Builder
	if success {
		sb.WriteString("✅ <b>Новая запись Cal.com перенесена в CampusLogin!</b>\n\n")
	} else {
		sb.WriteString("⚠️ <b>Не удалось перенести запись Cal.com в CampusLogin</b>\n\n")
		if failReason != "" {
			sb.WriteString(fmt.Sprintf("❌ <b>Причина:</b> %s\n\n", html.EscapeString(failReason)))
		}
		sb.WriteString("📌 <b>Ключевые данные записи:</b>\n")
	}

	if b.Title != "" {
		sb.WriteString(fmt.Sprintf("📅 <b>Событие:</b> %s\n", html.EscapeString(b.Title)))
	}
	if b.DisplayTime != "" {
		sb.WriteString(fmt.Sprintf("⏰ <b>Время:</b> %s (Winnipeg)\n", html.EscapeString(b.DisplayTime)))
	}
	if b.Name != "" {
		sb.WriteString(fmt.Sprintf("👤 <b>Клиент:</b> %s\n", html.EscapeString(b.Name)))
	}
	if b.Phone != "" {
		sb.WriteString(fmt.Sprintf("📞 <b>Телефон:</b> %s\n", html.EscapeString(b.Phone)))
	}
	if b.Email != "" {
		sb.WriteString(fmt.Sprintf("📧 <b>Email:</b> %s\n", html.EscapeString(b.Email)))
	}
	if b.Location != "" {
		sb.WriteString(fmt.Sprintf("📍 <b>Локация:</b> %s\n", html.EscapeString(b.Location)))
	}
	if b.ContactID != 0 {
		sb.WriteString(fmt.Sprintf("🆔 <b>Contact ID:</b> %d", b.ContactID))
		if b.ProgramID != 0 {
			sb.WriteString(fmt.Sprintf(" | <b>Program ID:</b> %d", b.ProgramID))
		}
		sb.WriteString("\n")
	}
	if b.BookingUID != "" {
		sb.WriteString(fmt.Sprintf("🔖 <b>Booking UID:</b> <code>%s</code>\n", html.EscapeString(b.BookingUID)))
	}
	if b.Notes != "" {
		sb.WriteString(fmt.Sprintf("📝 <b>Заметки:</b> %s\n", html.EscapeString(b.Notes)))
	}

	if err := notifier.SendHTML(sb.String()); err != nil {
		log.Printf("[CalCom Webhook] Failed to send Telegram notification: %v", err)
	} else {
		log.Println("[CalCom Webhook] Telegram notification sent successfully to group")
	}
}

package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// CalWebhookEvent represents the standard webhook event envelope from Cal.com
type CalWebhookEvent struct {
	TriggerEvent string            `json:"triggerEvent"` // e.g. "BOOKING_CREATED", "BOOKING_RESCHEDULED", "BOOKING_CANCELLED", "BOOKING_REJECTED", "BOOKING_REQUESTED", "MEETING_ENDED"
	CreatedAt    string            `json:"createdAt"`    // ISO-8601 timestamp
	Payload      CalBookingPayload `json:"payload"`      // Booking event data
}

// CalBookingPayload represents all fields provided in the Cal.com webhook payload
type CalBookingPayload struct {
	// Identifiers
	UID              string      `json:"uid"`
	BookingID        interface{} `json:"bookingId"`
	ICalUID          string      `json:"iCalUID"`
	EventTypeID      interface{} `json:"eventTypeId"`
	SeatReferenceUID string      `json:"seatReferenceUid,omitempty"`
	RescheduleUID    string      `json:"rescheduleUid,omitempty"`

	// Event information
	Type                 string      `json:"type"`
	Title                string      `json:"title"`
	Description          string      `json:"description"`
	AdditionalNotes      string      `json:"additionalNotes"`
	CustomInputs         interface{} `json:"customInputs,omitempty"`
	StartTime            string      `json:"startTime"`
	EndTime              string      `json:"endTime"`
	Location             string      `json:"location"`
	Status               string      `json:"status"`
	EventTitle           string      `json:"eventTitle,omitempty"`
	EventDescription     string      `json:"eventDescription,omitempty"`
	Price                interface{} `json:"price,omitempty"`
	Currency             string      `json:"currency,omitempty"`
	Length               interface{} `json:"length,omitempty"`
	HideCalendarNotes    bool        `json:"hideCalendarNotes,omitempty"`
	RequiresConfirmation *bool       `json:"requiresConfirmation,omitempty"`

	// Reschedule / Cancellation / Rejection info
	RescheduleStartTime string `json:"rescheduleStartTime,omitempty"`
	RescheduleEndTime   string `json:"rescheduleEndTime,omitempty"`
	CancellationReason  string `json:"cancellationReason,omitempty"`
	RejectionReason     string `json:"rejectionReason,omitempty"`

	// Host / Organizer
	Organizer CalOrganizer `json:"organizer"`

	// Participants / Attendees
	Attendees []CalAttendee `json:"attendees"`
	User      interface{}   `json:"user,omitempty"`

	// Custom responses / questions filled by the booker
	Responses           map[string]interface{} `json:"responses"`
	Metadata            map[string]interface{} `json:"metadata"`
	VideoCallData       interface{}            `json:"videoCallData,omitempty"`
	DestinationCalendar interface{}            `json:"destinationCalendar,omitempty"`

	// Calendar ICS contents (in newer Cal.com webhook schema versions)
	AttendeeIcsContent  string `json:"attendeeIcsContent,omitempty"`
	OrganizerIcsContent string `json:"organizerIcsContent,omitempty"`

	// OAuth platform client ID
	PlatformOAuthClientID interface{} `json:"platformOAuthClientId,omitempty"`
}

// CalOrganizer contains details about the event host
type CalOrganizer struct {
	ID         interface{} `json:"id"`
	Name       string      `json:"name"`
	Email      string      `json:"email"`
	Username   string      `json:"username"`
	TimeZone   string      `json:"timeZone"`
	TimeFormat string      `json:"timeFormat,omitempty"`
	UTCOffset  interface{} `json:"utcOffset,omitempty"`
}

// CalAttendee contains information about an attendee/booker
type CalAttendee struct {
	Name        string      `json:"name"`
	Email       string      `json:"email"`
	TimeZone    string      `json:"timeZone"`
	Language    interface{} `json:"language,omitempty"`
	PhoneNumber string      `json:"phoneNumber,omitempty"`
	Phone       string      `json:"phone,omitempty"`
}

// HandleCalWebhook processes incoming webhooks from Cal.com
func (h *AIHandler) HandleCalWebhook(c *gin.Context) {
	assistantID := c.Param("assistant_id")
	if assistantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assistant_id is required"})
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[CalCom Webhook] Error reading request body: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	signature := c.GetHeader("X-Cal-Signature-256")
	webhookVersion := c.GetHeader("X-Cal-Webhook-Version")

	// Verify webhook signature if secret is configured
	secret := os.Getenv("CAL_WEBHOOK_SECRET")
	if secret != "" && signature != "" {
		if !verifyCalSignature(bodyBytes, signature, secret) {
			log.Printf("[CalCom Webhook] Warning: Signature verification failed for signature: %s\n", signature)
		} else {
			log.Printf("[CalCom Webhook] Signature verified successfully.\n")
		}
	}

	var event CalWebhookEvent
	if err := json.Unmarshal(bodyBytes, &event); err != nil {
		log.Printf("[CalCom Webhook] Failed to unmarshal CalWebhookEvent: %v. Raw body: %s\n", err, string(bodyBytes))
	}

	// Attempt to lookup assistant details if assistantID was passed
	var assistantName string
	if assistantID != "" && h.db != nil {
		if assistant, err := h.db.GetAssistant(assistantID); err == nil && assistant != nil {
			assistantName = assistant.Name
		}
	}

	// Structured logs for testing and debugging
	log.Println("================== [CAL.COM WEBHOOK RECEIVED] ==================")
	log.Printf("[CalCom Webhook] Method: %s | Path: %s | Remote: %s\n", c.Request.Method, c.Request.URL.Path, c.ClientIP())
	if assistantID != "" {
		if assistantName != "" {
			log.Printf("[CalCom Webhook] Assistant: %s (ID: %s)\n", assistantName, assistantID)
		} else {
			log.Printf("[CalCom Webhook] Assistant ID: %s\n", assistantID)
		}
	}
	if webhookVersion != "" {
		log.Printf("[CalCom Webhook] Webhook Version: %s\n", webhookVersion)
	}
	if signature != "" {
		log.Printf("[CalCom Webhook] Signature: %s\n", signature)
	}
	log.Printf("[CalCom Webhook] Trigger Event: %s\n", event.TriggerEvent)
	log.Printf("[CalCom Webhook] Created At: %s\n", event.CreatedAt)

	log.Printf("[CalCom Webhook] Booking UID: %s | Booking ID: %v | EventTypeID: %v\n",
		event.Payload.UID, event.Payload.BookingID, event.Payload.EventTypeID)
	log.Printf("[CalCom Webhook] Title: %q | Type: %q | Status: %s\n",
		event.Payload.Title, event.Payload.Type, event.Payload.Status)
	log.Printf("[CalCom Webhook] Start: %s | End: %s\n", event.Payload.StartTime, event.Payload.EndTime)
	log.Printf("[CalCom Webhook] Location: %s\n", event.Payload.Location)

	log.Printf("[CalCom Webhook] Organizer: %s <%s> (ID: %v, TimeZone: %s)\n",
		event.Payload.Organizer.Name,
		event.Payload.Organizer.Email,
		event.Payload.Organizer.ID,
		event.Payload.Organizer.TimeZone,
	)

	for i, att := range event.Payload.Attendees {
		phone := att.PhoneNumber
		if phone == "" {
			phone = att.Phone
		}
		log.Printf("[CalCom Webhook] Attendee #%d: %s <%s> | Phone: %s | TimeZone: %s\n",
			i+1, att.Name, att.Email, phone, att.TimeZone)
	}

	if len(event.Payload.Responses) > 0 {
		log.Println("[CalCom Webhook] Form Responses:")
		for key, val := range event.Payload.Responses {
			log.Printf("  - %s: %+v\n", key, val)
		}
	}

	if len(event.Payload.Metadata) > 0 {
		log.Println("[CalCom Webhook] Metadata:")
		for key, val := range event.Payload.Metadata {
			log.Printf("  - %s: %+v\n", key, val)
		}
	}

	if event.Payload.CancellationReason != "" {
		log.Printf("[CalCom Webhook] Cancellation Reason: %s\n", event.Payload.CancellationReason)
	}
	if event.Payload.RejectionReason != "" {
		log.Printf("[CalCom Webhook] Rejection Reason: %s\n", event.Payload.RejectionReason)
	}
	if event.Payload.RescheduleUID != "" {
		log.Printf("[CalCom Webhook] Reschedule UID: %s | New Range: %s -> %s\n",
			event.Payload.RescheduleUID, event.Payload.RescheduleStartTime, event.Payload.RescheduleEndTime)
	}

	// Log full raw JSON formatted for testing inspection
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, bodyBytes, "", "  "); err == nil {
		log.Printf("[CalCom Webhook] Full Payload JSON:\n%s\n", prettyJSON.String())
	} else {
		log.Printf("[CalCom Webhook] Full Payload Raw: %s\n", string(bodyBytes))
	}
	log.Println("================================================================")

	c.JSON(http.StatusOK, gin.H{
		"status":       "ok",
		"message":      "webhook received successfully",
		"triggerEvent": event.TriggerEvent,
		"bookingUid":   event.Payload.UID,
		"assistantId":  assistantID,
	})
}

// verifyCalSignature verifies the HMAC-SHA256 signature from Cal.com
func verifyCalSignature(body []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedMAC), []byte(signature))
}

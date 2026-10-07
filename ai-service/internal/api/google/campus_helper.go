package google

import (
	"context"
	"log"
	"regexp"
	"time"

	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/campuslogin"
)

var phoneRegex = regexp.MustCompile(`\+?[1]?[-\s\.]?\(?\d{3}\)?[-\s\.]?\d{3}[-\s\.]?\d{4}`)
var nonDigitRegex = regexp.MustCompile(`\D`)

// TrySendCampusLogin извлекает телефон из текста, ищет запись в БД и отправляет назначение в CampusLogin.
func TrySendCampusLogin(ctx context.Context, dbClient *db.Client, cl *campuslogin.Client, text string, start, end time.Time, description string) bool {
	if cl == nil || dbClient == nil {
		return false
	}

	phoneStr := phoneRegex.FindString(text)
	if phoneStr == "" {
		return false
	}

	digits := nonDigitRegex.ReplaceAllString(phoneStr, "")
	if len(digits) < 10 {
		return false
	}
	phoneSuffix := digits[len(digits)-10:]

	campusRecord, err := dbClient.GetCampusloginByPhone(phoneSuffix)
	if err != nil {
		log.Printf("[CampusLoginHelper] user not found in CampusLogin by phone %s: %v", phoneSuffix, err)
		return false
	}

	loc, err := time.LoadLocation("America/Winnipeg")
	if err != nil {
		loc = time.UTC
	}

	startTimeFormatted := start.In(loc).Format("2006-01-02T15:04:05")
	endTimeFormatted := end.In(loc).Format("2006-01-02T15:04:05")

	log.Printf("[CampusLoginHelper] start time: %s", startTimeFormatted)
	log.Printf("[CampusLoginHelper] end time: %s", endTimeFormatted)
	contactID := int(campusRecord.ContactId)
	log.Printf("[CampusLoginHelper] Contact ID: %d", contactID)
	programID := int(campusRecord.ProgramId)
	log.Printf("[CampusLoginHelper] Program ID: %d", programID)

	err = cl.SendAppointment(ctx, "Campus Tour for "+campusRecord.FirstName, startTimeFormatted, endTimeFormatted, contactID, programID, description)
	if err != nil {
		log.Printf("[CampusLoginHelper] failed to send appointment to CampusLogin for phone %s: %v", phoneSuffix, err)
		return false
	}

	log.Printf("[CampusLoginHelper] successfully sent appointment to CampusLogin for phone %s", phoneSuffix)
	return true
}

// MarkChatsBookedByPhone извлекает телефон из текста бронирования и помечает все активные чаты
// этого клиента как забронированные и завершённые, чтобы follow-up сообщения больше не отправлялись.
func MarkChatsBookedByPhone(dbClient *db.Client, text string) {
	if dbClient == nil {
		return
	}

	phoneStr := phoneRegex.FindString(text)
	if phoneStr == "" {
		log.Printf("[CampusLoginHelper] MarkChatsBookedByPhone: no phone found in booking text")
		return
	}
	digits := nonDigitRegex.ReplaceAllString(phoneStr, "")
	if len(digits) < 10 {
		return
	}
	phoneSuffix := digits[len(digits)-10:]

	chats, err := dbClient.GetChatsForFollowup()
	if err != nil {
		log.Printf("[CampusLoginHelper] MarkChatsBookedByPhone: failed to get active chats: %v", err)
		return
	}

	for _, chat := range chats {
		customerDigits := nonDigitRegex.ReplaceAllString(chat.CustomerId, "")
		if len(customerDigits) < 10 || customerDigits[len(customerDigits)-10:] != phoneSuffix {
			continue
		}
		if _, err := dbClient.UpdateChatIsBooked(chat.Id, true); err != nil {
			log.Printf("[CampusLoginHelper] failed to set is_booked for chat %s: %v", chat.Id, err)
		}
		if _, err := dbClient.UpdateChatIsEnd(chat.Id, true); err != nil {
			log.Printf("[CampusLoginHelper] failed to set is_end for chat %s: %v", chat.Id, err)
		}
		log.Printf("[CampusLoginHelper] chat %s (customer %s) marked as booked, follow-ups stopped", chat.Id, chat.CustomerId)
	}
}

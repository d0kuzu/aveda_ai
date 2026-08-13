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

// trySendCampusLogin извлекает телефон из текста, ищет запись в БД и отправляет назначение в CampusLogin.
func trySendCampusLogin(ctx context.Context, dbClient *db.Client, cl *campuslogin.Client, text string, start, end time.Time, description string) bool {
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

package followup

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"diaxel/internal/constants"
	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/twilio"
)

type Listener struct {
	dbClient     *db.Client
	twilioClient *twilio.Client
}

func NewListener(dbClient *db.Client, twilioClient *twilio.Client) *Listener {
	return &Listener{
		dbClient:     dbClient,
		twilioClient: twilioClient,
	}
}

func (l *Listener) writeLog(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Print("[Followup Listener] " + msg)
	
	f, err := os.OpenFile("../followup.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		f, err = os.OpenFile("followup.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}
	if err == nil {
		defer f.Close()
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		f.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, msg))
	}
}

const fallbackText = "Enrolling at Aveda is a simple process. When are you looking to start school: right away or in the near future?"

func (l *Listener) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	l.writeLog("Started.")
	for {
		select {
		case <-ctx.Done():
			l.writeLog("Stopped.")
			return
		case <-ticker.C:
			l.processFollowups(ctx)
		}
	}
}

func (l *Listener) processFollowups(ctx context.Context) {
	loc, err := time.LoadLocation("America/Winnipeg")
	if err != nil {
		l.writeLog("Error loading timezone: %v", err)
		return
	}
	now := time.Now().In(loc)
	if hour := now.Hour(); hour < 9 || hour >= 18 {
		l.writeLog("Outside working hours (%d). Skipping.", hour)
		return
	}
	l.writeLog("Working")

	chats, err := l.dbClient.GetChatsForFollowup()
	if err != nil {
		l.writeLog("Error getting chats: %v", err)
		return
	}

	for _, chat := range chats {
		if chat.CustomerId == "" {
			l.writeLog("Chat %s has empty CustomerId, skipping.", chat.Id)
			continue
		}
		if chat.IsBooked {
			l.writeLog("Chat %s (%s) already booked a tour. Ending chat.", chat.Id, chat.CustomerId)
			l.dbClient.UpdateChatIsEnd(chat.Id, true)
			continue
		}
		campusLogin, err := l.dbClient.GetCampusloginByUserId(chat.CustomerId)
		if err != nil {
			l.writeLog("Could not get campuslogin for %s: %v, skipping.", chat.CustomerId, err)
			continue
		}
		if campusLogin.IsGrade11OrLower || campusLogin.IsInternationalStudent {
			l.writeLog("%s is unqualified (Grade11OrLower: %v, International: %v). Ending chat.", chat.CustomerId, campusLogin.IsGrade11OrLower, campusLogin.IsInternationalStudent)
			l.dbClient.UpdateChatIsEnd(chat.Id, true)
			continue
		}
		programIDStr := fmt.Sprintf("%d", campusLogin.ProgramId)
		programName, ok := constants.ProgramIDToName[programIDStr]
		if !ok {
			l.writeLog("Program ID %s not found in constants for %s, skipping.", programIDStr, chat.CustomerId)
			continue
		}
		if programName == "Hairstyling (evening)" {
			programName = "Hairstyling"
		}
		schedule, hasSchedule := constants.FollowupSchedules[programName]
		if !hasSchedule {
			l.writeLog("No followup schedule for program '%s' (Customer: %s), skipping.", programName, chat.CustomerId)
			continue
		}
		currentStage := int(chat.FollowupStage)
		stageConfig, hasStage := schedule[currentStage]
		if !hasStage {
			l.writeLog("No stage config for stage %d (Program: '%s', Customer: %s). Ending chat.", currentStage, programName, chat.CustomerId)
			l.dbClient.UpdateChatIsEnd(chat.Id, true)
			continue
		}
		updatedAt, err := time.Parse(time.RFC3339, chat.UpdatedAt)
		if err != nil {
			l.writeLog("Error parsing UpdatedAt '%s' for %s: %v. Skipping.", chat.UpdatedAt, chat.CustomerId, err)
			continue
		}
		if time.Since(updatedAt) < stageConfig.Delay {
			continue
		}
		twilioConfig, err := l.dbClient.GetTwilioConfig(chat.AssistantId)
		if err != nil {
			l.writeLog("Could not get twilio config for assistant %s (Customer: %s): %v. Skipping.", chat.AssistantId, chat.CustomerId, err)
			continue
		}

		if programName == "Hairstyling" && currentStage == 3 {
			if strings.Contains(stageConfig.Text, "{FirstName}") {
				if campusLogin.FirstName == "" {
					stageConfig.Text = fallbackText
				} else {
					stageConfig.Text = strings.ReplaceAll(stageConfig.Text, "{FirstName}", campusLogin.FirstName)
				}
			}
		}

		if err = l.twilioClient.SendMessage(ctx, twilioConfig.AccountSid, twilioConfig.AuthToken, twilioConfig.TwilioNumber, chat.CustomerId, stageConfig.Text); err != nil {
			l.writeLog("Error sending Twilio message to %s at stage %d: %v", chat.CustomerId, currentStage, err)
			continue
		}
		l.dbClient.SaveMessage(chat.Id, "assistant", stageConfig.Text, "twilio")
		if _, hasNext := schedule[currentStage+1]; hasNext {
			l.dbClient.UpdateChatFollowupStage(chat.Id, int32(currentStage+1))
			l.writeLog("SUCCESS: Sent followup %d to %s. Stage incremented to %d.", currentStage, chat.CustomerId, currentStage+1)
		} else {
			l.dbClient.UpdateChatIsEnd(chat.Id, true)
			l.writeLog("SUCCESS: Sent final followup %d to %s. Chat ended.", currentStage, chat.CustomerId)
		}
	}
	l.writeLog("Work ended")
}

package followup

import (
	"context"
	"fmt"
	"log"
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

func (l *Listener) Start(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	log.Println("[Followup Listener] Started.")
	for {
		select {
		case <-ctx.Done():
			log.Println("[Followup Listener] Stopped.")
			return
		case <-ticker.C:
			l.processFollowups(ctx)
		}
	}
}

func (l *Listener) processFollowups(ctx context.Context) {
	//loc, err := time.LoadLocation("America/Winnipeg")
	//if err != nil {
	//	log.Printf("[Followup Listener] Error loading timezone: %v\n", err)
	//	return
	//}
	//now := time.Now().In(loc)
	//if hour := now.Hour(); hour < 9 || hour >= 18 {
	//	log.Printf("[Followup Listener] Outside working hours (%d). Skipping.\n", hour)
	//	return
	//}
	log.Printf("[Followup Listener] Working")

	chats, err := l.dbClient.GetChatsForFollowup()
	if err != nil {
		log.Printf("[Followup Listener] Error getting chats: %v\n", err)
		return
	}

	for _, chat := range chats {
		if chat.CustomerId == "" {
			continue
		}
		campusLogin, err := l.dbClient.GetCampusloginByUserId(chat.CustomerId)
		if err != nil {
			log.Printf("[Followup Listener] Could not get campuslogin for %s: %v\n", chat.CustomerId, err)
			continue
		}
		if campusLogin.IsGrade11OrLower || campusLogin.IsInternationalStudent {
			log.Printf("[Followup Listener] %s is unqualified. Ending chat.", chat.CustomerId)
			l.dbClient.UpdateChatIsEnd(chat.Id, true)
			continue
		}
		programIDStr := fmt.Sprintf("%d", campusLogin.ProgramId)
		programName, ok := constants.ProgramIDToName[programIDStr]
		if !ok {
			continue
		}
		if programName == "Hairstyling (evening)" {
			programName = "Hairstyling"
		}
		schedule, hasSchedule := constants.FollowupSchedules[programName]
		if !hasSchedule {
			continue
		}
		currentStage := int(chat.FollowupStage)
		stageConfig, hasStage := schedule[currentStage]
		if !hasStage {
			l.dbClient.UpdateChatIsEnd(chat.Id, true)
			continue
		}
		updatedAt, err := time.Parse(time.RFC3339, chat.UpdatedAt)
		if err != nil {
			continue
		}
		if time.Since(updatedAt) < stageConfig.Delay {
			continue
		}
		twilioConfig, err := l.dbClient.GetTwilioConfig(chat.AssistantId)
		if err != nil {
			continue
		}
		if err = l.twilioClient.SendMessage(ctx, twilioConfig.AccountSid, twilioConfig.AuthToken, twilioConfig.TwilioNumber, chat.CustomerId, stageConfig.Text); err != nil {
			log.Printf("[Followup Listener] Error sending to %s: %v\n", chat.CustomerId, err)
			continue
		}
		l.dbClient.SaveMessage(chat.Id, "assistant", stageConfig.Text, "twilio")
		if _, hasNext := schedule[currentStage+1]; hasNext {
			l.dbClient.UpdateChatFollowupStage(chat.Id, int32(currentStage+1))
			log.Printf("[Followup Listener] Stage %d -> %d for %s\n", currentStage, currentStage+1, chat.CustomerId)
		} else {
			l.dbClient.UpdateChatIsEnd(chat.Id, true)
			log.Printf("[Followup Listener] Final stage %d for %s, chat ended\n", currentStage, chat.CustomerId)
		}
	}
	log.Printf("[Followup Listener] Work ended")
}

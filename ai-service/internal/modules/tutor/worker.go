package tutor

import (
	"context"
	"fmt"
	"log"
	"time"

	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/twilio"
)

type Worker struct {
	dbClient     *db.Client
	twilioClient *twilio.Client
}

func NewWorker(dbClient *db.Client, twilioClient *twilio.Client) *Worker {
	return &Worker{
		dbClient:     dbClient,
		twilioClient: twilioClient,
	}
}

func (w *Worker) Start(ctx context.Context) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		log.Printf("[Tutor Worker] Warning: failed to load America/Los_Angeles timezone (%v), using fixed PST/PDT offset", err)
		loc = time.FixedZone("PST", -8*3600)
	}

	log.Println("[Tutor Worker] Started (scheduled daily at 12:00 America/Los_Angeles).")

	for {
		now := time.Now().In(loc)
		next := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, loc)
		if !now.Before(next) {
			next = next.Add(24 * time.Hour)
		}
		duration := next.Sub(now)

		select {
		case <-ctx.Done():
			log.Println("[Tutor Worker] Stopped.")
			return
		case <-time.After(duration):
			w.processTutorChats(ctx)
		}
	}
}

func (w *Worker) processTutorChats(ctx context.Context) {
	log.Println("[Tutor Worker] Running daily check for active tutor chats...")

	chats, err := w.dbClient.GetActiveTutorChats()
	if err != nil {
		log.Printf("[Tutor Worker] Error getting active tutor chats: %v", err)
		return
	}

	for _, chat := range chats {
		if chat.CustomerId == "" || chat.IsEnd || !chat.IsTutor {
			continue
		}

		isBlocked, err := w.dbClient.IsCustomerBlocked(chat.CustomerId)
		if err == nil && isBlocked {
			log.Printf("[Tutor Worker] Skipping blocked customer %s", chat.CustomerId)
			continue
		}

		if chat.UpdatedAt != "" {
			if updatedAt, err := time.Parse(time.RFC3339, chat.UpdatedAt); err == nil {
				if time.Since(updatedAt) < 2*time.Hour {
					log.Printf("[Tutor Worker] Chat %s (%s) was active recently (%s), skipping daily reminder.", chat.Id, chat.CustomerId, chat.UpdatedAt)
					continue
				}
			}
		}

		twilioConfig, err := w.dbClient.GetTwilioConfig(chat.AssistantId)
		if err != nil || twilioConfig == nil {
			log.Printf("[Tutor Worker] Could not get Twilio config for assistant %s (chat %s): %v", chat.AssistantId, chat.Id, err)
			continue
		}

		var msgText string

		if !chat.IsTuting {
			// Case 1: Student hasn't started tutoring yet
			if chat.TutingProgram == "" {
				if IsCanadaAssistant(chat.AssistantId) {
					msgText = "Hi there! Just checking in from Exam Prep. Which program would you like to choose to start practicing questions?\n1. Esthetician / Skin Care\n2. Nail Technician\n3. Barbering\n4. Hair Styling"
				} else {
					msgText = "Hi there! Just checking in from State Board Exam Prep. Which program would you like to choose to start practicing questions?\n1. Barber\n2. Cosmetology\n3. Esthetician\n4. Nail Technician"
				}
			} else {
				msgText = fmt.Sprintf(
					"Hi! Just checking in — are you ready to start practicing your %s exam questions today?",
					ProgramDisplayNameForAssistant(chat.AssistantId, chat.TutingProgram),
				)
			}
		} else if chat.TutorCountPerDay >= 5 {
			// Case 2: Student completed yesterday's 5 questions -> start a new daily set
			q, total, qErr := GetQuestionForAssistant(chat.AssistantId, chat.TutingProgram, int(chat.TutorCount))
			if qErr != nil {
				log.Printf("[Tutor Worker] Error getting question for chat %s (program=%s, idx=%d): %v", chat.Id, chat.TutingProgram, chat.TutorCount, qErr)
				continue
			}
			if q == nil {
				_, _ = w.dbClient.UpdateChatIsEnd(chat.Id, true)
				continue
			}

			_, err = w.dbClient.UpdateChatTutorState(chat.Id, true, chat.TutingProgram, 0, chat.TutorCount)
			if err != nil {
				log.Printf("[Tutor Worker] Error resetting daily counter for chat %s: %v", chat.Id, err)
				continue
			}

			msgText = fmt.Sprintf(
				"Hi! Ready for today's set of 5 %s practice questions? Here is Question %d of %d:\n\n%s",
				ProgramDisplayNameForAssistant(chat.AssistantId, chat.TutingProgram),
				int(chat.TutorCount)+1,
				total,
				FormatQuestion(q),
			)
		} else {
			// Case 3: Previous question was left unanswered -> re-send the same question and reset daily counter
			q, total, qErr := GetQuestionForAssistant(chat.AssistantId, chat.TutingProgram, int(chat.TutorCount))
			if qErr != nil {
				log.Printf("[Tutor Worker] Error getting hanging question for chat %s (program=%s, idx=%d): %v", chat.Id, chat.TutingProgram, chat.TutorCount, qErr)
				continue
			}
			if q == nil {
				_, _ = w.dbClient.UpdateChatIsEnd(chat.Id, true)
				continue
			}

			_, err = w.dbClient.UpdateChatTutorState(chat.Id, true, chat.TutingProgram, 0, chat.TutorCount)
			if err != nil {
				log.Printf("[Tutor Worker] Error resetting daily counter for chat %s: %v", chat.Id, err)
				continue
			}

			msgText = fmt.Sprintf(
				"Hi! Last time we left off on this question — let's continue from here!\n\nQuestion %d of %d:\n%s",
				int(chat.TutorCount)+1,
				total,
				FormatQuestion(q),
			)
		}

		if err := w.twilioClient.SendMessage(ctx, twilioConfig.AccountSid, twilioConfig.AuthToken, twilioConfig.TwilioNumber, chat.CustomerId, msgText); err != nil {
			log.Printf("[Tutor Worker] Failed to send Twilio SMS to %s (chat %s): %v", chat.CustomerId, chat.Id, err)
			continue
		}

		if _, err := w.dbClient.SaveTutorMessage(chat.Id, "assistant", msgText, "twilio"); err != nil {
			log.Printf("[Tutor Worker] Failed to save tutor message for chat %s: %v", chat.Id, err)
		}

		log.Printf("[Tutor Worker] Sent daily tutor message to %s (chat %s, is_tuting=%v, program=%s, idx=%d)",
			chat.CustomerId, chat.Id, chat.IsTuting, chat.TutingProgram, chat.TutorCount)
	}

	log.Println("[Tutor Worker] Daily check completed.")
}

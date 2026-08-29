package summary

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"diaxel/internal/constants"
	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/campuslogin"
	"diaxel/internal/modules/llm"

	"github.com/sashabaranov/go-openai"
)

type AbandonedSummarizer struct {
	dbClient     *db.Client
	campusClient *campuslogin.Client
	llmClient    *llm.Client
}

func NewAbandonedSummarizer(dbClient *db.Client, campusClient *campuslogin.Client, llmClient *llm.Client) *AbandonedSummarizer {
	return &AbandonedSummarizer{
		dbClient:     dbClient,
		campusClient: campusClient,
		llmClient:    llmClient,
	}
}

func (s *AbandonedSummarizer) writeLog(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Print("[AbandonedSummarizer] " + msg)

	f, err := os.OpenFile("../summary_work.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		f, err = os.OpenFile("summary_work.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}
	if err == nil {
		defer f.Close()
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		f.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, msg))
	}
}

func (s *AbandonedSummarizer) Start(ctx context.Context) {
	// Calculate time until next 11:00 AM in America/Winnipeg
	loc, err := time.LoadLocation("America/Winnipeg")
	if err != nil {
		log.Printf("[AbandonedSummarizer] Error loading timezone: %v\n", err)
		return
	}

	for {
		now := time.Now().In(loc)
		next := time.Date(now.Year(), now.Month(), now.Day(), 11, 0, 0, 0, loc)
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		duration := next.Sub(now)

		select {
		case <-ctx.Done():
			log.Println("[AbandonedSummarizer] Stopped.")
			return
		case <-time.After(duration):
			s.processAbandonedChats(ctx)
		}
	}
}

func (s *AbandonedSummarizer) processAbandonedChats(ctx context.Context) {
	s.writeLog("Starting daily summarization...")

	chats, err := s.dbClient.GetUnreviewedActiveChats()
	if err != nil {
		s.writeLog("Error getting unreviewed active chats: %v", err)
		return
	}

	for _, chat := range chats {
		if chat.CustomerId == "" {
			continue
		}

		updatedAt, err := time.Parse(time.RFC3339, chat.UpdatedAt)
		if err != nil {
			s.writeLog("Error parsing UpdatedAt '%s' for chat %s: %v", chat.UpdatedAt, chat.Id, err)
			continue
		}

		if time.Since(updatedAt) < 1*time.Hour {
			s.writeLog("Chat %s for customer %s was active recently, skipping.", chat.Id, chat.CustomerId)
			continue
		}

		messages, err := s.dbClient.GetAllChatMessages(chat.Id)
		if err != nil {
			s.writeLog("Error getting messages for chat %s: %v", chat.Id, err)
			continue
		}

		if len(messages) == 0 {
			continue
		}

		hasUserMessage := false
		var userMessagesText string
		for _, msg := range messages {
			if msg.Role == "user" {
				hasUserMessage = true
			}
			userMessagesText += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
		}

		var summary string
		if !hasUserMessage {
			s.writeLog("Chat %s (customer %s): no user messages — using static summary.", chat.Id, chat.CustomerId)
			summary = "Lead didn't answer on the first message."
		} else {
			// Ask LLM for summary
			prompt := []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: constants.AbandonedChatSummaryPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userMessagesText,
				},
			}
			response, err := s.llmClient.GetAnswer(ctx, "", prompt)
			if err != nil || len(response.Choices) == 0 {
				s.writeLog("Error getting LLM answer for chat %s: %v", chat.Id, err)
				continue
			}
			summary = response.Choices[0].Message.Content
		}

		summary += fmt.Sprintf("\n\nFollowup Stage: %d", chat.FollowupStage)

		campusRecord, err := s.dbClient.GetCampusloginByUserId(chat.CustomerId)
		var contactID int
		if err != nil {
			s.writeLog("Failed to get ContactID for user %s: %v. Using fallback.", chat.CustomerId, err)
			contactID = 5972449 // default fallback
		} else {
			contactID = int(campusRecord.ContactId)
		}

		s.writeLog("Sending summary for customer %s (contact %d, stage %d): %s", chat.CustomerId, contactID, chat.FollowupStage, summary)
		err = s.campusClient.AddNewNote(ctx, contactID, summary)
		if err != nil {
			s.writeLog("Failed to add note for chat %s: %v", chat.Id, err)
			continue
		}

		_, err = s.dbClient.UpdateChatIsReviewed(chat.Id, true)
		if err != nil {
			s.writeLog("Failed to update chat is_reviewed for %s: %v", chat.Id, err)
		} else {
			s.writeLog("SUCCESS: Summarized abandoned chat for user %s.", chat.CustomerId)
		}
	}

	s.writeLog("Finished daily summarization.")
}

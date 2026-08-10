package twilio

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

func (h *TwilioWebhookHandler) HandleRecovery(c *gin.Context) {
	assistantID := c.Param("assistant_id")
	if assistantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assistant_id is required"})
		return
	}

	_, err := h.db.GetAssistant(assistantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assistant not found"})
		return
	}

	twilioConfig, err := h.db.GetTwilioConfig(assistantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "twilio configuration not found"})
		return
	}

	// Trigger background recovery job
	go h.runRecoveryJob(assistantID, twilioConfig.AccountSid, twilioConfig.AuthToken, twilioConfig.TwilioNumber)

	c.JSON(http.StatusOK, gin.H{"status": "recovery job started", "message": "Check server logs for progress"})
}

// twilioMessage is a lightweight wrapper for sorting
type twilioMessage struct {
	From     string
	To       string
	Body     string
	DateSent time.Time
}

func parseTwilioMessages(raw []openapi.ApiV2010Message) []twilioMessage {
	var result []twilioMessage
	for _, msg := range raw {
		if msg.From == nil || msg.To == nil || msg.Body == nil || msg.DateSent == nil {
			continue
		}
		t, err := time.Parse(time.RFC1123Z, *msg.DateSent)
		if err != nil {
			// Try alternative format
			t, err = time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", *msg.DateSent)
			if err != nil {
				log.Printf("[Recovery Job] Failed to parse date '%s': %v", *msg.DateSent, err)
				continue
			}
		}
		result = append(result, twilioMessage{
			From:     *msg.From,
			To:       *msg.To,
			Body:     *msg.Body,
			DateSent: t,
		})
	}
	return result
}

func (h *TwilioWebhookHandler) runRecoveryJob(assistantID, accountSID, authToken, twilioNumber string) {
	log.Printf("[Recovery Job] ========================================")
	log.Printf("[Recovery Job] Starting recovery for assistant %s", assistantID)
	log.Printf("[Recovery Job] Twilio number: %s", twilioNumber)
	log.Printf("[Recovery Job] ========================================")

	// Fetch from May 1, 2026 to today
	startDate, _ := time.Parse("2006-01-02", "2026-05-01")

	client := h.twilio.GetRestClient(accountSID, authToken)
	params := &openapi.ListMessageParams{}
	params.SetDateSentAfter(startDate)
	params.SetDateSentBefore(time.Now())
	params.SetLimit(100000)
	params.SetPageSize(1000) // max per page to reduce API calls

	log.Printf("[Recovery Job] Fetching messages from Twilio API (from %s to now)...", startDate.Format("2006-01-02"))
	rawMessages, err := client.Api.ListMessage(params)
	if err != nil {
		log.Printf("[Recovery Job] ERROR fetching messages: %v", err)
		return
	}

	log.Printf("[Recovery Job] Fetched %d raw messages from Twilio", len(rawMessages))

	// Parse and clean
	messages := parseTwilioMessages(rawMessages)
	log.Printf("[Recovery Job] Parsed %d valid messages", len(messages))

	// Group messages by customer phone number
	conversations := make(map[string][]twilioMessage)

	for _, msg := range messages {
		var customerPhone string
		if msg.From == twilioNumber {
			customerPhone = msg.To
		} else if msg.To == twilioNumber {
			customerPhone = msg.From
		} else {
			continue
		}
		conversations[customerPhone] = append(conversations[customerPhone], msg)
	}

	log.Printf("[Recovery Job] Found %d unique phone numbers", len(conversations))

	recoveredCount := 0
	skippedExisting := 0
	skippedNotAI := 0
	failedCount := 0

	for customerPhone, msgs := range conversations {
		// Sort messages by DateSent ascending
		sort.Slice(msgs, func(i, j int) bool {
			return msgs[i].DateSent.Before(msgs[j].DateSent)
		})

		if len(msgs) == 0 {
			continue
		}

		// --- FILTER: Check if this is a valid AI chat ---
		firstMsg := msgs[0]

		// Bot must have sent the first message
		if firstMsg.From != twilioNumber {
			log.Printf("[Recovery Job] SKIP %s: user wrote first (not AI-triggered). Message: %s", customerPhone, firstMsg.Body)
			skippedNotAI++
			continue
		}

		// First message must match the trigger pattern
		if !strings.Contains(firstMsg.Body, "this is Ally from Aveda!") {
			log.Printf("[Recovery Job] SKIP %s: first message doesn't match AI trigger pattern. Message: %s", customerPhone, firstMsg.Body)
			skippedNotAI++
			continue
		}

		// --- FILTER: Skip if chat already exists in DB ---
		existingChat, err := h.db.GetLatestChatByCustomer(assistantID, customerPhone)
		if err == nil && existingChat != nil && existingChat.Id != "" {
			log.Printf("[Recovery Job] SKIP %s: chat already exists in DB (id=%s)", customerPhone, existingChat.Id)
			skippedExisting++
			continue
		}

		log.Printf("[Recovery Job] ---- Recovering chat for %s (%d messages) ----", customerPhone, len(msgs))

		// Create Chat with the time of the first message
		chatResp, err := h.db.CreateChatWithTime(assistantID, customerPhone, "sms", firstMsg.DateSent.Format(time.RFC3339))
		if err != nil {
			log.Printf("[Recovery Job] FAIL creating chat for %s: %v", customerPhone, err)
			failedCount++
			continue
		}

		chatID := chatResp.Id
		log.Printf("[Recovery Job] Created chat %s for %s", chatID, customerPhone)

		// Save all messages and build transcript
		var transcriptBuilder strings.Builder

		for i, msg := range msgs {
			role := "user"
			if msg.From == twilioNumber {
				role = "assistant"
			}

			_, err := h.db.SaveMessageWithTime(chatID, role, msg.Body, "sms", msg.DateSent.Format(time.RFC3339))
			if err != nil {
				log.Printf("[Recovery Job] WARN: failed to save message %d for %s: %v", i, customerPhone, err)
			}

			transcriptBuilder.WriteString(fmt.Sprintf("[%s] %s: %s\n", msg.DateSent.Format("2006-01-02 15:04"), role, msg.Body))

			// Small sleep every 50 messages to not overload DB
			if i > 0 && i%10 == 0 {
				time.Sleep(100 * time.Millisecond)
			}
		}

		log.Printf("[Recovery Job] Saved %d messages for chat %s", len(msgs), chatID)

		// --- LLM Analysis (to restore campuslogin flags) ---
		// Sleep before LLM call to prevent rate limiting and server overload
		time.Sleep(3 * time.Second)

		transcript := transcriptBuilder.String()
		log.Printf("[Recovery Job] Running LLM analysis for %s (transcript length: %d chars)...", customerPhone, len(transcript))

		err = h.LLM.AnalyzeTranscript(context.Background(), customerPhone, assistantID, transcript)
		if err != nil {
			log.Printf("[Recovery Job] WARN: LLM analysis failed for %s: %v", customerPhone, err)
		} else {
			log.Printf("[Recovery Job] LLM analysis completed for %s", customerPhone)
		}

		recoveredCount++
		log.Printf("[Recovery Job] ---- Done with %s (recovered: %d so far) ----", customerPhone, recoveredCount)
	}

	log.Printf("[Recovery Job] ========================================")
	log.Printf("[Recovery Job] FINISHED")
	log.Printf("[Recovery Job] Recovered: %d chats", recoveredCount)
	log.Printf("[Recovery Job] Skipped (already exist): %d", skippedExisting)
	log.Printf("[Recovery Job] Skipped (not AI chat): %d", skippedNotAI)
	log.Printf("[Recovery Job] Failed: %d", failedCount)
	log.Printf("[Recovery Job] ========================================")
}

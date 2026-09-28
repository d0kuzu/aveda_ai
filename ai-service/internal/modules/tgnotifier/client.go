package tgnotifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Client is a lightweight Telegram Bot API wrapper for sending
// notifications to a specific chat/channel.
type Client struct {
	botToken string
	chatID   string
	client   *http.Client
}

type sendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type telegramAPIResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// New creates a new tgnotifier Client.
// botToken — Telegram Bot API token.
// chatID   — target chat/channel ID (can be numeric ID or @channel_username).
func New(botToken, chatID string) *Client {
	return &Client{
		botToken: botToken,
		chatID:   chatID,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// SendMessage sends a text message to the configured chat.
// Long messages are automatically split into 4096-character chunks.
func (c *Client) SendMessage(text string) error {
	return c.SendMessageWithParseMode(text, "")
}

// SendHTML sends a message with HTML parse mode.
func (c *Client) SendHTML(text string) error {
	return c.SendMessageWithParseMode(text, "HTML")
}

// SendMessageWithParseMode sends a message with a specified parse mode.
// If parseMode is empty, Telegram uses default (no formatting).
func (c *Client) SendMessageWithParseMode(text, parseMode string) error {
	const limit = 4096
	runes := []rune(text)

	for i := 0; i < len(runes); i += limit {
		end := i + limit
		if end > len(runes) {
			end = len(runes)
		}

		err := c.sendChunk(string(runes[i:end]), parseMode)
		if err != nil {
			return fmt.Errorf("failed to send chunk starting at %d: %w", i, err)
		}

		// Small delay between chunks to avoid rate limits
		if end < len(runes) {
			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}

func (c *Client) sendChunk(text, parseMode string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.botToken)

	reqBody := sendMessageRequest{
		ChatID:    c.chatID,
		Text:      text,
		ParseMode: parseMode,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			if attempt == maxRetries-1 {
				return fmt.Errorf("failed to send request after %d attempts: %w", maxRetries, err)
			}
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return nil
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := 1
			if retryHeader := resp.Header.Get("Retry-After"); retryHeader != "" {
				fmt.Sscanf(retryHeader, "%d", &retryAfter)
			}
			log.Printf("[TGNotifier] Rate limited (429). Retrying after %d seconds...", retryAfter)
			time.Sleep(time.Duration(retryAfter) * time.Second)
			continue
		}

		var apiResp telegramAPIResponse
		json.Unmarshal(body, &apiResp)

		return fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, apiResp.Description)
	}

	return fmt.Errorf("failed to send message after %d attempts", maxRetries)
}

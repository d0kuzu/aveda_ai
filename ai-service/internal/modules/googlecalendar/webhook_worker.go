package googlecalendar

import (
	"context"
	"log"
	"time"

	"diaxel/internal/grpc/db"
	"github.com/google/uuid"
)

type WebhookWorker struct {
	gcClient    *Client
	dbClient    *db.Client
	calendarID  string
	webhookHost string // e.g. "https://api.my-domain.com/google/webhook"
}

func NewWebhookWorker(gcClient *Client, dbClient *db.Client, calendarID, webhookHost string) *WebhookWorker {
	if calendarID == "" {
		calendarID = "primary"
	}
	return &WebhookWorker{
		gcClient:    gcClient,
		dbClient:    dbClient,
		calendarID:  calendarID,
		webhookHost: webhookHost,
	}
}

func (w *WebhookWorker) Start(ctx context.Context) {
	// First check at startup
	w.checkAndRefreshWebhook()

	// Then check every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	for {
		select {
		case <-ticker.C:
			w.checkAndRefreshWebhook()
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (w *WebhookWorker) checkAndRefreshWebhook() {
	log.Printf("[WebhookWorker] checking Google Calendar webhook expiration for calendar: %s", w.calendarID)
	
	syncData, err := w.dbClient.GetGoogleSyncToken(w.calendarID)
	if err != nil {
		log.Printf("[WebhookWorker] token not found or error: %v. Will create new webhook.", err)
	}

	needsRefresh := true
	var syncToken string

	if syncData != nil {
		syncToken = syncData.SyncToken
		if syncData.ExpiresAt != "" {
			expiresAt, err := time.Parse(time.RFC3339, syncData.ExpiresAt)
			if err == nil {
				timeLeft := time.Until(expiresAt)
				// Обновляем, если до истечения вебхука осталось меньше или равно 24 часа
				if timeLeft > 24*time.Hour {
					needsRefresh = false
					log.Printf("[WebhookWorker] Webhook is valid until %s, no refresh needed.", expiresAt.Format(time.RFC3339))
				} else {
					log.Printf("[WebhookWorker] Webhook expires soon (in %v), refreshing...", timeLeft)
				}
			}
		}

		if needsRefresh && syncData.ChannelId != "" && syncData.ResourceId != "" {
			// Try to stop the old watch
			log.Printf("[WebhookWorker] Stopping old webhook (channel: %s, resource: %s)", syncData.ChannelId, syncData.ResourceId)
			if err := w.gcClient.StopWatch(syncData.ChannelId, syncData.ResourceId); err != nil {
				log.Printf("[WebhookWorker] Error stopping old webhook (might be already expired): %v", err)
			}
		}
	}

	if needsRefresh {
		newChannelID := uuid.New().String()
		// Google max is 30 days. We set 7 days to be safe.
		newExpiration := time.Now().Add(7 * 24 * time.Hour)

		log.Printf("[WebhookWorker] Creating new webhook for calendar %s (channel: %s)", w.calendarID, newChannelID)
		channel, err := w.gcClient.WatchEvents(w.calendarID, newChannelID, w.webhookHost, newExpiration)
		if err != nil {
			log.Printf("[WebhookWorker] Error creating webhook: %v", err)
			return
		}

		// Save to DB
		log.Printf("[WebhookWorker] Saving new webhook to DB (channel: %s, resource: %s, expires: %s)", channel.Id, channel.ResourceId, newExpiration.Format(time.RFC3339))
		_, err = w.dbClient.UpsertGoogleSyncToken(w.calendarID, syncToken, channel.Id, channel.ResourceId, newExpiration.Format(time.RFC3339))
		if err != nil {
			log.Printf("[WebhookWorker] Error saving new webhook to DB: %v", err)
		}
	}
}

package telegram

import (
	"context"
	"diaxel/internal/modules/llm"
	"log"
	"sync"
	"time"
)

type TelegramTask struct {
	BotToken    string
	ChatID      int64
	UserID      string
	AssistantID string
	UserMessage string
	UpdateID    int64
}

type Orchestrator struct {
	llm     *llm.Client
	sender  *Sender
	queue   chan TelegramTask
	seen    sync.Map
	workers int
}

func NewOrchestrator(llmClient *llm.Client, workers int, queueSize int) *Orchestrator {
	if workers <= 0 {
		workers = 5
	}
	if queueSize <= 0 {
		queueSize = 1000
	}

	return &Orchestrator{
		llm:     llmClient,
		sender:  NewSender(),
		queue:   make(chan TelegramTask, queueSize),
		workers: workers,
	}
}

func (o *Orchestrator) Start(ctx context.Context) {
	log.Printf("[Telegram Orchestrator] Starting %d workers...", o.workers)
	for i := 0; i < o.workers; i++ {
		go o.worker(ctx, i)
	}

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				o.seen.Range(func(key, value interface{}) bool {
					if seenTime, ok := value.(time.Time); ok {
						if now.Sub(seenTime) > 24*time.Hour {
							o.seen.Delete(key)
						}
					}
					return true
				})
			}
		}
	}()
}

func (o *Orchestrator) worker(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("[Telegram Worker %d] Stopping...", id)
			return
		case task := <-o.queue:
			o.processTask(ctx, id, task)
		}
	}
}

func (o *Orchestrator) processTask(ctx context.Context, workerID int, task TelegramTask) {
	aiCtx, aiCancel := context.WithTimeout(ctx, 2*time.Minute)
	defer aiCancel()

	log.Printf("[Telegram Worker %d] Processing update %d for user %s", workerID, task.UpdateID, task.UserID)
	aiResponse, err := o.llm.Conversation(aiCtx, task.UserID, task.AssistantID, task.UserMessage)
	if err != nil {
		log.Printf("[Telegram Worker %d] AI Error for update %d: %v\n", workerID, task.UpdateID, err)
		o.sender.Send(task.BotToken, task.ChatID, "Извините, произошла техническая ошибка при обработке вашего запроса.")
		return
	}

	err = o.sender.Send(task.BotToken, task.ChatID, aiResponse)
	if err != nil {
		log.Printf("[Telegram Worker %d] Failed to send answer to Telegram: %v\n", workerID, err)
	} else {
		log.Printf("[Telegram Worker %d] Sent answer to Telegram for update %d", workerID, task.UpdateID)
	}
}

func (o *Orchestrator) Enqueue(task TelegramTask) bool {
	// Telegram assigns update IDs independently to each bot.
	key := struct {
		assistantID string
		updateID    int64
	}{task.AssistantID, task.UpdateID}
	if _, loaded := o.seen.LoadOrStore(key, time.Now()); loaded {
		return true
	}

	select {
	case o.queue <- task:
		return true
	default:
		o.seen.Delete(key)
		return false
	}
}

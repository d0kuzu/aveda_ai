package dailyreport

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/tgnotifier"
)

// Worker handles the daily report job.
// It sends a daily analytics report to a Telegram channel at the configured time.
type Worker struct {
	db       *db.Client
	notifier *tgnotifier.Client
	sendTime string // HH:MM format in Asia/Almaty timezone
}

// NewWorker creates a new daily report worker.
// sendTime should be in "HH:MM" format (e.g. "21:00").
func NewWorker(dbClient *db.Client, notifier *tgnotifier.Client, sendTime string) *Worker {
	if sendTime == "" {
		sendTime = "21:00"
	}

	return &Worker{
		db:       dbClient,
		notifier: notifier,
		sendTime: sendTime,
	}
}

// Start begins the daily report loop. It calculates the next send time
// in Asia/Almaty timezone and sleeps until then.
func (w *Worker) Start(ctx context.Context) {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		log.Printf("[DailyReport] Error loading Asia/Almaty timezone: %v", err)
		return
	}

	log.Printf("[DailyReport] Worker started. Report time: %s (Asia/Almaty)", w.sendTime)

	for {
		now := time.Now().In(loc)
		nextSend := w.nextSendTime(now, loc)
		duration := nextSend.Sub(now)

		log.Printf("[DailyReport] Next report at: %s (in %s)", nextSend.Format("2006-01-02 15:04:05 MST"), duration.Round(time.Second))

		select {
		case <-ctx.Done():
			log.Println("[DailyReport] Worker stopped.")
			return
		case <-time.After(duration):
			w.sendReport(ctx, loc)
		}
	}
}

// nextSendTime calculates the next execution time based on the configured HH:MM.
func (w *Worker) nextSendTime(now time.Time, loc *time.Location) time.Time {
	var hour, minute int
	fmt.Sscanf(w.sendTime, "%d:%d", &hour, &minute)

	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
	if now.After(next) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// sendReport collects analytics and sends the Telegram message.
func (w *Worker) sendReport(ctx context.Context, loc *time.Location) {
	log.Println("[DailyReport] Generating daily report...")

	now := time.Now().In(loc)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24 * time.Hour)

	startStr := startOfDay.UTC().Format(time.RFC3339)
	endStr := endOfDay.UTC().Format(time.RFC3339)

	// Collect metrics for all assistants (empty assistant_id = global)
	metrics, err := w.db.GetPeriodMetrics("", startStr, endStr)

	var metricsBlock string
	if err != nil {
		log.Printf("[DailyReport] Error getting period metrics: %v", err)
		metricsBlock = "⚠️ Не удалось получить метрики за период"
	} else {
		metricsBlock = fmt.Sprintf(
			"📊 <b>Чаты за сегодня:</b>\n"+
				"  • Начато: <b>%d</b>\n"+
				"  • Завершено: <b>%d</b>\n"+
				"  • Забронировано: <b>%d</b>",
			metrics.StartedChats,
			metrics.CompletedChats,
			metrics.BookedChats,
		)
	}

	// System health info
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	uptimeInfo := fmt.Sprintf(
		"🖥 <b>Состояние сервера:</b>\n"+
			"  • Горутины: <b>%d</b>\n"+
			"  • Память (Alloc): <b>%.1f MB</b>\n"+
			"  • Память (Sys): <b>%.1f MB</b>\n"+
			"  • GC циклов: <b>%d</b>",
		runtime.NumGoroutine(),
		float64(memStats.Alloc)/1024/1024,
		float64(memStats.Sys)/1024/1024,
		memStats.NumGC,
	)

	// Build the full report message
	hostname, _ := os.Hostname()
	report := fmt.Sprintf(
		"📋 <b>Ежедневный отчёт</b>\n"+
			"📅 %s\n"+
			"🕐 %s (Астана)\n\n"+
			"━━━━━━━━━━━━━━━━━━\n\n"+
			"%s\n\n"+
			"━━━━━━━━━━━━━━━━━━\n\n"+
			"%s\n\n"+
			"━━━━━━━━━━━━━━━━━━\n\n"+
			"✅ Сервис <b>ai-service</b> работает\n"+
			"🏠 Хост: <b>%s</b>\n"+
			"🔄 Go: <b>%s</b>",
		now.Format("02.01.2006"),
		now.Format("15:04"),
		metricsBlock,
		uptimeInfo,
		hostname,
		runtime.Version(),
	)

	err = w.notifier.SendHTML(report)
	if err != nil {
		log.Printf("[DailyReport] Failed to send report to Telegram: %v", err)
	} else {
		log.Println("[DailyReport] Report sent successfully!")
	}
}

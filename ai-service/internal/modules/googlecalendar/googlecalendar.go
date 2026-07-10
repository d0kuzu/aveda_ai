package googlecalendar

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// NotificationEmails — список email-адресов, которым отправляются уведомления при создании ивентов.
var NotificationEmails = []string{
	"cali@avedainstitutewinnipeg.ca",
	"dkobdabaev@mail.ru",
}

type Client struct {
	srv *calendar.Service
}

// NewClient создает нового клиента Google Calendar, используя файлы credentials и token.
func NewClient(credentialsFile, tokenFile string) (*Client, error) {
	ctx := context.Background()

	// Читаем файл credentials.json
	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла credentials: %w", err)
	}

	// Парсим конфигурацию (нужен скоуп CalendarEvents для чтения/записи ивентов, или CalendarScope для полного доступа)
	config, err := google.ConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга файла credentials: %w", err)
	}

	// Читаем файл token.json
	tokBytes, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла token: %w", err)
	}

	var tok oauth2.Token
	if err := json.Unmarshal(tokBytes, &tok); err != nil {
		return nil, fmt.Errorf("ошибка парсинга файла token: %w", err)
	}

	// Создаем http клиент с токеном
	httpClient := config.Client(ctx, &tok)

	// Инициализируем сервис календаря
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания сервиса Calendar: %w", err)
	}

	return &Client{srv: srv}, nil
}

// CreateEvent создает новую запись (ивент) в календаре.
// calendarID - обычно "primary" для основного календаря пользователя.
func (c *Client) CreateEvent(calendarID string, event *calendar.Event, customerEmail string) (*calendar.Event, error) {
	if calendarID == "" {
		calendarID = "primary"
	}

	// Добавляем attendees для рассылки уведомлений
	for _, email := range NotificationEmails {
		event.Attendees = append(event.Attendees, &calendar.EventAttendee{
			Email: email,
		})
	}
	// if customerEmail != "" {
	// 	event.Attendees = append(event.Attendees, &calendar.EventAttendee{
	// 		Email: customerEmail,
	// 	})
	// }

	// Выполняем запрос на добавление ивента с отправкой уведомлений участникам
	createdEvent, err := c.srv.Events.Insert(calendarID, event).SendUpdates("all").Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка создания ивента: %w", err)
	}

	return createdEvent, nil
}

// GetFreeBusy возвращает информацию о занятости (свободных слотах) в заданном промежутке времени.
// calendarID - ID календаря (обычно "primary" или email пользователя).
func (c *Client) GetFreeBusy(calendarID string, timeMin, timeMax time.Time) (*calendar.FreeBusyResponse, error) {
	if calendarID == "" {
		calendarID = "primary"
	}

	req := &calendar.FreeBusyRequest{
		TimeMin: timeMin.Format(time.RFC3339),
		TimeMax: timeMax.Format(time.RFC3339),
		Items: []*calendar.FreeBusyRequestItem{
			{Id: calendarID},
		},
	}

	// Запрашиваем информацию о занятости
	freeBusyResponse, err := c.srv.Freebusy.Query(req).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса freebusy: %w", err)
	}

	return freeBusyResponse, nil
}

// CreateSimpleEvent - упрощенная обертка для создания записи (в основном календаре).
func (c *Client) CreateSimpleEvent(title string, start, end time.Time, customerEmail string) (*calendar.Event, error) {
	event := &calendar.Event{
		Summary: title,
		Start: &calendar.EventDateTime{
			DateTime: start.Format(time.RFC3339),
			TimeZone: start.Location().String(),
		},
		End: &calendar.EventDateTime{
			DateTime: end.Format(time.RFC3339),
			TimeZone: end.Location().String(),
		},
	}
	return c.CreateEvent("", event, customerEmail)
}
// ListEvents выполняет инкрементальную синхронизацию событий календаря.
// Если syncToken пустой — выполняет полную синхронизацию (возвращает все события).
// Возвращает список событий и новый syncToken для следующего вызова.
func (c *Client) ListEvents(calendarID, syncToken string) ([]*calendar.Event, string, error) {
	if calendarID == "" {
		calendarID = "primary"
	}

	call := c.srv.Events.List(calendarID).SingleEvents(true)

	if syncToken != "" {
		call = call.SyncToken(syncToken)
	} else {
		// Получаем события только на 8 дней вперед, чтобы избежать огромных выгрузок
		now := time.Now()
		call = call.TimeMin(now.Format(time.RFC3339))
		call = call.TimeMax(now.AddDate(0, 0, 8).Format(time.RFC3339))
	}

	var allEvents []*calendar.Event
	var nextSyncToken string

	for {
		events, err := call.Do()
		if err != nil {
			return nil, "", fmt.Errorf("ошибка получения списка событий: %w", err)
		}

		allEvents = append(allEvents, events.Items...)

		if events.NextPageToken != "" {
			call = call.PageToken(events.NextPageToken)
		} else {
			nextSyncToken = events.NextSyncToken
			break
		}
	}

	return allEvents, nextSyncToken, nil
}

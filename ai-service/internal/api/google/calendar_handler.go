package google

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"diaxel/internal/config"
	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/campuslogin"
	"diaxel/internal/modules/googlecalendar"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/calendar/v3"
)

const calendarID = "primary"

// workWindow описывает рабочие часы для одного дня недели.
type workWindow struct {
	StartHour   int
	StartMinute int
	EndHour     int
	EndMinute   int
}

// slotInfo описывает один временной слот для ответа API.
type slotInfo struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Booked    int32  `json:"booked"`
	Available int32  `json:"available"`
}

// bookRequest — тело запроса на бронирование.
type bookRequest struct {
	StartTime   string `json:"start_time" binding:"required"`
	EndTime     string `json:"end_time" binding:"required"`
	Title       string `json:"title"`
	GuestName   string `json:"guest_name"`
	GuestEmail  string `json:"guest_email"`
	Description string `json:"description"`
}

type CalendarHandler struct {
	gc  *googlecalendar.Client
	db  *db.Client
	cl  *campuslogin.Client
	cfg *config.Settings

	// Парсированное расписание: day of week -> workWindow
	schedule map[time.Weekday]workWindow
	loc      *time.Location
}

func NewCalendarHandler(gc *googlecalendar.Client, db *db.Client, cl *campuslogin.Client, cfg *config.Settings) *CalendarHandler {
	h := &CalendarHandler{
		gc:  gc,
		db:  db,
		cl:  cl,
		cfg: cfg,
	}

	loc, err := time.LoadLocation("America/Winnipeg")
	if err != nil {
		log.Printf("[CalendarHandler] failed to load timezone America/Winnipeg, using UTC: %v", err)
		loc = time.UTC
	}
	h.loc = loc

	h.schedule = parseWorkSchedule(cfg.WorkSchedule)

	return h
}

// parseWorkSchedule парсит строку расписания формата "Tue=09:00-17:00,Wed=11:00-19:00,..."
func parseWorkSchedule(scheduleStr string) map[time.Weekday]workWindow {
	dayMap := map[string]time.Weekday{
		"Sun": time.Sunday,
		"Mon": time.Monday,
		"Tue": time.Tuesday,
		"Wed": time.Wednesday,
		"Thu": time.Thursday,
		"Fri": time.Friday,
		"Sat": time.Saturday,
	}

	result := make(map[time.Weekday]workWindow)

	parts := strings.Split(scheduleStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Tue=09:00-17:00
		eqIdx := strings.Index(part, "=")
		if eqIdx < 0 {
			log.Printf("[CalendarHandler] invalid schedule part (no '='): %s", part)
			continue
		}

		dayStr := strings.TrimSpace(part[:eqIdx])
		timeRange := strings.TrimSpace(part[eqIdx+1:])

		weekday, ok := dayMap[dayStr]
		if !ok {
			log.Printf("[CalendarHandler] unknown day: %s", dayStr)
			continue
		}

		dashIdx := strings.Index(timeRange, "-")
		if dashIdx < 0 {
			log.Printf("[CalendarHandler] invalid time range (no '-'): %s", timeRange)
			continue
		}

		startStr := strings.TrimSpace(timeRange[:dashIdx])
		endStr := strings.TrimSpace(timeRange[dashIdx+1:])

		startH, startM, err1 := parseHHMM(startStr)
		endH, endM, err2 := parseHHMM(endStr)
		if err1 != nil || err2 != nil {
			log.Printf("[CalendarHandler] invalid time format in schedule: %s", part)
			continue
		}

		result[weekday] = workWindow{
			StartHour:   startH,
			StartMinute: startM,
			EndHour:     endH,
			EndMinute:   endM,
		}
	}

	return result
}

func parseHHMM(s string) (int, int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, 0, err
	}
	return t.Hour(), t.Minute(), nil
}

// GetSlots возвращает свободные слоты на указанный день.
// GET /google/calendar/slots?date=2026-08-15
func (h *CalendarHandler) GetSlots(c *gin.Context) {
	dateStr := c.Query("date")
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'date' is required (format: 2006-01-02)"})
		return
	}

	date, err := time.ParseInLocation("2006-01-02", dateStr, h.loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, expected 2006-01-02"})
		return
	}

	// Проверяем есть ли расписание для этого дня недели
	ww, ok := h.schedule[date.Weekday()]
	if !ok {
		// Нерабочий день
		c.JSON(http.StatusOK, gin.H{
			"date":  dateStr,
			"slots": []slotInfo{},
		})
		return
	}

	// Генерируем слоты
	slotDuration := time.Duration(h.cfg.SlotDurationMinutes) * time.Minute
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), ww.StartHour, ww.StartMinute, 0, 0, h.loc)
	dayEnd := time.Date(date.Year(), date.Month(), date.Day(), ww.EndHour, ww.EndMinute, 0, 0, h.loc)

	// Получаем busy-периоды через FreeBusy (только opaque ивенты видны)
	freeBusyResp, err := h.gc.GetFreeBusy(calendarID, dayStart, dayEnd)
	if err != nil {
		log.Printf("[CalendarHandler] GetFreeBusy error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get calendar availability"})
		return
	}

	// Собираем busy-периоды
	var busyPeriods []struct{ Start, End time.Time }
	if calInfo, ok := freeBusyResp.Calendars[calendarID]; ok {
		for _, busy := range calInfo.Busy {
			busyStart, _ := time.Parse(time.RFC3339, busy.Start)
			busyEnd, _ := time.Parse(time.RFC3339, busy.End)
			busyPeriods = append(busyPeriods, struct{ Start, End time.Time }{busyStart, busyEnd})
		}
	}

	// Генерируем слоты и проверяем доступность
	var slots []slotInfo
	for slotStart := dayStart; slotStart.Add(slotDuration).Before(dayEnd) || slotStart.Add(slotDuration).Equal(dayEnd); slotStart = slotStart.Add(slotDuration) {
		slotEnd := slotStart.Add(slotDuration)

		// Проверяем пересечение с busy-периодами (opaque ивенты)
		isBusy := false
		for _, bp := range busyPeriods {
			if slotStart.Before(bp.End) && slotEnd.After(bp.Start) {
				isBusy = true
				break
			}
		}

		if isBusy {
			// Слот полностью заблокирован opaque ивентом — пропускаем
			continue
		}

		// Считаем количество записей на этот слот из БД
		booked, err := h.db.CountAppointmentsBySlot(calendarID, slotStart.Format(time.RFC3339))
		if err != nil {
			log.Printf("[CalendarHandler] CountAppointmentsBySlot error for slot %s: %v", slotStart.Format(time.RFC3339), err)
			booked = 0
		}

		available := int32(h.cfg.MaxSlotCapacity) - booked
		if available < 0 {
			available = 0
		}

		slots = append(slots, slotInfo{
			StartTime: slotStart.Format(time.RFC3339),
			EndTime:   slotEnd.Format(time.RFC3339),
			Booked:    booked,
			Available: available,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"date":  dateStr,
		"slots": slots,
	})
}

// BookSlot создает бронирование на указанный слот.
// POST /google/calendar/book
func (h *CalendarHandler) BookSlot(c *gin.Context) {
	var req bookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}
	log.Printf("[CalendarHandler] BookSlot request: %v", req)

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time format, expected RFC3339"})
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time format, expected RFC3339"})
		return
	}

	// Проверяем количество записей на слоте
	booked, err := h.db.CountAppointmentsBySlot(calendarID, startTime.Format(time.RFC3339))
	if err != nil {
		log.Printf("[CalendarHandler] CountAppointmentsBySlot error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check slot availability"})
		return
	}

	maxCapacity := int32(h.cfg.MaxSlotCapacity)
	if booked >= maxCapacity {
		c.JSON(http.StatusConflict, gin.H{"error": "slot is full", "booked": booked, "max_capacity": maxCapacity})
		return
	}

	// Определяем transparency
	transparency := "transparent"
	if booked+1 >= maxCapacity {
		transparency = "opaque"
	}

	// Формируем title
	title := req.Title
	if title == "" {
		title = "Campus Tour"
		if req.GuestName != "" {
			title = "Campus Tour - " + req.GuestName
		}
	}

	// Создаём ивент в Google Calendar
	event := &calendar.Event{
		Summary:     title,
		Description: req.Description,
		Start: &calendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
			TimeZone: h.loc.String(),
		},
		End: &calendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
			TimeZone: h.loc.String(),
		},
	}

	createdEvent, err := h.gc.CreateEventWithTransparency(calendarID, event, transparency, req.GuestEmail)
	if err != nil {
		log.Printf("[CalendarHandler] CreateEventWithTransparency error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create calendar event"})
		return
	}

	// Попытка отправить запись в CampusLogin если есть телефон
	success := TrySendCampusLogin(c.Request.Context(), h.db, h.cl, event.Description+" "+event.Summary, startTime, endTime, event.Description)

	// Сохраняем запись в БД
	_, err = h.db.CreateAppointment(
		createdEvent.Id,
		title,
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339),
		"confirmed",
		req.Description,
		calendarID,
		success,
		"", // createdAt — будет time.Now() в сервере
	)

	if err != nil {
		log.Printf("[CalendarHandler] CreateAppointment error: %v", err)
		// Ивент уже создан в Calendar, но запись в БД не сохранена
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":           "event created in calendar but failed to save to database",
			"google_event_id": createdEvent.Id,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "booking successful",
		"google_event_id": createdEvent.Id,
		"transparency":    transparency,
		"booked":          booked + 1,
		"max_capacity":    maxCapacity,
	})
}

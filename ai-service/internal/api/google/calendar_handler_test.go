package google

import (
	"testing"
	"time"

	"diaxel/internal/config"
)

func TestCalendarHandler_GetWorkWindow(t *testing.T) {
	loc, err := time.LoadLocation("America/Winnipeg")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	cfg := &config.Settings{
		WorkSchedule: "Tue=09:00-17:00,Wed=11:00-19:00,Thu=11:00-19:00,Fri=09:00-17:00,Sat=09:00-17:00",
	}

	h := NewCalendarHandler(nil, nil, nil, cfg)

	tests := []struct {
		name      string
		dateStr   string
		expectOk  bool
		startHour int
		endHour   int
	}{
		// 2026 (Before Jan 1st, 2027)
		{
			name:     "2026 Sunday (off)",
			dateStr:  "2026-12-27",
			expectOk: false,
		},
		{
			name:     "2026 Monday (off)",
			dateStr:  "2026-12-28",
			expectOk: false,
		},
		{
			name:      "2026 Tuesday (9am-5pm)",
			dateStr:   "2026-12-29",
			expectOk:  true,
			startHour: 9,
			endHour:   17,
		},
		{
			name:      "2026 Wednesday (11am-7pm)",
			dateStr:   "2026-12-30",
			expectOk:  true,
			startHour: 11,
			endHour:   19,
		},
		{
			name:      "2026 Thursday (11am-7pm)",
			dateStr:   "2026-12-31",
			expectOk:  true,
			startHour: 11,
			endHour:   19,
		},

		// 2027 (Starting Jan 1st, 2027)
		{
			name:      "2027-01-01 Friday (9am-5pm)",
			dateStr:   "2027-01-01",
			expectOk:  true,
			startHour: 9,
			endHour:   17,
		},
		{
			name:      "2027-01-02 Saturday (9am-5pm)",
			dateStr:   "2027-01-02",
			expectOk:  true,
			startHour: 9,
			endHour:   17,
		},
		{
			name:     "2027-01-03 Sunday (off)",
			dateStr:  "2027-01-03",
			expectOk: false,
		},
		{
			name:     "2027-01-04 Monday (off)",
			dateStr:  "2027-01-04",
			expectOk: false,
		},
		{
			name:      "2027-01-05 Tuesday (9am-5pm)",
			dateStr:   "2027-01-05",
			expectOk:  true,
			startHour: 9,
			endHour:   17,
		},
		{
			name:      "2027-01-06 Wednesday (now 9am-5pm)",
			dateStr:   "2027-01-06",
			expectOk:  true,
			startHour: 9,
			endHour:   17,
		},
		{
			name:      "2027-01-07 Thursday (now 9am-5pm)",
			dateStr:   "2027-01-07",
			expectOk:  true,
			startHour: 9,
			endHour:   17,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			date, err := time.ParseInLocation("2006-01-02", tc.dateStr, loc)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			ww, ok := h.getWorkWindow(date)
			if ok != tc.expectOk {
				t.Fatalf("expected ok=%v, got %v", tc.expectOk, ok)
			}
			if tc.expectOk {
				if ww.StartHour != tc.startHour || ww.EndHour != tc.endHour {
					t.Fatalf("expected %02d:00-%02d:00, got %02d:00-%02d:00",
						tc.startHour, tc.endHour, ww.StartHour, ww.EndHour)
				}
			}
		})
	}
}

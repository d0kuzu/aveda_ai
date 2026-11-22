package database

import (
	"context"
	"fmt"
	"time"
	_ "time/tzdata"

	"github.com/jackc/pgx/v5"
)

func periodBounds(startTime, endTime string) (time.Time, time.Time, error) {
	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start time: %w", err)
	}
	end, err := time.Parse(time.RFC3339, endTime)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid end time: %w", err)
	}
	return start, end, nil
}

func (s *Store) GetPeriodMetrics(ctx context.Context, assistantID, startTime, endTime string) (*PeriodMetrics, error) {
	start, end, err := periodBounds(startTime, endTime)
	if err != nil {
		return nil, err
	}
	var result PeriodMetrics
	err = s.pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE is_end)
		FROM chats WHERE assistant_id = $1 AND started_at >= $2 AND started_at < $3`, assistantID, start, end).
		Scan(&result.StartedChats, &result.CompletedChats)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Store) GetPeriodChats(ctx context.Context, assistantID, startTime, endTime, category string, limit, offset int32) (*PeriodChats, error) {
	start, end, err := periodBounds(startTime, endTime)
	if err != nil {
		return nil, err
	}
	filter := " FROM chats c WHERE c.assistant_id = $1 AND c.started_at >= $2 AND c.started_at < $3"
	switch category {
	case "completed":
		filter += " AND c.is_end"
	case "started", "all", "":
	default:
		return nil, fmt.Errorf("unknown chat category %q", category)
	}
	var result PeriodChats
	opts := pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}
	err = pgx.BeginTxFunc(ctx, s.pool, opts, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, "SELECT count(*)"+filter, assistantID, start, end).Scan(&result.TotalCount); err != nil {
			return err
		}
		var err error
		result.Chats, err = collectChats(tx.Query(ctx, "SELECT "+chatFields+filter+`
			ORDER BY c.started_at DESC, c.id DESC LIMIT CASE WHEN $4::integer > 0 THEN $4 END OFFSET GREATEST($5::integer, 0)`,
			assistantID, start, end, limit, offset))
		return err
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Store) GetWeeklyChatsStarted(ctx context.Context, assistantID, startTime, timezone string) (*WeeklyChatsStarted, error) {
	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}
	start = start.In(loc)
	rows, err := s.pool.Query(ctx, `SELECT to_char(started_at AT TIME ZONE $4, 'YYYY-MM-DD'), count(*)
		FROM chats WHERE assistant_id = $1 AND started_at >= $2 AND started_at < $3 GROUP BY 1`,
		assistantID, start, start.AddDate(0, 0, 7), loc.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int32, 7)
	for rows.Next() {
		var date string
		var count int32
		if err := rows.Scan(&date, &count); err != nil {
			return nil, err
		}
		counts[date] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := &WeeklyChatsStarted{Days: make([]*DailyCount, 7)}
	for i := range result.Days {
		date := start.AddDate(0, 0, i).Format(time.DateOnly)
		result.Days[i] = &DailyCount{Date: date, Count: counts[date]}
	}
	return result, nil
}

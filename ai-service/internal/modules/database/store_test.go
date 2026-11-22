package database

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// TEST_DATABASE_URL points at a PostgreSQL database where the test can create a schema.
func TestConversationStorage(t *testing.T) {
	rawURL := os.Getenv("TEST_DATABASE_URL")
	if rawURL == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	admin, err := pgx.Connect(ctx, rawURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(ctx)
	schema := fmt.Sprintf("ai_store_test_%d", time.Now().UnixNano())
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	dsn, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	q := dsn.Query()
	q.Set("search_path", schema)
	dsn.RawQuery = q.Encode()

	type opened struct {
		store *Store
		err   error
	}
	ready := make(chan opened, 2)
	for i := 0; i < cap(ready); i++ {
		go func() {
			s, err := Open(ctx, dsn.String())
			ready <- opened{s, err}
		}()
	}
	var store *Store
	for i := 0; i < cap(ready); i++ {
		result := <-ready
		if result.err != nil {
			t.Error(result.err)
			continue
		}
		defer result.store.Close()
		store = result.store
	}
	if t.Failed() {
		t.FailNow()
	}
	const owner = "d7bb7217-9c70-4499-aac1-015836a31781"
	const other = "968ea97c-e8be-4fc5-823b-1ca7568c8b19"
	a, err := store.CreateAssistant(ctx, "one", "sample-key", owner, "", "api")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetAssistantForUser(ctx, a.Id, other); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other owner read assistant: %v", err)
	}
	if _, err := store.UpdateAssistantForUser(ctx, a.Id, other, "changed", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other owner updated assistant: %v", err)
	}
	if assistant, err := store.GetAssistantByAPIToken(ctx, "sample-key"); err != nil || assistant.Id != a.Id {
		t.Fatalf("lookup by token: %v, %v", assistant, err)
	}
	if _, err := store.GetAssistantByAPIToken(ctx, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty assistant token: %v", err)
	}
	const customer = "contact-1"
	if chat, err := store.GetLatestChatByCustomer(ctx, a.Id, customer); err != nil || chat != nil {
		t.Fatalf("first contact: %v, %v", chat, err)
	}

	const turns = 8
	writes := make(chan error, turns)
	for i := 0; i < turns; i++ {
		go func(i int) {
			writes <- store.SaveConversationTurn(ctx, a.Id, customer, fmt.Sprint(i), fmt.Sprint(i))
		}(i)
	}
	for i := 0; i < turns; i++ {
		if err := <-writes; err != nil {
			t.Error(err)
		}
	}
	chats, err := store.GetChatPageByUserID(ctx, []string{a.Id}, 1, 10)
	if err != nil || len(chats) != 1 || chats[0].MessageCount != 2*turns {
		t.Fatalf("concurrent first turns: %+v, %v", chats, err)
	}
	chatID := chats[0].Id
	if _, err := store.GetChatForUser(ctx, chatID, other); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other owner read chat: %v", err)
	}
	history, err := store.GetAllChatMessages(ctx, chatID)
	if err != nil || len(history) != 2*turns {
		t.Fatalf("history: %d messages, %v", len(history), err)
	}
	for i := 0; i < len(history); i += 2 {
		if history[i].Role != "user" || history[i+1].Role != "assistant" || history[i].Content != history[i+1].Content {
			t.Fatalf("messages out of order: %+v, %+v", history[i], history[i+1])
		}
	}
	if _, err := store.SaveMessage(ctx, chatID, "assistant", "manual", "api"); err != nil {
		t.Fatal(err)
	}
	found, n, err := store.SearchChatsByCustomer(ctx, []string{a.Id}, "contact")
	if err != nil || n != 1 || len(found) != 1 || found[0].MessageCount != 2*turns+1 {
		t.Fatalf("search after append: %v chats, %d, %v", len(found), n, err)
	}
	start := time.Now().UTC().Truncate(24 * time.Hour)
	from, to := timestamp(start), timestamp(start.AddDate(0, 0, 1))
	metrics, err := store.GetPeriodMetrics(ctx, a.Id, from, to)
	if err != nil || metrics.StartedChats != 1 || metrics.CompletedChats != 0 {
		t.Fatalf("metrics: %+v, %v", metrics, err)
	}
	period, err := store.GetPeriodChats(ctx, a.Id, from, to, "started", 10, 10)
	if err != nil || period.TotalCount != 1 || len(period.Chats) != 0 {
		t.Fatalf("empty page retains total: %+v, %v", period, err)
	}
	week, err := store.GetWeeklyChatsStarted(ctx, a.Id, from, "UTC")
	if err != nil || len(week.Days) != 7 || week.Days[0].Count != 1 || week.Days[6].Count != 0 {
		t.Fatalf("weekly chart: %+v, %v", week, err)
	}
}

package campuslogin

import (
	"context"
	"os"
	"testing"
)

func TestGetAvailability(t *testing.T) {
	apiKey := os.Getenv("CAMPUSLOGIN_CINTA_API")
	if apiKey == "" {
		apiKey = "urIm4hakIXNm8fny+JeEunJBMBjjW6E0BcRNawDBVz7KOIQsw63v8VoI/oTybppL/2qDjZdz/o3ZHgc6LL3sbA=="
	}
	staffID := 12001 // Anna Clenshaw

	client := NewClient(apiKey)
	resp, err := client.GetAvailability(context.Background(), GetAvailabilityParams{
		StaffID:  staffID,
		Duration: 60,
		FromDate: "2026-10-25",
		ToDate:   "2026-10-26",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("expected status 'success', got '%s'", resp.Status)
	}

	slots26 := resp.Data.Slots["2026-10-26"]
	if len(slots26) == 0 {
		t.Errorf("expected slots for 2026-10-26, got 0")
	}
}

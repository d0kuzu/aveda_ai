package calcom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	baseURL       = "https://api.cal.com/v2"
	calAPIVersion = "2024-09-04"
)

type Client struct {
	apiKey      string
	eventTypeID int
	httpClient  *http.Client
}

// Slot represents a single available time slot
type Slot struct {
	Time string `json:"time"`
}

// SlotsResponse represents the Cal.com /v2/slots response
type SlotsResponse struct {
	Status string                    `json:"status"`
	Data   map[string][]SlotsEntry   `json:"data"`
}

type SlotsEntry struct {
	Start string `json:"start"`
}

// BookingRequest is sent to POST /v2/bookings
type BookingRequest struct {
	Start       string          `json:"start"`
	EventTypeID int             `json:"eventTypeId"`
	Attendee    BookingAttendee `json:"attendee"`
}

type BookingAttendee struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	TimeZone string `json:"timeZone"`
}

// BookingResponse from Cal.com
type BookingResponse struct {
	Status string      `json:"status"`
	Data   BookingData `json:"data"`
}

type BookingData struct {
	ID     int    `json:"id"`
	UID    string `json:"uid"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Start  string `json:"start"`
	End    string `json:"end"`
}

func New(apiKey string, eventTypeID int) *Client {
	return &Client{
		apiKey:      apiKey,
		eventTypeID: eventTypeID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetAvailableSlots fetches available slots for a given date (YYYY-MM-DD format)
func (c *Client) GetAvailableSlots(ctx context.Context, date string, timezone string) ([]string, error) {
	// Build the time range for the full day
	startTime := date + "T00:00:00Z"
	endTime := date + "T23:59:59Z"

	url := fmt.Sprintf("%s/slots?eventTypeId=%d&start=%s&end=%s&timeZone=%s",
		baseURL, c.eventTypeID, startTime, endTime, timezone)

	log.Printf("[CalCom] GET slots: %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("cal-api-version", calAPIVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch slots: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("[CalCom] Slots response status: %d, body: %s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cal.com API returned status %d: %s", resp.StatusCode, string(body))
	}

	var slotsResp SlotsResponse
	if err := json.Unmarshal(body, &slotsResp); err != nil {
		return nil, fmt.Errorf("failed to parse slots response: %w", err)
	}

	// Extract all slot times from the response
	var availableSlots []string
	for dateKey, slots := range slotsResp.Data {
		// Filter only the requested date to be sure
		if !strings.HasPrefix(dateKey, date) {
			continue
		}

		for _, slot := range slots {
			// Cal.com returns: "2026-05-18T00:00:00.000-05:00"
			// We try to parse it to show a clean time to the user
			t, err := time.Parse(time.RFC3339, slot.Start)
			if err != nil {
				// Fallback to raw string if parsing fails
				availableSlots = append(availableSlots, slot.Start)
				continue
			}

			// Format: "14:30" (or whatever the local time is)
			availableSlots = append(availableSlots, fmt.Sprintf("%s (UTC: %s)", t.Format("15:04"), slot.Start))
		}
	}

	return availableSlots, nil
}

// CreateBooking creates a new booking via Cal.com API
func (c *Client) CreateBooking(ctx context.Context, startTime, attendeeName, attendeeEmail, timezone string) (*BookingData, error) {
	bookingReq := BookingRequest{
		Start:       startTime,
		EventTypeID: c.eventTypeID,
		Attendee: BookingAttendee{
			Name:     attendeeName,
			Email:    attendeeEmail,
			TimeZone: timezone,
		},
	}

	jsonBody, err := json.Marshal(bookingReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal booking request: %w", err)
	}

	url := baseURL + "/bookings"
	log.Printf("[CalCom] POST booking: %s, body: %s", url, string(jsonBody))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("cal-api-version", calAPIVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create booking: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("[CalCom] Booking response status: %d, body: %s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cal.com API returned status %d: %s", resp.StatusCode, string(body))
	}

	var bookingResp BookingResponse
	if err := json.Unmarshal(body, &bookingResp); err != nil {
		return nil, fmt.Errorf("failed to parse booking response: %w", err)
	}

	return &bookingResp.Data, nil
}

package campuslogin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// AvailabilitySlot представляет отдельный свободный временной слот
type AvailabilitySlot struct {
	Time string `json:"time"`
}

// AvailabilityData содержит мапу доступных слотов по датам (YYYY-MM-DD -> список слотов)
type AvailabilityData struct {
	Slots map[string][]AvailabilitySlot `json:"slots"`
}

// AvailabilityResponse представляет структуру ответа CampusLogin API Availability
type AvailabilityResponse struct {
	Status string           `json:"status"`
	Data   AvailabilityData `json:"data"`
}

// GetAvailabilityParams параметры для запроса свободных слотов
type GetAvailabilityParams struct {
	APIKey   string `json:"api_key"`
	OrgID    int    `json:"org_id"`
	StaffID  int    `json:"staff_id"`
	Duration int    `json:"duration"`
	FromDate string `json:"from_date"`
	ToDate   string `json:"to_date"`
}

// GetAvailability запрашивает свободные слоты у CampusLogin API
func (c *Client) GetAvailability(ctx context.Context, params GetAvailabilityParams) (*AvailabilityResponse, error) {
	apiKey := c.apiKey
	if params.APIKey != "" {
		apiKey = params.APIKey
	}
	if apiKey == "" {
		return nil, fmt.Errorf("campuslogin api key is required")
	}

	if params.StaffID == 0 {
		return nil, fmt.Errorf("staffID is required")
	}

	duration := params.Duration
	if duration <= 0 {
		duration = 60
	}

	baseURL := fmt.Sprintf("https://connectorapi.campuslogin.com/api/Appointments/Availability/%d", params.StaffID)
	q := url.Values{}
	q.Set("duration", strconv.Itoa(duration))
	if params.FromDate != "" {
		q.Set("fromDate", params.FromDate)
	}
	if params.ToDate != "" {
		q.Set("toDate", params.ToDate)
	}

	endpoint := baseURL + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Api-Key", apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var availability AvailabilityResponse
	if err := json.Unmarshal(bodyBytes, &availability); err != nil {
		return nil, fmt.Errorf("failed to unmarshal availability response: %w", err)
	}

	return &availability, nil
}

// GetAvailabilityWithKey вспомогательная функция для разового вызова с указанным ключом
func GetAvailabilityWithKey(ctx context.Context, apiKey string, staffID int, duration int, fromDate, toDate string) (*AvailabilityResponse, error) {
	client := NewClient(apiKey)
	return client.GetAvailability(ctx, GetAvailabilityParams{
		APIKey:   apiKey,
		StaffID:  staffID,
		Duration: duration,
		FromDate: fromDate,
		ToDate:   toDate,
	})
}

package campuslogin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type AppointmentRequest struct {
	OrgId                int    `json:"OrgId"`
	AppointmentID        int    `json:"AppointmentID"`
	SchoolID             int    `json:"SchoolID"`
	CampusID             int    `json:"CampusID"`
	MailListID           int    `json:"MailListID"`
	ContactID            int    `json:"ContactID"`
	DateFrom             string `json:"DateFrom"`
	DateTo               string `json:"DateTo"`
	Text                 string `json:"Text"`
	Description          string `json:"Description"`
	WebinarLink          string `json:"WebinarLink"`
	StaffID              int    `json:"StaffID"`
	EmployeeID           int    `json:"EmployeeID"`
	EmployeeIDs          string `json:"EmployeeIDs"`
	CreateDate           string `json:"CreateDate"`
	StageID              int    `json:"StageID"`
	SubStageID           int    `json:"SubStageID"`
	ShowUp               string `json:"ShowUp"`
	NoteOptions          string `json:"NoteOptions"`
	OutcomeDataID        int    `json:"OutcomeDataID"`
	ShowOnBookerCalendar int    `json:"ShowOnBookerCalendar"`
	Scheduled            int    `json:"Scheduled"`
	LocationCampusID     int    `json:"LocationCampusID"`
	LocationDetails      string `json:"LocationDetails"`
	InitAppointmentID    int    `json:"InitAppointmentID"`
	EventUrl             string `json:"EventUrl"`
	ProgramID            int    `json:"ProgramID"`
	ApplicationID        int    `json:"ApplicationID"`
	CampaignID           int    `json:"CampaignID"`
	ContactType          string `json:"ContactType"`
	TakenBy              int    `json:"TakenBy"`
	TakenOn              string `json:"TakenOn"`
	Office365GID         string `json:"Office365GID"`
	GoogleGID            string `json:"GoogleGID"`
	Automation           string `json:"Automation"`
}

func (c *Client) SendAppointment(ctx context.Context, startTime, endTime string, contactID int, programID int, description string) error {
	reqBody := AppointmentRequest{
		OrgId:                24900,
		AppointmentID:        0,
		SchoolID:             24901,
		CampusID:             12490101,
		MailListID:           24901001,
		ContactID:            contactID,
		DateFrom:             startTime,
		DateTo:               endTime,
		Text:                 "Appointment",
		Description:          description,
		WebinarLink:          "",
		StaffID:              6710,
		EmployeeID:           1,
		EmployeeIDs:          "",
		CreateDate:           "2026-05-19T00:00:00",
		StageID:              2638,
		SubStageID:           1,
		ShowUp:               "0",
		NoteOptions:          "",
		OutcomeDataID:        1,
		ShowOnBookerCalendar: 1,
		Scheduled:            1,
		LocationCampusID:     1,
		LocationDetails:      "",
		InitAppointmentID:    0,
		EventUrl:             "",
		ProgramID:            programID,
		ApplicationID:        1,
		CampaignID:           1,
		ContactType:          "",
		TakenBy:              1,
		TakenOn:              "2026-05-19T00:00:00",
		Office365GID:         "",
		GoogleGID:            "",
		Automation:           "",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://connectorapi.campuslogin.com/api/Appointments/ImportAppointment", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// Using the provided API key
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

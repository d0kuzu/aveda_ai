package campuslogin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"diaxel/internal/constants"
)

type AppointmentRequest struct {
	OrgId                int    `json:"orgId"`
	AppointmentID        int    `json:"appointmentID"`
	SchoolID             int    `json:"schoolID"`
	CampusID             int    `json:"campusID"`
	MailListID           int    `json:"mailListID"`
	ContactID            int    `json:"contactID"`
	DateFrom             string `json:"dateFrom"`
	DateTo               string `json:"dateTo"`
	Text                 string `json:"text"`
	Description          string `json:"description"`
	WebinarLink          string `json:"webinarLink"`
	StaffID              int    `json:"staffID"`
	EmployeeID           int    `json:"employeeID"`
	EmployeeIDs          string `json:"employeeIDs"`
	CreateDate           string `json:"createDate"`
	StageID              int    `json:"stageID"`
	SubStageID           int    `json:"subStageID"`
	ShowUp               int    `json:"showUp"`
	NoteOptions          string `json:"noteOptions"`
	OutcomeDataID        int    `json:"outcomeDataID"`
	ShowOnBookerCalendar int    `json:"showOnBookerCalendar"`
	Scheduled            int    `json:"scheduled"`
	LocationCampusID     int    `json:"locationCampusID"`
	LocationDetails      string `json:"locationDetails"`
	InitAppointmentID    int    `json:"initAppointmentID"`
	EventUrl             string `json:"eventUrl"`
	ProgramID            int    `json:"programID"`
	ApplicationID        int    `json:"applicationID"`
	CampaignID           int    `json:"campaignID"`
	ContactType          string `json:"contactType"`
	TakenBy              int    `json:"takenBy"`
	TakenOn              string `json:"takenOn"`
	Office365GID         string `json:"office365GID"`
	GoogleGID            string `json:"googleGID"`
	Automation           string `json:"automation"`
}

type ImportAppointmentResponse struct {
	OrgId                int    `json:"orgId"`
	APIAccountId         int    `json:"apiAccountId"`
	IsSuccessful         bool   `json:"isSuccessful"`
	ErrorMessage         string `json:"errorMessage"`
	AppointmentID        int    `json:"appointmentID"`
	SchoolID             int    `json:"schoolID"`
	CampusID             int    `json:"campusID"`
	MailListID           int    `json:"mailListID"`
	ContactID            int    `json:"contactID"`
	DateFrom             string `json:"dateFrom"`
	DateTo               string `json:"dateTo"`
	Text                 string `json:"text"`
	Description          string `json:"description"`
	WebinarLink          string `json:"webinarLink"`
	StaffID              int    `json:"staffID"`
	EmployeeID           int    `json:"employeeID"`
	EmployeeIDs          string `json:"employeeIDs"`
	CreateDate           string `json:"createDate"`
	StageID              int    `json:"stageID"`
	SubStageID           int    `json:"subStageID"`
	ShowUp               int    `json:"showUp"`
	NoteOptions          string `json:"noteOptions"`
	OutcomeDataID        int    `json:"outcomeDataID"`
	ShowOnBookerCalendar int    `json:"showOnBookerCalendar"`
	Scheduled            int    `json:"scheduled"`
	LocationCampusID     int    `json:"locationCampusID"`
	LocationDetails      string `json:"locationDetails"`
	InitAppointmentID    int    `json:"initAppointmentID"`
	EventUrl             string `json:"eventUrl"`
	ProgramID            int    `json:"programID"`
	ApplicationID        int    `json:"applicationID"`
	CampaignID           int    `json:"campaignID"`
	ContactType          string `json:"contactType"`
	TakenBy              int    `json:"takenBy"`
	TakenOn              string `json:"takenOn"`
	Office365GID         string `json:"office365GID"`
	GoogleGID            string `json:"googleGID"`
	Automation           string `json:"automation"`
}

// ImportAppointment отправляет запрос на импорт встречи напрямую в CampusLogin
func (c *Client) ImportAppointment(ctx context.Context, req AppointmentRequest) (*ImportAppointmentResponse, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://connectorapi.campuslogin.com/api/Appointments/ImportAppointment", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBytes))
	}

	var result ImportAppointmentResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w (body: %s)", err, string(respBytes))
	}

	if !result.IsSuccessful && result.ErrorMessage != "" {
		return &result, fmt.Errorf("campuslogin error: %s", result.ErrorMessage)
	}

	return &result, nil
}

func (c *Client) SendAppointment(ctx context.Context, orgID int, staffID int, title, startTime, endTime string, contactID int, programID int, description string) error {
	var schoolID, campusID, mailListID, stageID, subStageID, outcomeDataID int
	var loc *time.Location

	if orgID == constants.CampusLoginCintaOrgID {
		schoolID = constants.CampusLoginCintaSchoolID
		campusID = constants.CampusLoginCintaCampusIDSF
		mailListID = constants.CampusLoginCintaMailListID
		stageID = constants.CampusLoginCintaStageID
		subStageID = constants.CampusLoginCintaSubStageID
		outcomeDataID = constants.CampusLoginCintaTypeCPS
		var err error
		loc, err = time.LoadLocation("America/Los_Angeles")
		if err != nil {
			loc = time.FixedZone("PDT", -7*3600)
		}
	} else {
		schoolID = constants.CampusLoginCanadaSchoolID
		campusID = constants.CampusLoginCanadaCampusID
		mailListID = constants.CampusLoginCanadaMailListID
		stageID = constants.CampusLoginCanadaStageID
		subStageID = constants.CampusLoginCanadaSubStageID
		outcomeDataID = 1
		var err error
		loc, err = time.LoadLocation("America/Winnipeg")
		if err != nil {
			loc = time.UTC
		}
	}

	currentTime := time.Now().In(loc).Format("2006-01-02T15:04:05")

	reqBody := AppointmentRequest{
		OrgId:                orgID,
		AppointmentID:        0,
		SchoolID:             schoolID,
		CampusID:             campusID,
		MailListID:           mailListID,
		ContactID:            contactID,
		DateFrom:             startTime,
		DateTo:               endTime,
		Text:                 title,
		Description:          description,
		WebinarLink:          "",
		StaffID:              staffID,
		EmployeeID:           1,
		EmployeeIDs:          "",
		CreateDate:           currentTime,
		StageID:              stageID,
		SubStageID:           subStageID,
		ShowUp:               0,
		NoteOptions:          "",
		OutcomeDataID:        outcomeDataID,
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
		TakenOn:              currentTime,
		Office365GID:         "",
		GoogleGID:            "",
		Automation:           "",
	}

	_, err := c.ImportAppointment(ctx, reqBody)
	return err
}

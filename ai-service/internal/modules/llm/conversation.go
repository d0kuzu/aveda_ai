package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"diaxel/internal/constants"
	"github.com/sashabaranov/go-openai"
)

type ConversationOption func(*conversationOptions)

type conversationOptions struct {
	SystemMessage string
}

func WithSystemMessage(msg string) ConversationOption {
	return func(o *conversationOptions) {
		o.SystemMessage = msg
	}
}

func (c *Client) Conversation(ctx context.Context, userId, assistantId, userMessage string, opts ...ConversationOption) (string, error) {
	var optsConfig conversationOptions
	for _, opt := range opts {
		opt(&optsConfig)
	}

	if optsConfig.SystemMessage != "" {
		log.Printf("[LLM Trigger] Системный промпт для пользователя %s: %s", userId, optsConfig.SystemMessage)
	} else {
		log.Printf("Сообщение от пользователя %s: %s", userId, userMessage)
	}

	messages, err := c.GetMessages(assistantId, userId)
	if err != nil {
		return "", err
	}

	if optsConfig.SystemMessage != "" {
		c.AddMessage(&messages, openai.ChatMessageRoleSystem, optsConfig.SystemMessage)
	} else {
		c.AddMessage(&messages, openai.ChatMessageRoleUser, userMessage)
	}

	response, err := c.GetAnswer(ctx, messages)
	if err != nil {
		return "", err
	}

	for len(response.Choices) > 0 && len(response.Choices[0].Message.ToolCalls) > 0 {
		assistantMsg := response.Choices[0].Message
		if assistantMsg.Content == "" && len(assistantMsg.ToolCalls) > 0 {
			assistantMsg.Content = " "
		}
		messages = append(messages, assistantMsg)

		for _, toolCall := range assistantMsg.ToolCalls {
			log.Printf("Функция вызвана: %s", toolCall.Function.Name)
			log.Printf("Аргументы: %s", toolCall.Function.Arguments)

			result, err := c.executeFunction(ctx, toolCall.Function.Name, toolCall.Function.Arguments, userId, assistantId)
			if err != nil {
				log.Printf("Ошибка выполнения функции %s: %v", toolCall.Function.Name, err)
				result = fmt.Sprintf("Error executing function: %s", err.Error())
			}

			log.Printf("Результат функции %s: %s", toolCall.Function.Name, result)

			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    result,
				ToolCallID: toolCall.ID,
			})
		}

		response, err = c.GetAnswer(ctx, messages)
		if err != nil {
			return "", err
		}
	}

	assistantResponse := response.Choices[0].Message.Content
	log.Printf("Ответ пользователю %s от ИИ: %s\n", userId, assistantResponse)

	c.AddMessage(&messages, "assistant", assistantResponse)

	err = c.SaveMessages(assistantId, userId, messages)
	if err != nil {
		return "", err
	}

	log.Println("Конец запроса")
	return assistantResponse, nil
}

func (c *Client) executeFunction(ctx context.Context, functionName, argsJSON, userId, assistantId string) (string, error) {
	switch functionName {
	case "calcom_get_available_slots":
		return c.handleGetAvailableSlots(ctx, argsJSON)
	case "calcom_create_booking":
		return c.handleCreateBooking(ctx, argsJSON)
	case "get_available_slots":
		return c.handleCheckCampusAppointment(ctx, argsJSON)
	case "create_booking":
		return c.handleCreateCampusAppointment(ctx, argsJSON, userId)
	case "send_summary":
		return c.handleSendSummary(ctx, argsJSON, userId, assistantId)
	case "mark_grade_11_or_lower":
		return c.handleMarkGrade11OrLower(ctx, userId)
	case "mark_international_student":
		return c.handleMarkInternationalStudent(ctx, userId)
	default:
		return "", fmt.Errorf("unknown function: %s", functionName)
	}
}

func (c *Client) handleGetAvailableSlots(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		Date string `json:"date"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.Date == "" {
		return "Error: date parameter is required in YYYY-MM-DD format", nil
	}

	slots, err := c.calcom.GetAvailableSlots(ctx, args.Date, constants.DefaultTimezone)
	if err != nil {
		return "", fmt.Errorf("failed to get slots from Cal.com: %w", err)
	}

	if len(slots) == 0 {
		return fmt.Sprintf("No available slots found for %s. Please suggest another date.", args.Date), nil
	}

	return fmt.Sprintf("Available time slots for %s:\n%s", args.Date, strings.Join(slots, "\n")), nil
}

func (c *Client) handleCreateBooking(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		StartTime     string `json:"start_time"`
		AttendeeName  string `json:"attendee_name"`
		AttendeeEmail string `json:"attendee_email"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.StartTime == "" || args.AttendeeName == "" || args.AttendeeEmail == "" {
		return "Error: start_time, attendee_name, and attendee_email are all required", nil
	}

	booking, err := c.calcom.CreateBooking(ctx, args.StartTime, args.AttendeeName, args.AttendeeEmail, constants.DefaultTimezone)
	if err != nil {
		return "", fmt.Errorf("failed to create booking on Cal.com: %w", err)
	}

	return fmt.Sprintf("Booking confirmed! Details:\n- ID: %d\n- Title: %s\n- Status: %s\n- Start: %s\n- End: %s",
		booking.ID, booking.Title, booking.Status, booking.Start, booking.End), nil
}

func (c *Client) handleCheckCampusAppointment(ctx context.Context, argsJSON string) (string, error) {
	// Stub implementation as requested by user
	return "Available time slots: Any time from 6 AM to 8 PM is free.", nil
}

func (c *Client) handleCreateCampusAppointment(ctx context.Context, argsJSON, userId string) (string, error) {
	var args struct {
		StartTime   string `json:"start_time"`
		EndTime     string `json:"end_time"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.StartTime == "" || args.EndTime == "" {
		return "Error: start_time and end_time are both required", nil
	}

	campusRecord, err := c.db.GetCampusloginByUserId(userId)
	var contactID int
	var programID int
	if err != nil {
		log.Printf("Failed to get ContactID/ProgramID for user %s: %v", userId, err)
		contactID = 5972449 // using the default one for fallback just in case
		programID = 1
	} else {
		contactID = int(campusRecord.ContactId)
		programID = int(campusRecord.ProgramId)
	}

	err = c.campuslogin.SendAppointment(ctx, args.StartTime, args.EndTime, contactID, programID, args.Description)
	if err != nil {
		return "", fmt.Errorf("failed to create appointment on CampusLogin: %w", err)
	}

	return "Appointment successfully created on CampusLogin.", nil
}

func (c *Client) handleSendSummary(ctx context.Context, argsJSON, userId, assistantId string) (string, error) {
	var args struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.Summary == "" {
		return "Error: summary parameter is required", nil
	}

	campusRecord, err := c.db.GetCampusloginByUserId(userId)
	var contactID int
	if err != nil {
		log.Printf("Failed to get ContactID for user %s: %v", userId, err)
		contactID = 5972449 // default fallback
	} else {
		contactID = int(campusRecord.ContactId)
	}

	err = c.campuslogin.AddNewNote(ctx, contactID, args.Summary)
	if err != nil {
		return "", fmt.Errorf("failed to add new note on CampusLogin: %w", err)
	}

	chat, err := c.db.GetLatestChatByCustomer(assistantId, userId)
	if err == nil && chat != nil {
		_, err = c.db.UpdateChatIsEnd(chat.Id, true)
		if err != nil {
			log.Printf("Failed to update chat is_end for chat %s: %v", chat.Id, err)
		}
	} else {
		log.Printf("Failed to get latest chat for customer %s to set isEnd: %v", userId, err)
	}

	return "Summary successfully sent to CampusLogin.", nil
}

func (c *Client) handleMarkGrade11OrLower(ctx context.Context, userId string) (string, error) {
	// Let's fetch it first.
	record, err := c.db.GetCampusloginByUserId(userId)
	if err != nil {
		log.Printf("Failed to get campuslogin for user %s: %v", userId, err)
		return "Failed to mark grade 11 or lower.", err
	}

	err = c.db.SetCampusloginFlags(userId, true, record.IsInternationalStudent)
	if err != nil {
		log.Printf("Failed to set campuslogin flags for user %s: %v", userId, err)
		return "Failed to mark grade 11 or lower.", err
	}
	return "Successfully marked user as in Grade 11 or lower. The user is now disqualified from certain follow-ups.", nil
}

func (c *Client) handleMarkInternationalStudent(ctx context.Context, userId string) (string, error) {
	record, err := c.db.GetCampusloginByUserId(userId)
	if err != nil {
		log.Printf("Failed to get campuslogin for user %s: %v", userId, err)
		return "Failed to mark international student.", err
	}

	err = c.db.SetCampusloginFlags(userId, record.IsGrade11OrLower, true)
	if err != nil {
		log.Printf("Failed to set campuslogin flags for user %s: %v", userId, err)
		return "Failed to mark international student.", err
	}
	return "Successfully marked user as an international student or on a visa. The user is now disqualified from certain follow-ups.", nil
}

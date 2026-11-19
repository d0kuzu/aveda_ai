package llm

import (
	"diaxel/internal/constants"
	"diaxel/internal/modules/tutor"
	dbpb "diaxel/proto/db"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

// Message represents a chat message (moved from database models)
type Message struct {
	ChatID  uuid.UUID `json:"chat_id"`
	Role    string    `json:"role"`
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}

func (c *Client) GetMessages(assistantId, customerId string) ([]openai.ChatCompletionMessage, error) {
	// 1. Get chat
	chatResp, err := c.db.GetLatestChatByCustomer(assistantId, customerId)
	if err != nil {
		return nil, err
	}

	var messages []openai.ChatCompletionMessage

	// If chat exists, fetch messages
	if chatResp.Id != "" {
		var messagesResp []*dbpb.MessageResponse
		if chatResp.IsTutor {
			messagesResp, err = c.db.GetAllTutorChatMessages(chatResp.Id)
		} else {
			messagesResp, err = c.db.GetAllChatMessages(chatResp.Id)
		}
		if err != nil {
			return nil, err
		}

		var history []Message
		for _, msg := range messagesResp {
			history = append(history, Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}

		convertedMessages, err := c.ConvertToOpenaiMessage(history)
		if err != nil {
			return nil, err
		}
		messages = append(messages, convertedMessages...)
	}

	// Append system prompt at the end
	var startMessages []openai.ChatCompletionMessage
	if chatResp != nil && chatResp.IsTutor {
		startMessages = c.StartTutorMessages(chatResp)
	} else {
		startMessages, err = c.StartMessages(assistantId, customerId)
		if err != nil {
			return nil, err
		}
	}
	messages = append(messages, startMessages...)

	return messages, nil
}

func (c *Client) StartTutorMessages(chatResp *dbpb.ChatResponse) []openai.ChatCompletionMessage {
	systemPrompt := constants.TutorSystemPrompt

	if tutor.IsCanadaAssistant(chatResp.AssistantId) {
		oldStep1 := `Step 1 (Program Selection):
- The student must first choose ONE of the 4 available programs (ALWAYS format the list with each program on its own new line):
  1. Barber
  2. Cosmetology
  3. Esthetician
  4. Nail Technician
- If they have not chosen a program yet (or ask what programs exist), list these 4 programs on separate lines as shown above and ask which one they want to study.
- When the student chooses one of the 4 programs (by name or number 1-4), MUST call the tool "select_tutor_program".`

		canadaStep1 := `Step 1 (Program Selection):
- The student must first choose ONE of the 2 available programs (ALWAYS format the list with each program on its own new line):
  1. Esthetician / Skin Care
  2. Nail Technician
- If they have not chosen a program yet (or ask what programs exist), list these 2 programs on separate lines as shown above and ask which one they want to study.
- When the student chooses one of the 2 programs (by name or number: 1 = Esthetician / Skin Care -> pass program="esthetician", 2 = Nail Technician -> pass program="nail-technician"), MUST call the tool "select_tutor_program".`

		systemPrompt = strings.Replace(systemPrompt, oldStep1, canadaStep1, 1)
	}

	progName := "None (not selected yet)"
	if chatResp.TutingProgram != "" {
		progName = tutor.ProgramDisplayNameForAssistant(chatResp.AssistantId, chatResp.TutingProgram)
	}

	var actionReminder string
	if chatResp.IsTuting && chatResp.TutorCountPerDay >= 5 {
		actionReminder = "CRITICAL REMINDER: The student has completed 5 / 5 questions today and was asked 'Would you like to do 5 more questions right now?'. If the student's reply is affirmative (yes, yeah, sure, ok, continue), you MUST call `continue_tutor_today` immediately! If the student's reply is negative (no, nope, tomorrow, stop), you MUST call `finish_tutor_for_today` immediately!"
	} else {
		actionReminder = "CRITICAL REMINDER: You do NOT have the question bank in your memory. Every single time the student answers a question (with A/B/C/D, 1/2/3/4, option text, or skip), you MUST call the function `submit_tutor_answer` FIRST so the backend increments `tutor_count` and returns the official next question. NEVER reply with 'Correct!', 'Not quite!', or 'Next question:' without calling `submit_tutor_answer` in this turn!"
	}

	stateInfo := fmt.Sprintf(
		"\n\n### Current Student Tutor State (from Database):\n- Selected Program: %s\n- Tutoring Active (is_tuting): %v\n- Questions Answered Today (tutor_count_per_day): %d / 5\n- Total Questions Completed (tutor_count): %d\n\n%s",
		progName,
		chatResp.IsTuting,
		chatResp.TutorCountPerDay,
		chatResp.TutorCount,
		actionReminder,
	)

	systemPrompt += stateInfo

	return []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}
}

func (c *Client) StartMessages(assistantId, customerId string) ([]openai.ChatCompletionMessage, error) {
	log.Printf("Getting system prompt")

	systemPrompt := "You are a helpful assistant."

	// Try to get dynamic prompt from assistant config via gRPC
	assistant, err := c.db.GetAssistant(assistantId)
	if err != nil {
		return nil, err
	}

	if assistant.Configuration != "" {
		systemPrompt = assistant.Configuration
	}

	if strings.Contains(systemPrompt, "https://cal.com/cinta-aveda/san-francisco-campus-tour") {
		systemPrompt = strings.ReplaceAll(systemPrompt, "https://cal.com/cinta-aveda/san-francisco-campus-tour", "https://cinta.zerde.co/")
	}

	if customerId != "" && strings.Contains(systemPrompt, "https://api.zerde.co/aveda-calendar/") {
		escapedClient := url.QueryEscape(customerId)
		systemPrompt = strings.ReplaceAll(systemPrompt, "https://api.zerde.co/aveda-calendar/", "https://api.zerde.co/aveda-calendar/?client="+escapedClient)
	}
	log.Printf("System prompt for customer %s: %s", customerId, systemPrompt)

	// Inject current date/time so the model can resolve relative dates ("this Monday", "next Friday", etc.)
	loc, err := time.LoadLocation("America/Winnipeg")
	now := time.Now()
	if err == nil {
		now = now.In(loc)
	}
	dateContext := fmt.Sprintf("\n\nToday is %s, %s, %s",
		now.Format("Monday"), now.Format("2006-01-02"), now.Format("15:04 MST"))
	systemPrompt += dateContext

	return []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}, nil
}

func (c *Client) AddMessage(messages *[]openai.ChatCompletionMessage, role string, message string) {
	*messages = append(*messages, openai.ChatCompletionMessage{Role: role, Content: message})
}

func (c *Client) SaveMessages(assistantId, customerId string, messages []openai.ChatCompletionMessage) error {
	log.Printf("Saving messages for user %s", customerId)

	chatResp, err := c.db.GetLatestChatByCustomer(assistantId, customerId)
	if err != nil {
		return err
	}

	chatID := chatResp.Id
	isTutor := chatResp.IsTutor
	messageCount := chatResp.MessageCount

	if chatID == "" {
		newChat, err := c.db.CreateChat(assistantId, customerId, "openai")
		if err != nil {
			return err
		}
		chatID = newChat.Id
		isTutor = newChat.IsTutor
		messageCount = 0
	}

	// Определяем, сколько сообщений уже было (системные + те, что в базе)
	// Нам нужно знать, сколько системных сообщений добавляет StartMessages.
	// Обычно это 1 сообщение.
	var systemCount int
	if isTutor {
		systemCount = 1
	} else {
		systemMessages, _ := c.StartMessages(assistantId, "")
		systemCount = len(systemMessages)
	}

	totalExistingCount := systemCount + int(messageCount)

	// Если новых сообщений в принципе нет, выходим
	if len(messages) <= totalExistingCount {
		return nil
	}

	// Берем только те, которые реально новые
	newCandidateMessages := messages[totalExistingCount:]

	for _, msg := range newCandidateMessages {
		// Фильтруем системные сообщения
		if msg.Role == openai.ChatMessageRoleSystem {
			continue
		}

		content := msg.Content
		if content == "" {
			content = " "
		}

		// Если это вызов функции или ответ функции, сохраняем расширенные данные в JSON
		if msg.Role == openai.ChatMessageRoleTool || (msg.Role == openai.ChatMessageRoleAssistant && len(msg.ToolCalls) > 0) {
			type extendedMsg struct {
				Content    string            `json:"content"`
				ToolCallID string            `json:"tool_call_id,omitempty"`
				ToolCalls  []openai.ToolCall `json:"tool_calls,omitempty"`
			}
			b, err := json.Marshal(extendedMsg{
				Content:    msg.Content,
				ToolCallID: msg.ToolCallID,
				ToolCalls:  msg.ToolCalls,
			})
			if err == nil {
				content = string(b)
			}
		}

		var saveErr error
		if isTutor {
			_, saveErr = c.db.SaveTutorMessage(chatID, msg.Role, content, "openai")
		} else {
			_, saveErr = c.db.SaveMessage(chatID, msg.Role, content, "openai")
		}
		if saveErr != nil {
			log.Printf("Failed to save message: %v", saveErr)
			return fmt.Errorf("failed to save message: %w", saveErr)
		}
	}

	return nil
}

func (c *Client) ConvertToMessage(customerId string, messages []openai.ChatCompletionMessage) []Message {
	var messagesArray []Message

	for _, message := range messages {
		messagesArray = append(messagesArray, Message{
			Role:    message.Role,
			Content: message.Content,
			Time:    time.Now(),
		})
	}

	return messagesArray
}

func (c *Client) ConvertToOpenaiMessage(arrayMessages []Message) ([]openai.ChatCompletionMessage, error) {
	var messages []openai.ChatCompletionMessage

	for _, message := range arrayMessages {
		content := message.Content

		msg := openai.ChatCompletionMessage{
			Role:    message.Role,
			Content: content,
		}

		// Декодируем ToolCallID и ToolCalls, если это tool/assistant сообщение
		if message.Role == openai.ChatMessageRoleTool || message.Role == openai.ChatMessageRoleAssistant {
			type extendedMsg struct {
				Content    string            `json:"content"`
				ToolCallID string            `json:"tool_call_id,omitempty"`
				ToolCalls  []openai.ToolCall `json:"tool_calls,omitempty"`
			}
			var em extendedMsg
			if err := json.Unmarshal([]byte(content), &em); err == nil {
				if message.Role == openai.ChatMessageRoleTool && em.ToolCallID != "" {
					msg.ToolCallID = em.ToolCallID
					msg.Content = em.Content
				} else if message.Role == openai.ChatMessageRoleAssistant && len(em.ToolCalls) > 0 {
					msg.ToolCalls = em.ToolCalls
					msg.Content = em.Content
				}
			}
		}

		if msg.Content == "" {
			msg.Content = " "
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

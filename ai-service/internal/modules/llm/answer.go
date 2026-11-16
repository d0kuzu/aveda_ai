package llm

import (
	"context"
	"diaxel/internal/constants"
	"diaxel/internal/modules/tutor"
	"strings"

	"github.com/sashabaranov/go-openai"
)

func (c *Client) GetAnswer(ctx context.Context, assistantId string, messages []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error) {
	isTutorMode := false
	var systemContent string
	for _, msg := range messages {
		if msg.Role == openai.ChatMessageRoleSystem && strings.Contains(msg.Content, "Aveda AI Exam Prep Tutor") {
			isTutorMode = true
			systemContent = msg.Content
			break
		}
	}

	var tools []openai.Tool
	var toolChoice any
	if isTutorMode {
		tools = constants.TutorTools
		toolChoice = resolveTutorToolChoice(systemContent, messages)
	} else {
		switch assistantId {
		case constants.AvedaSintaAssistantID:
			tools = constants.AvedaSintaTools
		case constants.AvedaCanadaAssistantID:
			tools = constants.AvedaCanadaTools
		default:
			tools = nil
		}
	}

	req := openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
	}
	if toolChoice != nil {
		req.ToolChoice = toolChoice
	}

	response, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}

	return response, nil
}

// resolveTutorToolChoice enforces deterministic tool calling on the first LLM step
// when the student's message clearly answers a question, selects a program, or starts tutoring.
func resolveTutorToolChoice(systemContent string, messages []openai.ChatCompletionMessage) any {
	if len(messages) == 0 {
		return nil
	}
	lastMsg := messages[len(messages)-1]
	// Only enforce ToolChoice on the initial user message of the turn (never after a tool has already run)
	if lastMsg.Role != openai.ChatMessageRoleUser {
		return nil
	}

	userText := strings.TrimSpace(lastMsg.Content)
	userLower := strings.ToLower(userText)
	userClean := strings.Trim(userLower, ".!?)(")

	noProgramSelected := strings.Contains(systemContent, "Selected Program: None")
	isTutingActive := strings.Contains(systemContent, "Tutoring Active (is_tuting): true")
	atDailyLimit := strings.Contains(systemContent, "Questions Answered Today (tutor_count_per_day): 5 / 5")

	// 1. Program selection stage
	if noProgramSelected {
		if _, ok := tutor.NormalizeProgram(userClean); ok {
			return openai.ToolChoice{
				Type:     openai.ToolTypeFunction,
				Function: openai.ToolFunction{Name: "select_tutor_program"},
			}
		}
		return nil
	}

	// 2. Program is selected, waiting for student readiness to start tutoring
	if !isTutingActive {
		switch userClean {
		case "yes", "y", "yeah", "yep", "sure", "ok", "okay", "ready", "start", "lets go", "let's go", "i am ready", "im ready", "i'm ready":
			return openai.ToolChoice{
				Type:     openai.ToolTypeFunction,
				Function: openai.ToolFunction{Name: "start_tutoring"},
			}
		}
		return nil
	}

	// 3. Tutoring is active and student is answering one of the 5 daily questions
	if isTutingActive && !atDailyLimit {
		if isAnswerOrSkipInput(userClean, systemContent) {
			return openai.ToolChoice{
				Type:     openai.ToolTypeFunction,
				Function: openai.ToolFunction{Name: "submit_tutor_answer"},
			}
		}
	}

	return nil
}

func isAnswerOrSkipInput(userClean string, systemContent string) bool {
	switch userClean {
	case "a", "b", "c", "d", "1", "2", "3", "4",
		"option a", "option b", "option c", "option d",
		"skip", "pass", "idk", "i don't know", "i dont know", "dont know", "don't know", "not sure", "next":
		return true
	}

	// Also check if the student typed the text of one of the options A/B/C/D in the active question
	if len(userClean) >= 3 {
		for _, prefix := range []string{"\nA - ", "\nB - ", "\nC - ", "\nD - "} {
			if idx := strings.Index(systemContent, prefix); idx != -1 {
				lineStart := idx + len(prefix)
				lineEnd := strings.Index(systemContent[lineStart:], "\n")
				var optText string
				if lineEnd == -1 {
					optText = systemContent[lineStart:]
				} else {
					optText = systemContent[lineStart : lineStart+lineEnd]
				}
				optLower := strings.ToLower(strings.TrimSpace(optText))
				if optLower != "" && (userClean == optLower || strings.Contains(optLower, userClean)) {
					return true
				}
			}
		}
	}

	return false
}

package llm

import (
	"context"
	"diaxel/internal/modules/tutor"
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) handleSelectTutorProgram(ctx context.Context, argsJSON, userId, assistantId string) (string, error) {
	var args struct {
		Program string `json:"program"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	normProgram, ok := tutor.NormalizeProgram(args.Program)
	if !ok {
		return "Error: Invalid program. Available programs are: 1. Barber, 2. Cosmetology, 3. Esthetician, 4. Nail Technician. Please ask the student to choose one of these four.", nil
	}

	examData, err := tutor.GetExamData(normProgram)
	if err != nil {
		return "", err
	}

	chat, err := c.db.GetLatestChatByCustomer(assistantId, userId)
	if err != nil || chat == nil || chat.Id == "" {
		return "", fmt.Errorf("active tutor chat not found for user %s", userId)
	}

	_, err = c.db.UpdateChatTutorState(chat.Id, false, normProgram, 0, 0)
	if err != nil {
		return "", fmt.Errorf("failed to save selected program: %w", err)
	}

	return fmt.Sprintf(
		"Program '%s' (%d total questions) has been saved for this student. Now confirm their choice of %s and ask if they are ready to start practicing their first 5 questions right now.",
		tutor.ProgramDisplayName(normProgram),
		len(examData.Questions),
		tutor.ProgramDisplayName(normProgram),
	), nil
}

func (c *Client) handleStartTutoring(ctx context.Context, userId, assistantId string) (string, error) {
	chat, err := c.db.GetLatestChatByCustomer(assistantId, userId)
	if err != nil || chat == nil || chat.Id == "" {
		return "", fmt.Errorf("active tutor chat not found for user %s", userId)
	}

	if chat.TutingProgram == "" {
		return "Error: No program has been selected yet. Please ask the student which of the 4 programs (Barber, Cosmetology, Esthetician, or Nail Technician) they want to study first.", nil
	}

	q, total, err := tutor.GetQuestion(chat.TutingProgram, int(chat.TutorCount))
	if err != nil {
		return "", err
	}
	if q == nil {
		return "All questions for this program have already been completed!", nil
	}

	_, err = c.db.UpdateChatTutorState(chat.Id, true, chat.TutingProgram, 0, chat.TutorCount)
	if err != nil {
		return "", fmt.Errorf("failed to update tutoring state: %w", err)
	}

	return fmt.Sprintf(
		"Tutoring started! Present Question %d of %d to the student in the exact A/B/C/D format:\n\n%s",
		int(chat.TutorCount)+1,
		total,
		tutor.FormatQuestion(q),
	), nil
}

func (c *Client) handleGetCurrentTutorQuestion(ctx context.Context, userId, assistantId string) (string, error) {
	chat, err := c.db.GetLatestChatByCustomer(assistantId, userId)
	if err != nil || chat == nil || chat.Id == "" {
		return "", fmt.Errorf("active tutor chat not found for user %s", userId)
	}

	if chat.TutingProgram == "" {
		return "Error: No program selected yet. Ask the student to choose Barber, Cosmetology, Esthetician, or Nail Technician.", nil
	}

	q, total, err := tutor.GetQuestion(chat.TutingProgram, int(chat.TutorCount))
	if err != nil {
		return "", err
	}
	if q == nil {
		return "All questions in this program have been completed.", nil
	}

	return fmt.Sprintf(
		"Current Question (%d of %d):\n\n%s",
		int(chat.TutorCount)+1,
		total,
		tutor.FormatQuestion(q),
	), nil
}

func (c *Client) handleSubmitTutorAnswer(ctx context.Context, argsJSON, userId, assistantId string) (string, error) {
	var args struct {
		SelectedOption string `json:"selected_option"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	chat, err := c.db.GetLatestChatByCustomer(assistantId, userId)
	if err != nil || chat == nil || chat.Id == "" {
		return "", fmt.Errorf("active tutor chat not found for user %s", userId)
	}

	if chat.TutingProgram == "" {
		return "Error: No program selected yet. Ask the student to choose Barber, Cosmetology, Esthetician, or Nail Technician.", nil
	}

	currentIndex := int(chat.TutorCount)
	q, total, err := tutor.GetQuestion(chat.TutingProgram, currentIndex)
	if err != nil {
		return "", err
	}
	if q == nil {
		_, _ = c.db.UpdateChatIsEnd(chat.Id, true)
		return "All questions in this program are already completed! Congratulate the student.", nil
	}

	choice := strings.ToUpper(strings.TrimSpace(args.SelectedOption))
	switch choice {
	case "1":
		choice = "A"
	case "2":
		choice = "B"
	case "3":
		choice = "C"
	case "4":
		choice = "D"
	}

	isSkipped := choice == "SKIP" || choice == ""
	isCorrect := !isSkipped && strings.EqualFold(choice, q.CorrectAnswer)

	var chosenText string
	for _, opt := range q.Options {
		if strings.EqualFold(opt.Label, choice) {
			chosenText = opt.Text
			break
		}
	}

	newTutorCount := chat.TutorCount + 1
	newCountPerDay := chat.TutorCountPerDay + 1

	_, err = c.db.UpdateChatTutorState(chat.Id, true, chat.TutingProgram, newCountPerDay, newTutorCount)
	if err != nil {
		return "", fmt.Errorf("failed to update tutor counters: %w", err)
	}

	// Check if the entire program is completed
	if int(newTutorCount) >= total {
		_, _ = c.db.UpdateChatIsEnd(chat.Id, true)
		resp := map[string]interface{}{
			"answered_question_number": currentIndex + 1,
			"total_questions":          total,
			"student_choice":           choice,
			"student_choice_text":      chosenText,
			"is_correct":               isCorrect,
			"is_skipped":               isSkipped,
			"correct_answer":           q.CorrectAnswer,
			"correct_answer_text":      q.CorrectAnswerText,
			"daily_limit_reached":      false,
			"program_completed":        true,
			"instruction": fmt.Sprintf(
				"First, provide feedback/explanation on question %d (Correct Answer: %s - %s). Then warmly congratulate the student on completing all %d practice questions in the %s program! The course is now finished.",
				currentIndex+1, q.CorrectAnswer, q.CorrectAnswerText, total, tutor.ProgramDisplayName(chat.TutingProgram),
			),
		}
		b, _ := json.Marshal(resp)
		return string(b), nil
	}

	// Check if 5 questions for today have been reached
	if newCountPerDay >= 5 {
		resp := map[string]interface{}{
			"answered_question_number": currentIndex + 1,
			"total_questions":          total,
			"questions_today":          newCountPerDay,
			"student_choice":           choice,
			"student_choice_text":      chosenText,
			"is_correct":               isCorrect,
			"is_skipped":               isSkipped,
			"correct_answer":           q.CorrectAnswer,
			"correct_answer_text":      q.CorrectAnswerText,
			"daily_limit_reached":      true,
			"program_completed":        false,
			"instruction": fmt.Sprintf(
				"First, provide feedback/explanation on question %d (Correct Answer: %s - %s). Then in the same message (after a blank line), tell the student we have completed today's 5 questions and ask if they want to wrap up for today and continue tomorrow, or do 5 more questions today.",
				currentIndex+1, q.CorrectAnswer, q.CorrectAnswerText,
			),
		}
		b, _ := json.Marshal(resp)
		return string(b), nil
	}

	// Otherwise, fetch the next question in today's batch
	nextQ, _, err := tutor.GetQuestion(chat.TutingProgram, int(newTutorCount))
	if err != nil || nextQ == nil {
		return "", fmt.Errorf("failed to fetch next question: %w", err)
	}

	resp := map[string]interface{}{
		"answered_question_number": currentIndex + 1,
		"total_questions":          total,
		"questions_today":          newCountPerDay,
		"student_choice":           choice,
		"student_choice_text":      chosenText,
		"is_correct":               isCorrect,
		"is_skipped":               isSkipped,
		"correct_answer":           q.CorrectAnswer,
		"correct_answer_text":      q.CorrectAnswerText,
		"daily_limit_reached":      false,
		"program_completed":        false,
		"next_question_number":     int(newTutorCount) + 1,
		"next_question":            tutor.FormatQuestion(nextQ),
		"instruction": fmt.Sprintf(
			"First, provide feedback/explanation on question %d (Correct Answer: %s - %s). Then in the same message (separated by a blank line), present Next Question (%d of %d) in the exact A/B/C/D format.",
			currentIndex+1, q.CorrectAnswer, q.CorrectAnswerText, int(newTutorCount)+1, total,
		),
	}
	b, _ := json.Marshal(resp)
	return string(b), nil
}

func (c *Client) handleContinueTutorToday(ctx context.Context, userId, assistantId string) (string, error) {
	chat, err := c.db.GetLatestChatByCustomer(assistantId, userId)
	if err != nil || chat == nil || chat.Id == "" {
		return "", fmt.Errorf("active tutor chat not found for user %s", userId)
	}

	if chat.TutingProgram == "" {
		return "Error: No program selected.", nil
	}

	q, total, err := tutor.GetQuestion(chat.TutingProgram, int(chat.TutorCount))
	if err != nil {
		return "", err
	}
	if q == nil {
		_, _ = c.db.UpdateChatIsEnd(chat.Id, true)
		return "All questions in this program have been completed! Congratulate the student.", nil
	}

	_, err = c.db.UpdateChatTutorState(chat.Id, true, chat.TutingProgram, 0, chat.TutorCount)
	if err != nil {
		return "", fmt.Errorf("failed to reset tutor_count_per_day: %w", err)
	}

	return fmt.Sprintf(
		"Daily counter reset for a new set of 5 questions! Present Question %d of %d to the student in the exact A/B/C/D format:\n\n%s",
		int(chat.TutorCount)+1,
		total,
		tutor.FormatQuestion(q),
	), nil
}

func (c *Client) handleFinishTutorForToday(ctx context.Context, userId, assistantId string) (string, error) {
	chat, err := c.db.GetLatestChatByCustomer(assistantId, userId)
	if err != nil || chat == nil || chat.Id == "" {
		return "", fmt.Errorf("active tutor chat not found for user %s", userId)
	}

	// Ensure tutor_count_per_day is >= 5 so the daily worker knows today's batch was finished
	countPerDay := chat.TutorCountPerDay
	if countPerDay < 5 {
		countPerDay = 5
	}
	_, _ = c.db.UpdateChatTutorState(chat.Id, true, chat.TutingProgram, countPerDay, chat.TutorCount)

	return "Student is done for today. Wish them a great rest of their day and let them know you will message them tomorrow with the next 5 questions!", nil
}

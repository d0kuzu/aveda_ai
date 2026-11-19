package tutor

import (
	"diaxel/internal/constants"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

type QuestionOption struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

type Question struct {
	ID                string           `json:"id"`
	Category          string           `json:"category"`
	Question          string           `json:"question"`
	Options           []QuestionOption `json:"options"`
	CorrectAnswer     string           `json:"correct_answer"`
	CorrectAnswerText string           `json:"correct_answer_text"`
	SourceQuestionID  int              `json:"source_question_id"`
}

type ExamData struct {
	Exam          string     `json:"exam"`
	QuestionCount int        `json:"question_count"`
	Questions     []Question `json:"questions"`
}

var (
	examsOnce      sync.Once
	examsMap       map[string]*ExamData
	canadaExamsMap map[string]*ExamData
)

var programFiles = map[string]string{
	"barber":          "barber.json",
	"cosmetology":     "cosmetology.json",
	"esthetician":     "esthetician.json",
	"nail-technician": "nail-technician.json",
}

func IsCanadaAssistant(assistantID string) bool {
	return strings.EqualFold(strings.TrimSpace(assistantID), constants.AvedaCanadaAssistantID)
}

func NormalizeProgram(input string) (string, bool) {
	s := strings.ToLower(strings.TrimSpace(input))
	s = strings.ReplaceAll(s, "_", "-")
	switch s {
	case "1", "barber", "barbering":
		return "barber", true
	case "2", "cosmetology", "cosmetologist":
		return "cosmetology", true
	case "3", "esthetician", "esthetics", "aesthetician":
		return "esthetician", true
	case "4", "nail-technician", "nail technician", "nails", "nail", "manicuring":
		return "nail-technician", true
	default:
		if strings.Contains(s, "barber") {
			return "barber", true
		}
		if strings.Contains(s, "cosmetolog") {
			return "cosmetology", true
		}
		if strings.Contains(s, "esthetic") || strings.Contains(s, "aesthetic") {
			return "esthetician", true
		}
		if strings.Contains(s, "nail") {
			return "nail-technician", true
		}
		return "", false
	}
}

func ProgramDisplayName(program string) string {
	switch program {
	case "barber":
		return "Barber"
	case "cosmetology":
		return "Cosmetology"
	case "esthetician":
		return "Esthetician"
	case "nail-technician":
		return "Nail Technician"
	default:
		return program
	}
}

func loadExamPool(searchDirs []string, poolLabel string) map[string]*ExamData {
	pool := make(map[string]*ExamData)
	for progKey, fileName := range programFiles {
		var content []byte
		var err error
		for _, dir := range searchDirs {
			path := fmt.Sprintf("%s/%s", dir, fileName)
			content, err = os.ReadFile(path)
			if err == nil {
				break
			}
		}
		if err != nil {
			log.Printf("[Tutor Questions] Error reading %s (%s): %v", fileName, poolLabel, err)
			continue
		}

		var data ExamData
		if err := json.Unmarshal(content, &data); err != nil {
			log.Printf("[Tutor Questions] Error parsing %s (%s): %v", fileName, poolLabel, err)
			continue
		}
		pool[progKey] = &data
		log.Printf("[Tutor Questions] Loaded %s [%s] (%d questions)", data.Exam, poolLabel, len(data.Questions))
	}
	return pool
}

func loadAllExams() {
	defaultDirs := []string{"json", "ai-service/json", "../json", "../ai-service/json"}
	canadaDirs := []string{"json/canada", "ai-service/json/canada", "../json/canada", "../ai-service/json/canada"}

	examsMap = loadExamPool(defaultDirs, "default")
	canadaExamsMap = loadExamPool(canadaDirs, "canada")
}

func GetExamData(program string) (*ExamData, error) {
	return GetExamDataForAssistant("", program)
}

func GetExamDataForAssistant(assistantID, program string) (*ExamData, error) {
	examsOnce.Do(loadAllExams)
	norm, ok := NormalizeProgram(program)
	if !ok {
		return nil, fmt.Errorf("invalid program '%s': must be one of barber, cosmetology, esthetician, nail-technician", program)
	}

	if IsCanadaAssistant(assistantID) {
		if data, exists := canadaExamsMap[norm]; exists && data != nil {
			return data, nil
		}
	}

	data, exists := examsMap[norm]
	if !exists || data == nil {
		return nil, fmt.Errorf("questions for program '%s' not found", norm)
	}
	return data, nil
}

func GetQuestion(program string, index int) (*Question, int, error) {
	return GetQuestionForAssistant("", program, index)
}

func GetQuestionForAssistant(assistantID, program string, index int) (*Question, int, error) {
	data, err := GetExamDataForAssistant(assistantID, program)
	if err != nil {
		return nil, 0, err
	}
	total := len(data.Questions)
	if total == 0 {
		return nil, 0, fmt.Errorf("no questions available for program '%s'", program)
	}
	if index < 0 || index >= total {
		return nil, total, nil
	}
	return &data.Questions[index], total, nil
}

func FormatQuestion(q *Question) string {
	var sb strings.Builder
	sb.WriteString(q.Question)
	for _, opt := range q.Options {
		sb.WriteString(fmt.Sprintf("\n%s - %s", opt.Label, opt.Text))
	}
	return sb.String()
}

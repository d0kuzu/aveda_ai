package tutor

import (
	"diaxel/internal/constants"
	"os"
	"strings"
	"testing"
)

func TestLoadAllExams(t *testing.T) {
	// Change to ai-service root if running from package dir
	if _, err := os.Stat("../../../json/barber.json"); err == nil {
		_ = os.Chdir("../../..")
	}

	programs := map[string]int{
		"barber":          125,
		"cosmetology":     323,
		"esthetician":     236,
		"nail-technician": 233,
	}

	assistantIDs := []string{
		constants.AvedaTutorAssistantID,
		constants.AvedaCanadaAssistantID,
	}

	for _, assistantID := range assistantIDs {
		for prog, expectedCount := range programs {
			q, total, err := GetQuestionForAssistant(assistantID, prog, 0)
			if err != nil {
				t.Fatalf("GetQuestionForAssistant(%s, %s, 0) failed: %v", assistantID, prog, err)
			}
			if total != expectedCount {
				t.Errorf("assistant %s program %s: expected %d questions, got %d", assistantID, prog, expectedCount, total)
			}
			if q == nil {
				t.Fatalf("assistant %s program %s: first question is nil", assistantID, prog)
			}
			formatted := FormatQuestion(q)
			if !strings.Contains(formatted, "A - ") || !strings.Contains(formatted, "D - ") {
				t.Errorf("assistant %s program %s: unexpected formatted question:\n%s", assistantID, prog, formatted)
			}
		}
	}
}

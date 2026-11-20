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

	usPrograms := map[string]int{
		"barber":          125,
		"cosmetology":     323,
		"esthetician":     236,
		"nail-technician": 233,
	}

	for prog, expectedCount := range usPrograms {
		q, total, err := GetQuestionForAssistant(constants.AvedaTutorAssistantID, prog, 0)
		if err != nil {
			t.Fatalf("GetQuestionForAssistant(US, %s, 0) failed: %v", prog, err)
		}
		if total != expectedCount {
			t.Errorf("US program %s: expected %d questions, got %d", prog, expectedCount, total)
		}
		if q == nil {
			t.Fatalf("US program %s: first question is nil", prog)
		}
		formatted := FormatQuestion(q)
		if !strings.Contains(formatted, "A - ") || !strings.Contains(formatted, "B - ") {
			t.Errorf("US program %s: unexpected formatted question:\n%s", prog, formatted)
		}
	}

	// Verify Canada program selection mapping (1 = Esthetician / Skin Care, 2 = Nail Technician, 3 = Barbering, 4 = Hair Styling)
	canadaCases := map[string]string{
		"1":                       "esthetician",
		"Esthetician / Skin Care": "esthetician",
		"Skin Care":               "esthetician",
		"esthetician":             "esthetician",
		"2":                       "nail-technician",
		"Nail Technician":         "nail-technician",
		"nail-technician":         "nail-technician",
		"3":                       "barber",
		"Barbering":               "barber",
		"barbering":               "barber",
		"barber":                  "barber",
		"4":                       "cosmetology",
		"Hair Styling":            "cosmetology",
		"hair-styling":            "cosmetology",
		"hairstyling":             "cosmetology",
		"cosmetology":             "cosmetology",
	}
	for input, expectedNorm := range canadaCases {
		got, ok := NormalizeProgramForAssistant(constants.AvedaCanadaTutorAssistantID, input)
		if !ok || got != expectedNorm {
			t.Errorf("Canada NormalizeProgramForAssistant(%q) = (%q, %v), want (%q, true)", input, got, ok, expectedNorm)
		}
		q, total, err := GetQuestionForAssistant(constants.AvedaCanadaTutorAssistantID, input, 0)
		if err != nil || q == nil || total != 200 {
			t.Errorf("Canada GetQuestionForAssistant(%q) failed: q=%v, total=%d (want 200), err=%v", input, q, total, err)
		}
	}

	// Validate every question in all 4 Canada JSON files and all 4 US JSON files
	for _, assistantID := range []string{constants.AvedaTutorAssistantID, constants.AvedaCanadaTutorAssistantID} {
		for _, prog := range []string{"barber", "cosmetology", "esthetician", "nail-technician"} {
			data, err := GetExamDataForAssistant(assistantID, prog)
			if err != nil || data == nil {
				t.Fatalf("GetExamDataForAssistant(%s, %s) failed: %v", assistantID, prog, err)
			}
			for idx, q := range data.Questions {
				if strings.TrimSpace(q.ID) == "" {
					t.Errorf("[%s/%s] question %d has empty ID", assistantID, prog, idx)
				}
				if strings.TrimSpace(q.Question) == "" {
					t.Errorf("[%s/%s] question %d (%s) has empty Question text", assistantID, prog, idx, q.ID)
				}
				if len(q.Options) < 2 {
					t.Errorf("[%s/%s] question %d (%s) has fewer than 2 options (%d)", assistantID, prog, idx, q.ID, len(q.Options))
				}
				foundCorrect := false
				for _, opt := range q.Options {
					if strings.TrimSpace(opt.Label) == "" || strings.TrimSpace(opt.Text) == "" {
						t.Errorf("[%s/%s] question %d (%s) has option with empty label or text: %+v", assistantID, prog, idx, q.ID, opt)
					}
					if strings.EqualFold(strings.TrimSpace(opt.Label), strings.TrimSpace(q.CorrectAnswer)) {
						foundCorrect = true
					}
				}
				if !foundCorrect {
					t.Errorf("[%s/%s] question %d (%s) correct_answer %q not found among options", assistantID, prog, idx, q.ID, q.CorrectAnswer)
				}
				if strings.TrimSpace(q.CorrectAnswerText) == "" {
					t.Errorf("[%s/%s] question %d (%s) has empty correct_answer_text", assistantID, prog, idx, q.ID)
				}
			}
		}
	}
}

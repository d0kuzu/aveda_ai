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

	// Verify Canada program selection mapping (1 = Esthetician / Skin Care, 2 = Nail Technician)
	canadaCases := map[string]string{
		"1":                       "esthetician",
		"Esthetician / Skin Care": "esthetician",
		"Skin Care":               "esthetician",
		"esthetician":             "esthetician",
		"2":                       "nail-technician",
		"Nail Technician":         "nail-technician",
		"nail-technician":         "nail-technician",
	}
	for input, expectedNorm := range canadaCases {
		got, ok := NormalizeProgramForAssistant(constants.AvedaCanadaAssistantID, input)
		if !ok || got != expectedNorm {
			t.Errorf("Canada NormalizeProgramForAssistant(%q) = (%q, %v), want (%q, true)", input, got, ok, expectedNorm)
		}
		q, total, err := GetQuestionForAssistant(constants.AvedaCanadaAssistantID, input, 0)
		if err != nil || q == nil || total != 200 {
			t.Errorf("Canada GetQuestionForAssistant(%q) failed: q=%v, total=%d (want 200), err=%v", input, q, total, err)
		}
	}

	// Ensure Barber and Cosmetology are not valid for Canada
	for _, invalidCanadaProg := range []string{"barber", "cosmetology", "3", "4"} {
		if _, ok := NormalizeProgramForAssistant(constants.AvedaCanadaAssistantID, invalidCanadaProg); ok {
			t.Errorf("Canada NormalizeProgramForAssistant(%q) should be false", invalidCanadaProg)
		}
	}
}

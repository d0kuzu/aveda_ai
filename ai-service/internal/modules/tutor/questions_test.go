package tutor

import (
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

	for prog, expectedCount := range programs {
		q, total, err := GetQuestion(prog, 0)
		if err != nil {
			t.Fatalf("GetQuestion(%s, 0) failed: %v", prog, err)
		}
		if total != expectedCount {
			t.Errorf("program %s: expected %d questions, got %d", prog, expectedCount, total)
		}
		if q == nil {
			t.Fatalf("program %s: first question is nil", prog)
		}
		formatted := FormatQuestion(q)
		if !strings.Contains(formatted, "A - ") || !strings.Contains(formatted, "D - ") {
			t.Errorf("program %s: unexpected formatted question:\n%s", prog, formatted)
		}
	}
}

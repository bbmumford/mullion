package web

import "testing"

// The button labels are filled once, here, rather than in each platform file,
// so a platform that forgets cannot show an unlabelled button.
func TestQuestionDefaultsFillBothButtons(t *testing.T) {
	got := QuestionOptions{Title: "Allow?"}.withDefaults()
	if got.Confirm != "Yes" {
		t.Errorf("Confirm = %q, want Yes", got.Confirm)
	}
	if got.Cancel != "No" {
		t.Errorf("Cancel = %q, want No", got.Cancel)
	}
}

func TestQuestionDefaultsKeepWhatTheCallerGave(t *testing.T) {
	got := QuestionOptions{Title: "Allow?", Confirm: "Allow", Cancel: "Deny"}.withDefaults()
	if got.Confirm != "Allow" || got.Cancel != "Deny" {
		t.Errorf("labels = %q/%q, want Allow/Deny", got.Confirm, got.Cancel)
	}
}

// A caller may label one button and not the other.
func TestQuestionDefaultsFillOnlyWhatIsMissing(t *testing.T) {
	got := QuestionOptions{Confirm: "Allow"}.withDefaults()
	if got.Confirm != "Allow" {
		t.Errorf("Confirm = %q, want Allow", got.Confirm)
	}
	if got.Cancel != "No" {
		t.Errorf("Cancel = %q, want the default No", got.Cancel)
	}
}

//go:build darwin || linux || windows

package dialog

import (
	"fmt"

	"github.com/bbmumford/mullion/web"
)

// askQuestion hands the prompt to the platform code, which lives beside the
// WebView for the same reason the file dialogs do: that is where each
// platform's window toolkit is already bound.
//
// Every failure is wrapped in ErrNotShown and the answer forced to false, so a
// caller has one check to make and cannot be handed a true it should not act
// on.
func askQuestion(q Question) (bool, error) {
	ok, err := web.AskQuestion(q.Parent, web.QuestionOptions{
		Title:       q.Title,
		Message:     q.Message,
		Confirm:     q.Confirm,
		Cancel:      q.Cancel,
		Destructive: q.Destructive,
	})
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrNotShown, err)
	}
	return ok, nil
}

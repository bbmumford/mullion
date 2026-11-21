//go:build darwin || linux || windows

package dialog

import "github.com/bbmumford/mullion/web"

// askQuestion hands the prompt to the platform code, which lives beside the
// WebView for the same reason the file dialogs do: that is where each
// platform's window toolkit is already bound.
func askQuestion(q Question) (bool, error) {
	return web.AskQuestion(q.Parent, web.QuestionOptions{
		Title:       q.Title,
		Message:     q.Message,
		Confirm:     q.Confirm,
		Cancel:      q.Cancel,
		Destructive: q.Destructive,
	})
}

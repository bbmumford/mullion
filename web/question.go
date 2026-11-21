package web

import "unsafe"

// QuestionOptions is a yes/no prompt, as the dialog package passes it down.
// The field meanings are documented there.
type QuestionOptions struct {
	Title       string
	Message     string
	Confirm     string
	Cancel      string
	Destructive bool
}

// AskQuestion shows a platform alert with two buttons and reports whether the
// affirmative one was chosen. Dismissing without answering is false.
//
// window is the parent, and may be nil. The platform code lives in this
// package because each platform's window toolkit is bound here already; the
// dialog package is the door to it, for either surface.
func AskQuestion(window unsafe.Pointer, opts QuestionOptions) (bool, error) {
	return askQuestion(window, opts.withDefaults())
}

// withDefaults fills the button labels, so each platform file does not repeat
// the same two fallbacks.
func (o QuestionOptions) withDefaults() QuestionOptions {
	if o.Confirm == "" {
		o.Confirm = "Yes"
	}
	if o.Cancel == "" {
		o.Cancel = "No"
	}
	return o
}

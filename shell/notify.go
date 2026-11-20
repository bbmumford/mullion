package shell

import "fmt"

// A Notification is a message posted to the desktop's notification service.
type Notification struct {
	// Title is the bold first line.
	Title string

	// Body is the message under the title.
	Body string
}

// Notify posts n through tray, which must be a tray that Show has brought up.
//
// A tray is not an implementation detail that could be hidden here: on Windows
// a notification is a balloon belonging to a notification-area icon, so without
// a tray there is nothing for the balloon to come from. macOS and Linux would
// each manage without one, and routing all three the same way keeps a
// notification that works on the developer's machine working on the others.
//
// An application with no tray should say so to its user rather than post
// silently; errors.Is(err, errors.ErrUnsupported) does not cover this, because
// the platform is capable and the application simply has no icon.
func Notify(tray *Tray, n Notification) error {
	if tray == nil {
		return fmt.Errorf("shell: a notification needs a tray to come from; give the App a Tray")
	}
	if n.Title == "" && n.Body == "" {
		return fmt.Errorf("shell: the notification has no title and no body")
	}
	return tray.Notify(n.Title, n.Body)
}

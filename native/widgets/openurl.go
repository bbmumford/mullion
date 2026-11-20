package widgets

import . "github.com/bbmumford/mullion/native"

// OpenURL asks the platform backend to open url in the system browser (or the
// scheme's handler) after this frame. Same Host → FrameOutput path as
// RequestTextCopy: widgets never call a backend package. Errors are ignored.
func OpenURL(url string) {
	RequestOpenURL(url)
}

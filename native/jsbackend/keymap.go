package jsbackend

import "github.com/bbmumford/mullion/native"

// KeyboardEvent.code (UI Events) is positional US-QWERTY legend — maps
// directly onto native.KeyCode the way Cocoa kVK_ANSI_* does on macOS.
// Meta/OS keys are not in this table: mapCode picks KeyCommand vs KeySuper
// from the browser host OS (Apple → Command).
var codeToKey = map[string]native.KeyCode{
	"KeyA": native.KeyA, "KeyB": native.KeyB, "KeyC": native.KeyC, "KeyD": native.KeyD,
	"KeyE": native.KeyE, "KeyF": native.KeyF, "KeyG": native.KeyG, "KeyH": native.KeyH,
	"KeyI": native.KeyI, "KeyJ": native.KeyJ, "KeyK": native.KeyK, "KeyL": native.KeyL,
	"KeyM": native.KeyM, "KeyN": native.KeyN, "KeyO": native.KeyO, "KeyP": native.KeyP,
	"KeyQ": native.KeyQ, "KeyR": native.KeyR, "KeyS": native.KeyS, "KeyT": native.KeyT,
	"KeyU": native.KeyU, "KeyV": native.KeyV, "KeyW": native.KeyW, "KeyX": native.KeyX,
	"KeyY": native.KeyY, "KeyZ": native.KeyZ,

	"Digit0": native.Key0, "Digit1": native.Key1, "Digit2": native.Key2, "Digit3": native.Key3,
	"Digit4": native.Key4, "Digit5": native.Key5, "Digit6": native.Key6, "Digit7": native.Key7,
	"Digit8": native.Key8, "Digit9": native.Key9,

	"ArrowLeft": native.KeyLeft, "ArrowRight": native.KeyRight,
	"ArrowUp": native.KeyUp, "ArrowDown": native.KeyDown,

	"Enter": native.KeyEnter, "NumpadEnter": native.KeyEnter,
	"Escape": native.KeyEscape, "Tab": native.KeyTab, "Space": native.KeySpace,
	"Backspace": native.KeyDeleteBackward, "Delete": native.KeyDeleteForward,
	"Home": native.KeyHome, "End": native.KeyEnd,
	"PageUp": native.KeyPageUp, "PageDown": native.KeyPageDown,
	"Insert": native.KeyInsert,

	"ShiftLeft": native.KeyShift, "ShiftRight": native.KeyShift,
	"ControlLeft": native.KeyCtrl, "ControlRight": native.KeyCtrl,
	"AltLeft": native.KeyAlt, "AltRight": native.KeyAlt,

	"F1": native.KeyF1, "F2": native.KeyF2, "F3": native.KeyF3, "F4": native.KeyF4,
	"F5": native.KeyF5, "F6": native.KeyF6, "F7": native.KeyF7, "F8": native.KeyF8,
	"F9": native.KeyF9, "F10": native.KeyF10, "F11": native.KeyF11, "F12": native.KeyF12,
}

// mapCode resolves KeyboardEvent.code to a positional KeyCode.
// appleHost selects Command (macOS/iOS browsers) vs Super (Windows/Linux)
// for MetaLeft/MetaRight/OSLeft/OSRight — matching Cocoa KeyCommand.
func mapCode(code string, appleHost bool) native.KeyCode {
	switch code {
	case "MetaLeft", "MetaRight", "OSLeft", "OSRight":
		if appleHost {
			return native.KeyCommand
		}
		return native.KeySuper
	}
	return codeToKey[code]
}

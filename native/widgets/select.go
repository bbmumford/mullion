package widgets

// Added for mullion; not part of the merged go-shirei snapshot. Kept in a file
// of its own so a later re-sync with upstream can see at a glance what is
// local.

// Select shows the chosen option and opens a menu to change it.
//
// It is MenuButton with the bookkeeping done: the label is whatever value
// currently holds, choosing an option writes it back, and the menu closes
// itself. The keyboard behaviour is the menu's -- Down to open, Up and Down to
// move, Enter to choose, Escape to dismiss -- and typing filters, which a long
// list of options wants and a short one does not notice.
//
// An empty value shows placeholder and matches nothing, so "no choice yet" is
// distinguishable from a chosen empty string.
//
//	widgets.Select(&unit, "Choose a unit", "bytes", "lines", "pages")
//
// It reports true on the frame the value changes, for a caller that needs to
// act on the change rather than read the variable next frame.
func Select(value *string, placeholder string, options ...string) bool {
	return SelectExt(value, placeholder, options, func(opt string) string { return opt })
}

// SelectExt is Select over values that are not their own labels: label turns
// one into the text shown for it, so a list of ids, codes or enum values can be
// presented readably without the caller building a parallel slice of strings.
func SelectExt[T comparable](value *T, placeholder string, options []T, label func(T) string) bool {
	if value == nil || label == nil {
		return false
	}
	shown := selectLabel(value, placeholder, label)

	changed := false
	MenuButton(MenuIcon, shown, func() {
		for _, opt := range options {
			text := label(opt)
			// Typing filters the list. A menu that never asks has no filter
			// field at all, so this is what opts a Select into typeahead.
			if !MenuFilterMatches(text) {
				continue
			}
			icon := NoIcon
			if opt == *value {
				icon = SymITick
			}
			if MenuItem(icon, text) {
				if *value != opt {
					*value = opt
					changed = true
				}
				ClosePopupPanel()
			}
		}
	})
	return changed
}

// selectLabel is the text the trigger shows: the placeholder until something is
// chosen, and the chosen value's label after that. Separate from SelectExt so
// it can be checked without building a frame.
func selectLabel[T comparable](value *T, placeholder string, label func(T) string) string {
	var zero T
	if value == nil || label == nil || *value == zero {
		return placeholder
	}
	return label(*value)
}

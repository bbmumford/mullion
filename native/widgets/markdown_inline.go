package widgets

// Added for mullion; see markdown_parse.go.

import "strings"

// ParseInline reads the inline marks of a line into spans.
//
// Supported: **bold**, *italic* and _italic_, `code`, [text](url), and a
// backslash escape for any punctuation that follows it.
//
// An unterminated mark is not a mark. "a * b" has one asterisk and no italic,
// and comes out as the text that was written, because a reader that swallows
// the rest of a paragraph looking for a closing mark loses more than it fixes.
func ParseInline(s string) []MarkdownSpan {
	var out []MarkdownSpan
	var plain strings.Builder

	flush := func() {
		if plain.Len() > 0 {
			out = append(out, MarkdownSpan{Text: plain.String()})
			plain.Reset()
		}
	}

	for i := 0; i < len(s); {
		switch {
		case s[i] == '\\' && i+1 < len(s) && isMarkdownPunct(s[i+1]):
			// An escape puts the next character through as itself, which is
			// the only way to write a literal asterisk or bracket.
			plain.WriteByte(s[i+1])
			i += 2

		case s[i] == '`':
			if text, next, ok := scanCode(s, i); ok {
				flush()
				out = append(out, MarkdownSpan{Text: text, Code: true})
				i = next
				continue
			}
			plain.WriteByte(s[i])
			i++

		case strings.HasPrefix(s[i:], "**"):
			if text, next, ok := scanDelimited(s, i, "**"); ok {
				flush()
				out = append(out, marked(ParseInline(text), func(sp *MarkdownSpan) { sp.Bold = true })...)
				i = next
				continue
			}
			plain.WriteString("**")
			i += 2

		case s[i] == '*' || s[i] == '_':
			d := string(s[i])
			if text, next, ok := scanDelimited(s, i, d); ok {
				flush()
				out = append(out, marked(ParseInline(text), func(sp *MarkdownSpan) { sp.Italic = true })...)
				i = next
				continue
			}
			plain.WriteByte(s[i])
			i++

		case s[i] == '[':
			if label, url, next, ok := scanLink(s, i); ok {
				flush()
				inner := ParseInline(label)
				if len(inner) == 0 {
					inner = []MarkdownSpan{{Text: label}}
				}
				out = append(out, marked(inner, func(sp *MarkdownSpan) { sp.URL = url })...)
				i = next
				continue
			}
			plain.WriteByte(s[i])
			i++

		default:
			plain.WriteByte(s[i])
			i++
		}
	}
	flush()
	return out
}

// scanDelimited finds the closing delimiter for the one opening at i and
// returns what lies between. An unterminated run is not a match.
//
// Emphasis also has to flank the text it marks, which is what stops "2 * 3 * 4"
// from turning into "2  3  4" with an italic three in the middle. An opener may
// not be followed by a space and a closer may not be preceded by one, so an
// asterisk used as multiplication is multiplication. Underscores carry the
// extra rule that they do not mark inside a word, so a_variable_name keeps its
// underscores; asterisks do mark inside a word, which is how a*b*c italicises
// the b.
func scanDelimited(s string, i int, delim string) (text string, next int, ok bool) {
	start := i + len(delim)
	if start >= len(s) || !canOpen(s, i, delim) {
		return "", 0, false
	}
	// An empty pair is not a mark: "**" on its own is two asterisks.
	for j := start; j+len(delim) <= len(s); j++ {
		if s[j] == '\\' {
			j++ // an escaped delimiter does not close the run
			continue
		}
		if strings.HasPrefix(s[j:], delim) && j > start && canClose(s, j, delim) {
			return s[start:j], j + len(delim), true
		}
	}
	return "", 0, false
}

// canOpen reports whether the delimiter at i can begin a run.
func canOpen(s string, i int, delim string) bool {
	after := i + len(delim)
	if after >= len(s) || isSpaceByte(s[after]) {
		return false
	}
	if delim == "_" && i > 0 && isWordByte(s[i-1]) {
		return false // intra-word underscore
	}
	return true
}

// canClose reports whether the delimiter at j can end a run.
func canClose(s string, j int, delim string) bool {
	if j == 0 || isSpaceByte(s[j-1]) {
		return false
	}
	if delim == "_" {
		after := j + len(delim)
		if after < len(s) && isWordByte(s[after]) {
			return false // intra-word underscore
		}
	}
	return true
}

// Backticks are not emphasis and do not flank: `a * b` is code containing an
// asterisk, and a code span may legitimately begin or end with a space.
func isSpaceByte(c byte) bool { return c == ' ' || c == '\t' || c == '\n' }

func isWordByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// scanLink reads [label](url) starting at i.
func scanLink(s string, i int) (label, url string, next int, ok bool) {
	close := strings.IndexByte(s[i:], ']')
	if close < 0 {
		return "", "", 0, false
	}
	close += i
	if close+1 >= len(s) || s[close+1] != '(' {
		return "", "", 0, false
	}
	end := strings.IndexByte(s[close+2:], ')')
	if end < 0 {
		return "", "", 0, false
	}
	end += close + 2
	return s[i+1 : close], s[close+2 : end], end + 1, true
}

// marked applies a mark to every span of a nested parse, so **a `b` c** keeps
// the code span inside the bold rather than flattening it.
func marked(spans []MarkdownSpan, fn func(*MarkdownSpan)) []MarkdownSpan {
	for i := range spans {
		fn(&spans[i])
	}
	return spans
}

// isMarkdownPunct reports whether a backslash before this character is an
// escape. Escaping a letter is not a thing, and treating it as one would eat
// the backslash out of a Windows path.
func isMarkdownPunct(c byte) bool {
	switch c {
	case '\\', '`', '*', '_', '[', ']', '(', ')', '#', '+', '-', '.', '!', '>', '|', '{', '}':
		return true
	}
	return false
}

// scanCode finds a closing backtick without applying the flanking rules:
// `a * b` is code containing an asterisk, and a code span may begin or end
// with a space.
func scanCode(s string, i int) (text string, next int, ok bool) {
	for j := i + 1; j < len(s); j++ {
		if s[j] == '`' && j > i+1 {
			return s[i+1 : j], j + 1, true
		}
	}
	return "", 0, false
}

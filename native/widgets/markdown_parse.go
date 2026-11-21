package widgets

// Added for mullion; not part of the merged go-mullion snapshot. Kept in files
// of its own so a later re-sync with upstream can see at a glance what is
// local.

import "strings"

// The rule this parser is built around: anything it does not understand comes
// out as the literal text that was written. A renderer that silently drops
// what it cannot parse turns a typo in a document into missing content, which
// is worse than showing the markup. Every path below either produces a span
// carrying the original characters or leaves them in the paragraph text.

// MarkdownKind says what a MarkdownBlock is.
type MarkdownKind int

const (
	// MarkdownParagraph is ordinary prose.
	MarkdownParagraph MarkdownKind = iota
	// MarkdownHeading is a heading; Level is 1 to 6.
	MarkdownHeading
	// MarkdownCode is a code block, shown verbatim in a monospace face.
	MarkdownCode
	// MarkdownQuote is a quoted passage.
	MarkdownQuote
	// MarkdownList is a bullet or numbered list; Items holds the entries.
	MarkdownList
	// MarkdownRule is a horizontal rule and carries no content.
	MarkdownRule
)

// A MarkdownBlock is one top-level piece of a document.
type MarkdownBlock struct {
	Kind MarkdownKind

	// Level is the heading level, 1 to 6, for MarkdownHeading.
	Level int

	// Spans is the inline content, for everything except MarkdownCode and
	// MarkdownRule.
	Spans []MarkdownSpan

	// Text is the verbatim content of a MarkdownCode.
	Text string

	// Lang is a fenced code block's info string, empty when there was none.
	// Nothing here highlights by language; it is carried so a caller can.
	Lang string

	// Ordered says a MarkdownList is numbered rather than bulleted.
	Ordered bool

	// Items are the entries of a MarkdownList, each its own run of spans.
	Items [][]MarkdownSpan
}

// A MarkdownSpan is a run of inline text with one set of marks.
type MarkdownSpan struct {
	Text string

	Bold   bool
	Italic bool
	Code   bool

	// URL makes the span a link. Empty for ordinary text.
	URL string
}

// ParseMarkdown reads a small, documented subset of Markdown into blocks.
//
// Supported: ATX headings (# to ######), paragraphs, fenced code blocks (```)
// and indented ones, blockquotes (>), bullet lists (-, * or +), ordered lists
// (1.), horizontal rules (--- or ***), and inline **bold**, *italic*, `code`
// and [links](url). A backslash escapes the character after it.
//
// Not supported, and deliberately left as the literal text that was written:
// tables, footnotes, images, HTML, reference links, setext headings, nested
// lists. A document using them shows the markup rather than losing the words.
func ParseMarkdown(src string) []MarkdownBlock {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	var blocks []MarkdownBlock

	for i := 0; i < len(lines); {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		switch {
		case trimmed == "":
			i++

		case isRule(trimmed):
			blocks = append(blocks, MarkdownBlock{Kind: MarkdownRule})
			i++

		case strings.HasPrefix(trimmed, "```"):
			block, next := parseFence(lines, i)
			blocks = append(blocks, block)
			i = next

		case headingLevel(trimmed) > 0:
			level := headingLevel(trimmed)
			text := strings.TrimSpace(trimmed[level:])
			blocks = append(blocks, MarkdownBlock{Kind: MarkdownHeading, Level: level, Spans: ParseInline(text)})
			i++

		case strings.HasPrefix(trimmed, ">"):
			block, next := parseQuote(lines, i)
			blocks = append(blocks, block)
			i = next

		case bulletPrefix(trimmed) > 0 || orderedPrefix(trimmed) > 0:
			block, next := parseList(lines, i)
			blocks = append(blocks, block)
			i = next

		case strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t"):
			block, next := parseIndentedCode(lines, i)
			blocks = append(blocks, block)
			i = next

		default:
			block, next := parseParagraph(lines, i)
			blocks = append(blocks, block)
			i = next
		}
	}
	return blocks
}

// isRule reports whether a line is a horizontal rule: three or more of -, * or
// _ and nothing else.
func isRule(s string) bool {
	if len(s) < 3 {
		return false
	}
	c := s[0]
	if c != '-' && c != '*' && c != '_' {
		return false
	}
	for i := range len(s) {
		if s[i] != c {
			return false
		}
	}
	return true
}

// headingLevel returns 1 to 6 for an ATX heading, or 0.
func headingLevel(s string) int {
	n := 0
	for n < len(s) && s[n] == '#' {
		n++
	}
	// "#heading" is not one: a space is required, as every reader requires.
	if n == 0 || n > 6 || n >= len(s) || s[n] != ' ' {
		return 0
	}
	return n
}

// bulletPrefix returns the length of a bullet marker including its space, or 0.
func bulletPrefix(s string) int {
	if len(s) >= 2 && (s[0] == '-' || s[0] == '*' || s[0] == '+') && s[1] == ' ' {
		return 2
	}
	return 0
}

// orderedPrefix returns the length of a "1. " marker, or 0.
func orderedPrefix(s string) int {
	n := 0
	for n < len(s) && s[n] >= '0' && s[n] <= '9' {
		n++
	}
	if n == 0 || n+1 >= len(s) || s[n] != '.' || s[n+1] != ' ' {
		return 0
	}
	return n + 2
}

func parseFence(lines []string, start int) (MarkdownBlock, int) {
	lang := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[start]), "```"))
	var body []string
	i := start + 1
	for ; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			i++ // step over the closing fence
			break
		}
		body = append(body, lines[i])
	}
	// An unclosed fence runs to the end of the document rather than being
	// discarded, which is what every reader does and what loses the least.
	return MarkdownBlock{Kind: MarkdownCode, Text: strings.Join(body, "\n"), Lang: lang}, i
}

func parseIndentedCode(lines []string, start int) (MarkdownBlock, int) {
	var body []string
	i := start
	for ; i < len(lines); i++ {
		switch {
		case strings.HasPrefix(lines[i], "    "):
			body = append(body, lines[i][4:])
		case strings.HasPrefix(lines[i], "\t"):
			body = append(body, lines[i][1:])
		case strings.TrimSpace(lines[i]) == "":
			// A blank line inside indented code belongs to it only if more
			// indented code follows; otherwise the block has ended.
			if i+1 < len(lines) && (strings.HasPrefix(lines[i+1], "    ") || strings.HasPrefix(lines[i+1], "\t")) {
				body = append(body, "")
				continue
			}
			return MarkdownBlock{Kind: MarkdownCode, Text: strings.Join(body, "\n")}, i
		default:
			return MarkdownBlock{Kind: MarkdownCode, Text: strings.Join(body, "\n")}, i
		}
	}
	return MarkdownBlock{Kind: MarkdownCode, Text: strings.Join(body, "\n")}, i
}

func parseQuote(lines []string, start int) (MarkdownBlock, int) {
	var parts []string
	i := start
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(t, ">") {
			break
		}
		parts = append(parts, strings.TrimPrefix(strings.TrimPrefix(t, ">"), " "))
	}
	return MarkdownBlock{Kind: MarkdownQuote, Spans: ParseInline(strings.Join(parts, " "))}, i
}

func parseList(lines []string, start int) (MarkdownBlock, int) {
	ordered := orderedPrefix(strings.TrimSpace(lines[start])) > 0
	block := MarkdownBlock{Kind: MarkdownList, Ordered: ordered}
	i := start
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		n := bulletPrefix(t)
		if ordered {
			n = orderedPrefix(t)
		}
		if n == 0 {
			break
		}
		block.Items = append(block.Items, ParseInline(t[n:]))
	}
	return block, i
}

func parseParagraph(lines []string, start int) (MarkdownBlock, int) {
	var parts []string
	i := start
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if t == "" || isRule(t) || headingLevel(t) > 0 || strings.HasPrefix(t, ">") ||
			strings.HasPrefix(t, "```") || bulletPrefix(t) > 0 || orderedPrefix(t) > 0 {
			break
		}
		parts = append(parts, t)
	}
	// Lines within a paragraph join with a space, as Markdown does: a
	// paragraph wrapped at 80 columns in the source is one paragraph on screen.
	return MarkdownBlock{Kind: MarkdownParagraph, Spans: ParseInline(strings.Join(parts, " "))}, i
}

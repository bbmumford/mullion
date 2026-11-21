package widgets

// Added for mullion; see markdown_parse.go.

import (
	"strings"
	"testing"
)

// plain joins the text of every span, which is what the reader sees whatever
// marks were applied. The rule this parser is built around is that this never
// loses a character the author typed, so most of these tests check it.
func plain(spans []MarkdownSpan) string {
	var b strings.Builder
	for _, s := range spans {
		b.WriteString(s.Text)
	}
	return b.String()
}

func TestHeadingsByLevel(t *testing.T) {
	blocks := ParseMarkdown("# One\n## Two\n###### Six\n")
	if len(blocks) != 3 {
		t.Fatalf("parsed %d blocks, want 3", len(blocks))
	}
	for i, want := range []int{1, 2, 6} {
		if blocks[i].Kind != MarkdownHeading {
			t.Errorf("block %d is kind %v, want a heading", i, blocks[i].Kind)
		}
		if blocks[i].Level != want {
			t.Errorf("block %d is level %d, want %d", i, blocks[i].Level, want)
		}
	}
}

// Seven hashes is not a heading, and a hash with no space is not one either.
// Both have to come out as the text that was written.
func TestWhatIsNotAHeadingStaysText(t *testing.T) {
	for _, src := range []string{"####### Seven", "#NoSpace"} {
		blocks := ParseMarkdown(src)
		if len(blocks) != 1 || blocks[0].Kind != MarkdownParagraph {
			t.Fatalf("%q parsed as %v, want a paragraph", src, blocks[0].Kind)
		}
		if got := plain(blocks[0].Spans); got != src {
			t.Errorf("%q came out as %q", src, got)
		}
	}
}

func TestParagraphLinesJoin(t *testing.T) {
	blocks := ParseMarkdown("one\ntwo\nthree\n\nsecond")
	if len(blocks) != 2 {
		t.Fatalf("parsed %d blocks, want 2", len(blocks))
	}
	if got := plain(blocks[0].Spans); got != "one two three" {
		t.Errorf("wrapped lines joined as %q", got)
	}
}

func TestFencedCodeIsVerbatim(t *testing.T) {
	blocks := ParseMarkdown("```go\nif x {\n\ty()\n}\n```\n")
	if len(blocks) != 1 || blocks[0].Kind != MarkdownCode {
		t.Fatalf("parsed %v, want one code block", blocks)
	}
	if blocks[0].Lang != "go" {
		t.Errorf("language is %q, want go", blocks[0].Lang)
	}
	if blocks[0].Text != "if x {\n\ty()\n}" {
		t.Errorf("code came out as %q", blocks[0].Text)
	}
}

// An unclosed fence runs to the end rather than swallowing the document into
// nothing, which is what loses the least.
func TestAnUnclosedFenceKeepsItsContent(t *testing.T) {
	blocks := ParseMarkdown("```\nstill here\n")
	if len(blocks) != 1 || blocks[0].Kind != MarkdownCode {
		t.Fatalf("parsed %v, want a code block", blocks)
	}
	if !strings.Contains(blocks[0].Text, "still here") {
		t.Errorf("content was lost: %q", blocks[0].Text)
	}
}

func TestListsBulletedAndOrdered(t *testing.T) {
	blocks := ParseMarkdown("- one\n- two\n\n1. first\n2. second\n")
	if len(blocks) != 2 {
		t.Fatalf("parsed %d blocks, want 2", len(blocks))
	}
	if blocks[0].Kind != MarkdownList || blocks[0].Ordered {
		t.Error("the first list is not an unordered list")
	}
	if len(blocks[0].Items) != 2 || plain(blocks[0].Items[1]) != "two" {
		t.Errorf("bullet items are %v", blocks[0].Items)
	}
	if !blocks[1].Ordered {
		t.Error("the numbered list is not marked ordered")
	}
	if len(blocks[1].Items) != 2 || plain(blocks[1].Items[0]) != "first" {
		t.Errorf("ordered items are %v", blocks[1].Items)
	}
}

func TestRulesAndQuotes(t *testing.T) {
	blocks := ParseMarkdown("---\n\n> quoted words\n> more\n")
	if blocks[0].Kind != MarkdownRule {
		t.Errorf("--- parsed as %v, want a rule", blocks[0].Kind)
	}
	if blocks[1].Kind != MarkdownQuote {
		t.Fatalf("> parsed as %v, want a quote", blocks[1].Kind)
	}
	if got := plain(blocks[1].Spans); got != "quoted words more" {
		t.Errorf("quote text is %q", got)
	}
}

func TestInlineMarks(t *testing.T) {
	spans := ParseInline("a **bold** b *italic* c `code` d")
	var bold, italic, code string
	for _, s := range spans {
		switch {
		case s.Bold:
			bold = s.Text
		case s.Italic:
			italic = s.Text
		case s.Code:
			code = s.Text
		}
	}
	if bold != "bold" || italic != "italic" || code != "code" {
		t.Errorf("marks parsed as bold=%q italic=%q code=%q", bold, italic, code)
	}
	if got := plain(spans); got != "a bold b italic c code d" {
		t.Errorf("text came out as %q", got)
	}
}

func TestLinks(t *testing.T) {
	spans := ParseInline("see [the docs](https://example.com/x) now")
	var found bool
	for _, s := range spans {
		if s.URL != "" {
			found = true
			if s.URL != "https://example.com/x" || s.Text != "the docs" {
				t.Errorf("link is %q -> %q", s.Text, s.URL)
			}
		}
	}
	if !found {
		t.Error("no link span was produced")
	}
	if got := plain(spans); got != "see the docs now" {
		t.Errorf("text came out as %q", got)
	}
}

// The rule the whole parser is built around: anything it cannot read comes out
// as what was typed. These are the shapes most likely to tempt a parser into
// swallowing the rest of the line.
func TestUnparseableMarkupSurvivesAsText(t *testing.T) {
	for _, src := range []string{
		"a * b",               // a lone asterisk is multiplication
		"unclosed **bold",     // no closing mark
		"[label] with no url", // a bracket that is not a link
		"[label](unclosed",    // a link that never closes
		"trailing backtick `", // no closing backtick
		"2 * 3 * 4",           // two asterisks that are not italic
		"a_variable_name",     // underscores inside a word
		"C:\\Users\\name",     // a Windows path, not escapes
	} {
		spans := ParseInline(src)
		if got := plain(spans); got != src {
			t.Errorf("ParseInline(%q) lost or changed text: %q", src, got)
		}
	}
}

func TestEscapes(t *testing.T) {
	spans := ParseInline(`\*not italic\* and \[not a link\]`)
	if got := plain(spans); got != "*not italic* and [not a link]" {
		t.Errorf("escapes came out as %q", got)
	}
	for _, s := range spans {
		if s.Italic || s.URL != "" {
			t.Errorf("an escaped mark was applied: %+v", s)
		}
	}
}

// Marks inside marks keep both, so **a `b` c** does not flatten to plain bold.
func TestNestedMarks(t *testing.T) {
	spans := ParseInline("**bold with `code` inside**")
	var sawBoldCode bool
	for _, s := range spans {
		if s.Bold && s.Code {
			sawBoldCode = true
		}
		if !s.Bold {
			t.Errorf("a span inside the bold run is not bold: %+v", s)
		}
	}
	if !sawBoldCode {
		t.Error("the nested code span lost one of its marks")
	}
}

// Nothing in, nothing out, and no panic.
func TestEmptyInput(t *testing.T) {
	if got := ParseMarkdown(""); len(got) != 0 {
		t.Errorf("empty source produced %d blocks", len(got))
	}
	if got := ParseInline(""); len(got) != 0 {
		t.Errorf("empty line produced %d spans", len(got))
	}
}

// Windows line endings must not leave a carriage return in the text.
func TestCarriageReturnsAreNormalised(t *testing.T) {
	blocks := ParseMarkdown("# Title\r\n\r\nbody\r\n")
	if len(blocks) != 2 {
		t.Fatalf("parsed %d blocks, want 2", len(blocks))
	}
	if strings.Contains(plain(blocks[0].Spans), "\r") || strings.Contains(plain(blocks[1].Spans), "\r") {
		t.Error("a carriage return survived into the text")
	}
}

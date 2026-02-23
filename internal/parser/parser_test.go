package parser

import (
	"testing"
)

func TestParseTableValid(t *testing.T) {
	content := "| Front | Back |\n|-------|------|\n| hello | world |\n| foo | bar |\n"
	result := ParseTable(content)

	if len(result.Cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(result.Cards))
	}
	if result.Cards[0].Front != "hello" || result.Cards[0].Back != "world" {
		t.Errorf("card 0 = %+v, want Front=hello Back=world", result.Cards[0])
	}
	if result.Cards[1].Front != "foo" || result.Cards[1].Back != "bar" {
		t.Errorf("card 1 = %+v, want Front=foo Back=bar", result.Cards[1])
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
	}
}

func TestParseTableNoTable(t *testing.T) {
	content := "Just some text\nNothing to see here\n"
	result := ParseTable(content)

	if len(result.Cards) != 0 {
		t.Errorf("expected 0 cards, got %d", len(result.Cards))
	}
}

func TestParseTableWrongHeaders(t *testing.T) {
	content := "| Word | Translation |\n|------|-------------|\n| hello | world |\n"
	result := ParseTable(content)

	if len(result.Cards) != 0 {
		t.Errorf("expected 0 cards, got %d", len(result.Cards))
	}
}

func TestParseTableNoSeparator(t *testing.T) {
	content := "| Front | Back |\n| hello | world |\n"
	result := ParseTable(content)

	if len(result.Cards) != 0 {
		t.Errorf("expected 0 cards for missing separator, got %d", len(result.Cards))
	}
}

func TestParseTableEmptyTable(t *testing.T) {
	content := "| Front | Back |\n|-------|------|\n"
	result := ParseTable(content)

	if len(result.Cards) != 0 {
		t.Errorf("expected 0 cards for empty table, got %d", len(result.Cards))
	}
}

func TestParseTableTrimWhitespace(t *testing.T) {
	content := "| Front | Back |\n|-------|------|\n|  hello  |  world  |\n"
	result := ParseTable(content)

	if len(result.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(result.Cards))
	}
	if result.Cards[0].Front != "hello" {
		t.Errorf("Front = %q, want %q", result.Cards[0].Front, "hello")
	}
	if result.Cards[0].Back != "world" {
		t.Errorf("Back = %q, want %q", result.Cards[0].Back, "world")
	}
}

func TestParseTablePreserveBR(t *testing.T) {
	content := "| Front | Back |\n|-------|------|\n| hello<br>hi | world<br>earth |\n"
	result := ParseTable(content)

	if len(result.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(result.Cards))
	}
	if result.Cards[0].Front != "hello<br>hi" {
		t.Errorf("Front = %q, want %q", result.Cards[0].Front, "hello<br>hi")
	}
	if result.Cards[0].Back != "world<br>earth" {
		t.Errorf("Back = %q, want %q", result.Cards[0].Back, "world<br>earth")
	}
}

func TestParseTableWrongColumnCount(t *testing.T) {
	content := "| Front | Back |\n|-------|------|\n| a | b | c |\n| d | e |\n"
	result := ParseTable(content)

	if len(result.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(result.Cards))
	}
	if result.Cards[0].Front != "d" || result.Cards[0].Back != "e" {
		t.Errorf("card = %+v, want Front=d Back=e", result.Cards[0])
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
	}
	if result.Warnings[0].Line != 3 {
		t.Errorf("warning line = %d, want 3", result.Warnings[0].Line)
	}
}

func TestParseTableEmptyFront(t *testing.T) {
	content := "| Front | Back |\n|-------|------|\n|  | world |\n| hello | earth |\n"
	result := ParseTable(content)

	if len(result.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(result.Cards))
	}
	if result.Cards[0].Front != "hello" {
		t.Errorf("Front = %q, want %q", result.Cards[0].Front, "hello")
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
	}
	if result.Warnings[0].Line != 3 {
		t.Errorf("warning line = %d, want 3", result.Warnings[0].Line)
	}
}

func TestParseTableEmptyBack(t *testing.T) {
	content := "| Front | Back |\n|-------|------|\n| hello |  |\n| foo | bar |\n"
	result := ParseTable(content)

	if len(result.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(result.Cards))
	}
	if result.Cards[0].Front != "foo" {
		t.Errorf("Front = %q, want %q", result.Cards[0].Front, "foo")
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
	}
	if result.Warnings[0].Line != 3 {
		t.Errorf("warning line = %d, want 3", result.Warnings[0].Line)
	}
}

func TestParseTableTableAfterText(t *testing.T) {
	content := "Some intro text\n\n| Front | Back |\n|-------|------|\n| a | b |\n"
	result := ParseTable(content)

	if len(result.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(result.Cards))
	}
	if result.Cards[0].Front != "a" || result.Cards[0].Back != "b" {
		t.Errorf("card = %+v, want Front=a Back=b", result.Cards[0])
	}
}

func TestParseTableWithFrontmatter(t *testing.T) {
	content := "---\ntitle: vocab\n---\n\n| Front | Back |\n|-------|------|\n| hello | world |\n"
	result := ParseTable(content)

	if len(result.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(result.Cards))
	}
	if result.Cards[0].Front != "hello" || result.Cards[0].Back != "world" {
		t.Errorf("card = %+v, want Front=hello Back=world", result.Cards[0])
	}
}

func TestParseTableAlignmentMarkers(t *testing.T) {
	content := "| Front | Back |\n|:------|-----:|\n| a | b |\n"
	result := ParseTable(content)

	if len(result.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(result.Cards))
	}
}

func TestParseTableStopsAtBlankLine(t *testing.T) {
	content := "| Front | Back |\n|-------|------|\n| a | b |\n\n| c | d |\n"
	result := ParseTable(content)

	if len(result.Cards) != 1 {
		t.Fatalf("expected 1 card (table stops at blank line), got %d", len(result.Cards))
	}
}

func TestParseTableStopsAtNonPipeLine(t *testing.T) {
	content := "| Front | Back |\n|-------|------|\n| a | b |\nSome text\n| c | d |\n"
	result := ParseTable(content)

	if len(result.Cards) != 1 {
		t.Fatalf("expected 1 card (table stops at non-pipe line), got %d", len(result.Cards))
	}
}

func TestParseTableFirstTableOnly(t *testing.T) {
	content := "| Front | Back |\n|-------|------|\n| a | b |\n\n| Front | Back |\n|-------|------|\n| c | d |\n"
	result := ParseTable(content)

	if len(result.Cards) != 1 {
		t.Fatalf("expected 1 card (first table only), got %d", len(result.Cards))
	}
	if result.Cards[0].Front != "a" {
		t.Errorf("Front = %q, want %q", result.Cards[0].Front, "a")
	}
}

func TestParseTableMultipleWarnings(t *testing.T) {
	content := "| Front | Back |\n|-------|------|\n| a | b | c |\n|  | world |\n| hello |  |\n| good | card |\n"
	result := ParseTable(content)

	if len(result.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(result.Cards))
	}
	if result.Cards[0].Front != "good" || result.Cards[0].Back != "card" {
		t.Errorf("card = %+v, want Front=good Back=card", result.Cards[0])
	}
	if len(result.Warnings) != 3 {
		t.Fatalf("expected 3 warnings, got %d", len(result.Warnings))
	}
}

func TestParseTablePreservesSpecialContent(t *testing.T) {
	content := "| Front | Back |\n|-------|------|\n| **bold** text | _italic_ & <em>html</em> |\n"
	result := ParseTable(content)

	if len(result.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(result.Cards))
	}
	if result.Cards[0].Front != "**bold** text" {
		t.Errorf("Front = %q, want %q", result.Cards[0].Front, "**bold** text")
	}
	if result.Cards[0].Back != "_italic_ & <em>html</em>" {
		t.Errorf("Back = %q, want %q", result.Cards[0].Back, "_italic_ & <em>html</em>")
	}
}

func TestParseTableNoOuterPipes(t *testing.T) {
	content := "Front | Back\n---|---\nhello | world\n"
	result := ParseTable(content)

	if len(result.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(result.Cards))
	}
	if result.Cards[0].Front != "hello" || result.Cards[0].Back != "world" {
		t.Errorf("card = %+v, want Front=hello Back=world", result.Cards[0])
	}
}

func TestParseTableHeaderOnly(t *testing.T) {
	content := "| Front | Back |\n"
	result := ParseTable(content)

	if len(result.Cards) != 0 {
		t.Errorf("expected 0 cards for header without separator, got %d", len(result.Cards))
	}
}

func TestParseTableCaseSensitiveHeaders(t *testing.T) {
	content := "| front | back |\n|-------|------|\n| a | b |\n"
	result := ParseTable(content)

	if len(result.Cards) != 0 {
		t.Errorf("expected 0 cards for lowercase headers, got %d", len(result.Cards))
	}
}

func TestParseTableEmptyContent(t *testing.T) {
	result := ParseTable("")
	if len(result.Cards) != 0 {
		t.Errorf("expected 0 cards for empty content, got %d", len(result.Cards))
	}
}

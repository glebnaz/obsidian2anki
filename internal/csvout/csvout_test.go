package csvout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebnaz/obsidian2anki/internal/parser"
)

var fixedTime = time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

func TestExportBasic(t *testing.T) {
	dir := t.TempDir()
	cards := []parser.Card{
		{Front: "hello", Back: "world"},
		{Front: "foo", Back: "bar"},
	}

	path, err := Export(cards, dir, "vocab", fixedTime)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	wantName := "vocab-20250115T103000Z.csv"
	if filepath.Base(path) != wantName {
		t.Errorf("filename = %q, want %q", filepath.Base(path), wantName)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	content := string(data)
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 data), got %d", len(lines))
	}
	if lines[0] != "Front;Back" {
		t.Errorf("header = %q, want %q", lines[0], "Front;Back")
	}
	if lines[1] != "hello;world" {
		t.Errorf("line 1 = %q, want %q", lines[1], "hello;world")
	}
	if lines[2] != "foo;bar" {
		t.Errorf("line 2 = %q, want %q", lines[2], "foo;bar")
	}
}

func TestExportSemicolonReplacement(t *testing.T) {
	dir := t.TempDir()
	cards := []parser.Card{
		{Front: "a;b", Back: "c;d;e"},
	}

	path, err := Export(cards, dir, "test", fixedTime)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if lines[1] != "a,b;c,d,e" {
		t.Errorf("line 1 = %q, want %q", lines[1], "a,b;c,d,e")
	}
}

func TestExportLineEndingNormalization(t *testing.T) {
	dir := t.TempDir()
	cards := []parser.Card{
		{Front: "a\r\nb", Back: "c\rd"},
	}

	path, err := Export(cards, dir, "test", fixedTime)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	content := string(data)
	// After normalization, \r\n -> \n and \r -> \n.
	// The data line will be: "a\nb;c\nd\n"
	// Full content: "Front;Back\na\nb;c\nd\n"
	want := "Front;Back\na\nb;c\nd\n"
	if content != want {
		t.Errorf("content = %q, want %q", content, want)
	}
	// Verify no \r remains
	if strings.Contains(content, "\r") {
		t.Errorf("content still contains \\r")
	}
}

func TestExportNoCards(t *testing.T) {
	dir := t.TempDir()
	_, err := Export(nil, dir, "empty", fixedTime)
	if err == nil {
		t.Fatal("expected error for empty cards")
	}
	if !strings.Contains(err.Error(), "no cards") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "no cards")
	}
}

func TestExportExactlyOneSemicolonPerLine(t *testing.T) {
	dir := t.TempDir()
	cards := []parser.Card{
		{Front: "hello", Back: "world"},
		{Front: "a;b;c", Back: "d;e"},
	}

	path, err := Export(cards, dir, "validate", fixedTime)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	for i, line := range lines[1:] {
		count := strings.Count(line, ";")
		if count != 1 {
			t.Errorf("data line %d has %d semicolons: %q", i+1, count, line)
		}
	}
}

func TestExportCreatesDirectory(t *testing.T) {
	base := t.TempDir()
	csvDir := filepath.Join(base, "sub", "dir")

	cards := []parser.Card{
		{Front: "a", Back: "b"},
	}

	path, err := Export(cards, csvDir, "test", fixedTime)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("csv file does not exist at %s", path)
	}
}

func TestExportNoQuoting(t *testing.T) {
	dir := t.TempDir()
	cards := []parser.Card{
		{Front: "hello world", Back: "foo bar"},
		{Front: "has, comma", Back: "also, comma"},
	}

	path, err := Export(cards, dir, "noquote", fixedTime)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "\"") {
		t.Errorf("csv contains quotes, expected no quoting: %s", content)
	}
}

func TestExportPreservesBRTags(t *testing.T) {
	dir := t.TempDir()
	cards := []parser.Card{
		{Front: "hello<br>hi", Back: "world<br>earth"},
	}

	path, err := Export(cards, dir, "br", fixedTime)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if lines[1] != "hello<br>hi;world<br>earth" {
		t.Errorf("line 1 = %q, want %q", lines[1], "hello<br>hi;world<br>earth")
	}
}

func TestSanitizeField(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"a;b", "a,b"},
		{"a;b;c", "a,b,c"},
		{"a\r\nb", "a\nb"},
		{"a\rb", "a\nb"},
		{"a\nb", "a\nb"},
		{"a;b\r\nc;d\re", "a,b\nc,d\ne"},
	}

	for _, tt := range tests {
		got := sanitizeField(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeField(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExportTimestampFormat(t *testing.T) {
	dir := t.TempDir()
	cards := []parser.Card{{Front: "a", Back: "b"}}
	ts := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	path, err := Export(cards, dir, "ts", ts)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	wantName := "ts-20241231T235959Z.csv"
	if filepath.Base(path) != wantName {
		t.Errorf("filename = %q, want %q", filepath.Base(path), wantName)
	}
}

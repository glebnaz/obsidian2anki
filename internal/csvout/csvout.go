package csvout

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebnaz/obsidian2anki/internal/parser"
)

// Export writes cards to a CSV file in csvDir using semicolon as the delimiter.
// The filename is derived from mdBaseName with an RFC3339-basic timestamp appended.
// It returns the path to the written CSV file.
func Export(cards []parser.Card, csvDir, mdBaseName string, now time.Time) (string, error) {
	if len(cards) == 0 {
		return "", fmt.Errorf("no cards to export")
	}

	ts := now.UTC().Format("20060102T150405Z")
	filename := mdBaseName + "-" + ts + ".csv"
	path := filepath.Join(csvDir, filename)

	var b strings.Builder
	b.WriteString("Front;Back\n")

	for i, card := range cards {
		front := sanitizeField(card.Front)
		back := sanitizeField(card.Back)
		line := front + ";" + back
		if strings.Count(line, ";") != 1 {
			return "", fmt.Errorf("line %d has %d semicolons, expected 1", i+1, strings.Count(line, ";"))
		}
		b.WriteString(line + "\n")
	}

	if err := os.MkdirAll(csvDir, 0o755); err != nil {
		return "", fmt.Errorf("create csv dir: %w", err)
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("write csv: %w", err)
	}

	return path, nil
}

// sanitizeField replaces semicolons with commas and normalizes line endings.
func sanitizeField(s string) string {
	s = strings.ReplaceAll(s, ";", ",")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

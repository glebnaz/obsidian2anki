package obsidian

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileInfo holds scan results for a single markdown file.
type FileInfo struct {
	Path       string
	Synced     bool
	HasTable   bool
	CardsCount int
}

// ScanFiles recursively lists all .md files under notesDir,
// sorted lexicographically, and returns scan info for each.
func ScanFiles(notesDir string) ([]FileInfo, error) {
	paths, err := listMarkdownFiles(notesDir)
	if err != nil {
		return nil, err
	}

	var results []FileInfo
	for _, p := range paths {
		info, err := scanFile(p)
		if err != nil {
			return nil, fmt.Errorf("scanning %s: %w", p, err)
		}
		results = append(results, info)
	}
	return results, nil
}

func listMarkdownFiles(dir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", dir, err)
	}
	sort.Strings(paths)
	return paths, nil
}

func scanFile(path string) (FileInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FileInfo{}, fmt.Errorf("reading file: %w", err)
	}

	content := string(data)

	synced := isSynced(content)
	hasTable, cardsCount := detectTable(content)

	return FileInfo{
		Path:       path,
		Synced:     synced,
		HasTable:   hasTable,
		CardsCount: cardsCount,
	}, nil
}

// isSynced extracts YAML frontmatter and checks if anki_synced is boolean true.
func isSynced(content string) bool {
	fm := extractFrontmatter(content)
	if fm == "" {
		return false
	}

	var meta map[string]any
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return false
	}

	val, ok := meta["anki_synced"]
	if !ok {
		return false
	}

	b, ok := val.(bool)
	return ok && b
}

// extractFrontmatter returns the YAML frontmatter content (without delimiters)
// only if the first line of the content is exactly "---".
func extractFrontmatter(content string) string {
	// Normalize \r\n to \n for consistent handling.
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	lines := strings.Split(content, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return ""
	}

	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			return strings.Join(lines[1:i], "\n")
		}
	}
	return ""
}

// detectTable does a lightweight check for a markdown table with Front|Back headers.
// Returns whether such a table exists and how many data rows it has.
func detectTable(content string) (bool, int) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if isHeaderRow(line) {
			// Check next line is a separator
			if !scanner.Scan() {
				return false, 0
			}
			sep := strings.TrimSpace(scanner.Text())
			if !isSeparatorRow(sep) {
				continue
			}
			// Count data rows
			count := 0
			for scanner.Scan() {
				row := strings.TrimSpace(scanner.Text())
				if row == "" || !strings.Contains(row, "|") {
					break
				}
				cells := splitTableRow(row)
				if len(cells) == 2 {
					count++
				}
			}
			return true, count
		}
	}
	return false, 0
}

// isHeaderRow checks if a line is a table header with exactly Front and Back columns.
func isHeaderRow(line string) bool {
	cells := splitTableRow(line)
	if len(cells) != 2 {
		return false
	}
	return cells[0] == "Front" && cells[1] == "Back"
}

// isSeparatorRow checks if a line is a valid markdown table separator.
func isSeparatorRow(line string) bool {
	cells := splitTableRow(line)
	if len(cells) < 2 {
		return false
	}
	for _, cell := range cells {
		cleaned := strings.ReplaceAll(cell, "-", "")
		cleaned = strings.ReplaceAll(cleaned, ":", "")
		if cleaned != "" {
			return false
		}
		if !strings.Contains(cell, "-") {
			return false
		}
	}
	return true
}

// splitTableRow splits a markdown table row by | and trims whitespace.
func splitTableRow(line string) []string {
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

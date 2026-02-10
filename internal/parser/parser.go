package parser

import (
	"bufio"
	"fmt"
	"strings"
)

// Card holds the front and back content extracted from a table row.
type Card struct {
	Front string
	Back  string
}

// Warning describes a skipped or problematic row during parsing.
type Warning struct {
	Line    int
	Message string
}

// Result holds parsed cards and any warnings from parsing.
type Result struct {
	Cards    []Card
	Warnings []Warning
}

// ParseTable finds the first markdown table with exactly "Front" and "Back"
// headers (case-sensitive), parses data rows into cards, and returns warnings
// for skipped rows.
func ParseTable(content string) Result {
	var result Result
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !isHeaderRow(trimmed) {
			continue
		}

		// Check next line is a valid separator row.
		if !scanner.Scan() {
			return result
		}
		lineNum++
		sep := strings.TrimSpace(scanner.Text())
		if !isSeparatorRow(sep) {
			continue
		}

		// Parse data rows.
		for scanner.Scan() {
			lineNum++
			row := scanner.Text()
			trimmedRow := strings.TrimSpace(row)
			if trimmedRow == "" || !strings.Contains(trimmedRow, "|") {
				break
			}

			cells := splitTableRow(trimmedRow)
			if len(cells) != 2 {
				result.Warnings = append(result.Warnings, Warning{
					Line:    lineNum,
					Message: fmt.Sprintf("expected 2 cells, got %d; skipping row", len(cells)),
				})
				continue
			}

			front := strings.TrimSpace(cells[0])
			back := strings.TrimSpace(cells[1])

			if front == "" {
				result.Warnings = append(result.Warnings, Warning{
					Line:    lineNum,
					Message: "Front cell is empty; skipping row",
				})
				continue
			}

			if back == "" {
				result.Warnings = append(result.Warnings, Warning{
					Line:    lineNum,
					Message: "Back cell is empty; skipping row",
				})
				continue
			}

			result.Cards = append(result.Cards, Card{
				Front: front,
				Back:  back,
			})
		}

		return result
	}

	return result
}

// isHeaderRow checks if a line is a table header with exactly "Front" and "Back" columns.
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

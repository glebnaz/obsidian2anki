package obsidian

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanFilesEmpty(t *testing.T) {
	dir := t.TempDir()
	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestScanFilesNonexistentDir(t *testing.T) {
	_, err := ScanFiles("/nonexistent/dir")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestScanFilesSorted(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "c.md"), "no frontmatter")
	writeFile(t, filepath.Join(dir, "a.md"), "no frontmatter")
	writeFile(t, filepath.Join(dir, "b.md"), "no frontmatter")

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}

	names := []string{
		filepath.Base(files[0].Path),
		filepath.Base(files[1].Path),
		filepath.Base(files[2].Path),
	}
	if names[0] != "a.md" || names[1] != "b.md" || names[2] != "c.md" {
		t.Errorf("files not sorted: %v", names)
	}
}

func TestScanFilesRecursive(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "root.md"), "content")
	writeFile(t, filepath.Join(dir, "sub", "nested.md"), "content")
	writeFile(t, filepath.Join(dir, "sub", "deep", "deep.md"), "content")
	writeFile(t, filepath.Join(dir, "not_md.txt"), "content")

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("expected 3 .md files, got %d", len(files))
	}
}

func TestScanFileSyncedTrue(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "synced.md"), "---\nanki_synced: true\n---\nSome content\n")

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if !files[0].Synced {
		t.Error("expected synced=true")
	}
}

func TestScanFileSyncedFalse(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "not_synced.md"), "---\nanki_synced: false\n---\nContent\n")

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if files[0].Synced {
		t.Error("expected synced=false for anki_synced: false")
	}
}

func TestScanFileNoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "plain.md"), "Just some text\nNo frontmatter here\n")

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if files[0].Synced {
		t.Error("expected synced=false when no frontmatter")
	}
}

func TestScanFileFrontmatterNoAnkiSynced(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "no_key.md"), "---\ntitle: test\n---\nContent\n")

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if files[0].Synced {
		t.Error("expected synced=false when anki_synced key missing")
	}
}

func TestScanFileAnkiSyncedNotBoolean(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "string_val.md"), "---\nanki_synced: \"yes\"\n---\nContent\n")

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if files[0].Synced {
		t.Error("expected synced=false when anki_synced is not boolean")
	}
}

func TestScanFileAnkiSyncedInteger(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "int_val.md"), "---\nanki_synced: 1\n---\nContent\n")

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if files[0].Synced {
		t.Error("expected synced=false when anki_synced is integer")
	}
}

func TestScanFileFrontmatterNotFirst(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "late_fm.md"), "Some text\n---\nanki_synced: true\n---\n")

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if files[0].Synced {
		t.Error("expected synced=false when --- is not the first line")
	}
}

func TestScanFileFrontmatterUnclosed(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "unclosed.md"), "---\nanki_synced: true\nNo closing delimiter\n")

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if files[0].Synced {
		t.Error("expected synced=false when frontmatter is not closed")
	}
}

func TestScanFileWithTable(t *testing.T) {
	dir := t.TempDir()
	content := `---
title: vocab
---

| Front | Back |
|-------|------|
| hello | world |
| foo   | bar   |
`
	writeFile(t, filepath.Join(dir, "vocab.md"), content)

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !files[0].HasTable {
		t.Error("expected has_table=true")
	}
	if files[0].CardsCount != 2 {
		t.Errorf("expected cards_count=2, got %d", files[0].CardsCount)
	}
}

func TestScanFileNoTable(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "no_table.md"), "---\ntitle: test\n---\nJust text\n")

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if files[0].HasTable {
		t.Error("expected has_table=false")
	}
	if files[0].CardsCount != 0 {
		t.Errorf("expected cards_count=0, got %d", files[0].CardsCount)
	}
}

func TestScanFileTableWrongHeaders(t *testing.T) {
	dir := t.TempDir()
	content := `| Word | Translation |
|------|-------------|
| hello | world |
`
	writeFile(t, filepath.Join(dir, "wrong_header.md"), content)

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if files[0].HasTable {
		t.Error("expected has_table=false for wrong headers")
	}
}

func TestScanFileTableNoSeparator(t *testing.T) {
	dir := t.TempDir()
	content := `| Front | Back |
| hello | world |
`
	writeFile(t, filepath.Join(dir, "no_sep.md"), content)

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if files[0].HasTable {
		t.Error("expected has_table=false when no separator row")
	}
}

func TestScanFileTableWithBR(t *testing.T) {
	dir := t.TempDir()
	content := `| Front | Back |
|-------|------|
| hello<br>hi | world<br>earth |
`
	writeFile(t, filepath.Join(dir, "br.md"), content)

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !files[0].HasTable {
		t.Error("expected has_table=true")
	}
	if files[0].CardsCount != 1 {
		t.Errorf("expected cards_count=1, got %d", files[0].CardsCount)
	}
}

func TestScanFileCRLFLineEndings(t *testing.T) {
	dir := t.TempDir()
	content := "---\r\nanki_synced: true\r\n---\r\nContent\r\n"
	writeFile(t, filepath.Join(dir, "crlf.md"), content)

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !files[0].Synced {
		t.Error("expected synced=true with CRLF line endings")
	}
}

func TestExtractFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"valid", "---\ntitle: test\n---\nbody", "title: test"},
		{"empty fm", "---\n---\nbody", ""},
		{"no fm", "no frontmatter", ""},
		{"not first line", "text\n---\ntitle: test\n---\n", ""},
		{"unclosed", "---\ntitle: test\n", ""},
		{"crlf", "---\r\ntitle: test\r\n---\r\nbody", "title: test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFrontmatter(tt.content)
			if got != tt.want {
				t.Errorf("extractFrontmatter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsSynced(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"true", "---\nanki_synced: true\n---\n", true},
		{"false", "---\nanki_synced: false\n---\n", false},
		{"missing key", "---\ntitle: test\n---\n", false},
		{"no frontmatter", "just text", false},
		{"string value", "---\nanki_synced: \"yes\"\n---\n", false},
		{"integer value", "---\nanki_synced: 1\n---\n", false},
		{"invalid yaml", "---\n: :\n---\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSynced(tt.content)
			if got != tt.want {
				t.Errorf("isSynced() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectTable(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantTable  bool
		wantCards  int
	}{
		{
			"valid table",
			"| Front | Back |\n|-------|------|\n| a | b |\n| c | d |\n",
			true, 2,
		},
		{
			"no table",
			"just text",
			false, 0,
		},
		{
			"wrong headers",
			"| Word | Def |\n|------|-----|\n| a | b |\n",
			false, 0,
		},
		{
			"no separator",
			"| Front | Back |\n| a | b |\n",
			false, 0,
		},
		{
			"empty table",
			"| Front | Back |\n|-------|------|\n",
			true, 0,
		},
		{
			"table after text",
			"Some intro text\n\n| Front | Back |\n|-------|------|\n| a | b |\n",
			true, 1,
		},
		{
			"table with alignment markers",
			"| Front | Back |\n|:------|-----:|\n| a | b |\n",
			true, 1,
		},
		{
			"row with wrong column count skipped",
			"| Front | Back |\n|-------|------|\n| a | b | c |\n| d | e |\n",
			true, 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTable, gotCards := detectTable(tt.content)
			if gotTable != tt.wantTable {
				t.Errorf("detectTable() hasTable = %v, want %v", gotTable, tt.wantTable)
			}
			if gotCards != tt.wantCards {
				t.Errorf("detectTable() cardsCount = %d, want %d", gotCards, tt.wantCards)
			}
		})
	}
}

func TestSplitTableRow(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string
	}{
		{"with pipes", "| a | b |", []string{"a", "b"}},
		{"no outer pipes", "a | b", []string{"a", "b"}},
		{"leading pipe only", "| a | b", []string{"a", "b"}},
		{"extra spaces", "|  hello  |  world  |", []string{"hello", "world"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitTableRow(tt.line)
			if len(got) != len(tt.want) {
				t.Fatalf("splitTableRow() len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitTableRow()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsSeparatorRow(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"basic", "|-------|------|", true},
		{"with colons", "|:------|-----:|", true},
		{"center", "|:-----:|:----:|", true},
		{"no dashes", "|       |      |", false},
		{"text", "| hello | world |", false},
		{"single cell", "|------|", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSeparatorRow(tt.line)
			if got != tt.want {
				t.Errorf("isSeparatorRow(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

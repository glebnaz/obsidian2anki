package obsidian

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var testTime = time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

func TestMarkSynced_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("# Hello\nSome content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := MarkSynced(path, MarkSyncedOptions{
		Deck:  "TestDeck",
		Model: "Basic",
		Now:   testTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.HasPrefix(content, "---\n") {
		t.Error("expected content to start with ---")
	}
	assertContains(t, content, "anki_synced: true")
	assertContains(t, content, "anki_synced_at: 2025-06-15T10:30:00Z")
	assertContains(t, content, "anki_deck: TestDeck")
	assertContains(t, content, "anki_model: Basic")
	assertContains(t, content, "# Hello")
	assertContains(t, content, "Some content")
}

func TestMarkSynced_ExistingFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	original := "---\ntitle: My Note\ntags: [vocab]\n---\n# Hello\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	err := MarkSynced(path, MarkSyncedOptions{
		Deck:  "German",
		Model: "Basic",
		Now:   testTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	assertContains(t, content, "title: My Note")
	assertContains(t, content, "anki_synced: true")
	assertContains(t, content, "anki_synced_at: 2025-06-15T10:30:00Z")
	assertContains(t, content, "anki_deck: German")
	assertContains(t, content, "anki_model: Basic")
	assertContains(t, content, "# Hello")
}

func TestMarkSynced_UpdateExistingSyncedFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	original := "---\nanki_synced: false\ntitle: test\n---\nBody\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	err := MarkSynced(path, MarkSyncedOptions{
		Deck:  "Vocab",
		Model: "Basic",
		Now:   testTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	assertContains(t, content, "anki_synced: true")
	assertContains(t, content, "title: test")
	assertContains(t, content, "anki_deck: Vocab")
	assertContains(t, content, "Body")

	// Should not have duplicate anki_synced
	if strings.Count(content, "anki_synced:") != 1 {
		t.Errorf("expected exactly one anki_synced key, got %d in:\n%s",
			strings.Count(content, "anki_synced:"), content)
	}
}

func TestMarkSynced_UpdateExistingSyncKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	original := "---\nanki_synced: true\nanki_synced_at: 2024-01-01T00:00:00Z\nanki_deck: OldDeck\nanki_model: OldModel\n---\nBody\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	err := MarkSynced(path, MarkSyncedOptions{
		Deck:  "NewDeck",
		Model: "NewModel",
		Now:   testTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	assertContains(t, content, "anki_synced: true")
	assertContains(t, content, "anki_synced_at: 2025-06-15T10:30:00Z")
	assertContains(t, content, "anki_deck: NewDeck")
	assertContains(t, content, "anki_model: NewModel")
	assertNotContains(t, content, "OldDeck")
	assertNotContains(t, content, "OldModel")
	assertNotContains(t, content, "2024-01-01")
}

func TestMarkSynced_RFC3339Timestamp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ts := time.Date(2025, 12, 31, 23, 59, 59, 0, time.FixedZone("CET", 3600))
	err := MarkSynced(path, MarkSyncedOptions{
		Deck:  "D",
		Model: "M",
		Now:   ts,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	assertContains(t, content, "anki_synced_at: 2025-12-31T23:59:59+01:00")
}

func TestMarkSynced_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := MarkSynced(path, MarkSyncedOptions{
		Deck:  "D",
		Model: "M",
		Now:   testTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify no temp files left behind.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestMarkSynced_PreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := MarkSynced(path, MarkSyncedOptions{
		Deck:  "D",
		Model: "M",
		Now:   testTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected permissions 0600, got %o", info.Mode().Perm())
	}
}

func TestMarkSynced_MarkCheckbox_ReplaceExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	original := "---\ntitle: test\n---\n# Heading\n- [ ] anki_synced\nMore text\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	err := MarkSynced(path, MarkSyncedOptions{
		Deck:         "D",
		Model:        "M",
		MarkCheckbox: true,
		Now:          testTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	assertContains(t, content, "- [x] anki_synced")
	assertNotContains(t, content, "- [ ] anki_synced")
	assertContains(t, content, "More text")
}

func TestMarkSynced_MarkCheckbox_AppendIfAbsent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	original := "---\ntitle: test\n---\n# Content\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	err := MarkSynced(path, MarkSyncedOptions{
		Deck:         "D",
		Model:        "M",
		MarkCheckbox: true,
		Now:          testTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	assertContains(t, content, "- [x] anki_synced")
}

func TestMarkSynced_MarkCheckbox_AlreadyChecked(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	original := "---\ntitle: test\n---\n- [x] anki_synced\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	err := MarkSynced(path, MarkSyncedOptions{
		Deck:         "D",
		Model:        "M",
		MarkCheckbox: true,
		Now:          testTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Should have exactly one checked checkbox.
	count := strings.Count(content, "- [x] anki_synced")
	if count != 1 {
		t.Errorf("expected exactly one checked checkbox, got %d", count)
	}
}

func TestMarkSynced_MarkCheckboxFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	original := "---\ntitle: test\n---\n- [ ] anki_synced\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	err := MarkSynced(path, MarkSyncedOptions{
		Deck:         "D",
		Model:        "M",
		MarkCheckbox: false,
		Now:          testTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Checkbox should remain unchecked when MarkCheckbox is false.
	assertContains(t, content, "- [ ] anki_synced")
}

func TestMarkSynced_CRLFNormalization(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	original := "---\r\ntitle: test\r\n---\r\n# Content\r\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	err := MarkSynced(path, MarkSyncedOptions{
		Deck:  "D",
		Model: "M",
		Now:   testTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	assertContains(t, content, "anki_synced: true")
	assertContains(t, content, "title: test")
	assertContains(t, content, "# Content")
}

func TestMarkSynced_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	err := MarkSynced(path, MarkSyncedOptions{
		Deck:  "D",
		Model: "M",
		Now:   testTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	assertContains(t, content, "---")
	assertContains(t, content, "anki_synced: true")
}

func TestMarkSynced_FileNotFound(t *testing.T) {
	err := MarkSynced("/nonexistent/path.md", MarkSyncedOptions{
		Deck:  "D",
		Model: "M",
		Now:   testTime,
	})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestMarkSynced_PreservesBody(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	original := "---\ntitle: vocab\n---\n\n# German Words\n\n| Front | Back |\n| --- | --- |\n| Hund | dog |\n| Katze | cat |\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	err := MarkSynced(path, MarkSyncedOptions{
		Deck:  "German",
		Model: "Basic",
		Now:   testTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	assertContains(t, content, "# German Words")
	assertContains(t, content, "| Hund | dog |")
	assertContains(t, content, "| Katze | cat |")
	assertContains(t, content, "title: vocab")
	assertContains(t, content, "anki_synced: true")
}

func TestMarkSynced_DefaultTimeUsesNow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	before := time.Now()
	err := MarkSynced(path, MarkSyncedOptions{
		Deck:  "D",
		Model: "M",
		// Now is zero - should use time.Now()
	})
	if err != nil {
		t.Fatal(err)
	}
	after := time.Now()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// The timestamp should be between before and after.
	assertContains(t, content, "anki_synced_at:")
	// Just verify the year is current.
	assertContains(t, content, before.Format("2006"))
	_ = after // just ensuring the timestamp is reasonable
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := atomicWrite(path, []byte("updated"))
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "updated" {
		t.Errorf("expected 'updated', got %q", string(data))
	}
}

func TestUpdateCheckbox_Replace(t *testing.T) {
	input := "# Title\n- [ ] anki_synced\nMore text\n"
	result := updateCheckbox(input)
	assertContains(t, result, "- [x] anki_synced")
	assertNotContains(t, result, "- [ ] anki_synced")
}

func TestUpdateCheckbox_AlreadyChecked(t *testing.T) {
	input := "# Title\n- [x] anki_synced\nMore text\n"
	result := updateCheckbox(input)
	if result != input {
		t.Errorf("expected no changes, got:\n%s", result)
	}
}

func TestUpdateCheckbox_Append(t *testing.T) {
	input := "# Title\nSome text\n"
	result := updateCheckbox(input)
	assertContains(t, result, "- [x] anki_synced")
}

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
	}{
		{"with frontmatter", "---\ntitle: test\n---\nbody\n", "body\n"},
		{"no frontmatter", "just body\n", "just body\n"},
		{"empty body", "---\ntitle: test\n---\n", ""},
		{"unclosed", "---\ntitle: test\n", "---\ntitle: test\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripFrontmatter(tt.input)
			if got != tt.want {
				t.Errorf("stripFrontmatter(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func assertContains(t *testing.T, content, substr string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("expected content to contain %q, got:\n%s", substr, content)
	}
}

func assertNotContains(t *testing.T, content, substr string) {
	t.Helper()
	if strings.Contains(content, substr) {
		t.Errorf("expected content NOT to contain %q, got:\n%s", substr, content)
	}
}

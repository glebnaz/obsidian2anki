package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/glebnaz/obsidian2anki/internal/config"
	"github.com/glebnaz/obsidian2anki/internal/obsidian"
)

func testConfig(t *testing.T) (*config.Config, string) {
	t.Helper()
	dir := t.TempDir()
	vault := filepath.Join(dir, "vault")
	notesDir := filepath.Join(vault, "vocab")
	csvDir := filepath.Join(vault, ".anki_csv")
	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		VaultPath:        vault,
		NotesDir:         notesDir,
		Deck:             "TestDeck",
		Model:            "TestModel",
		CSVDir:           csvDir,
		AnkiEndpoint:     "http://127.0.0.1:1",
		Tags:             []string{"test"},
		RequestTimeoutMs: 100,
		BatchSize:        100,
	}
	return cfg, notesDir
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNewModel(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	if m.cfg != cfg {
		t.Error("expected config to be set")
	}
	if m.dryRun {
		t.Error("expected dryRun to be false")
	}
	if m.mode != modeList {
		t.Error("expected initial mode to be modeList")
	}
}

func TestNewModelDryRun(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, true)
	if !m.dryRun {
		t.Error("expected dryRun to be true")
	}
}

func TestInitReturnsCommand(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected Init to return a command")
	}
}

func TestScanDoneMsg(t *testing.T) {
	cfg, notesDir := testConfig(t)
	writeTestFile(t, notesDir, "words.md", `| Front | Back |
|-------|------|
| hello | hola |
| cat   | gato |
`)

	m := New(cfg, false)
	files, err := obsidian.ScanFiles(notesDir)
	if err != nil {
		t.Fatal(err)
	}

	updated, _ := m.Update(scanDoneMsg{files: files})
	um := updated.(Model)
	if len(um.files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(um.files))
	}
	if um.files[0].CardsCount != 2 {
		t.Errorf("expected 2 cards, got %d", um.files[0].CardsCount)
	}
}

func TestScanDoneMsgError(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)

	updated, _ := m.Update(scanDoneMsg{err: os.ErrNotExist})
	um := updated.(Model)
	if um.err == nil {
		t.Error("expected error to be set")
	}
}

func TestNavigationKeys(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.files = []obsidian.FileInfo{
		{Path: "/a.md", CardsCount: 1},
		{Path: "/b.md", CardsCount: 2},
		{Path: "/c.md", CardsCount: 3},
	}
	m.cursor = 0

	// Move down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	um := updated.(Model)
	if um.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", um.cursor)
	}

	// Move down again
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	um = updated.(Model)
	if um.cursor != 2 {
		t.Errorf("expected cursor 2, got %d", um.cursor)
	}

	// Move down at bottom should not go past end
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	um = updated.(Model)
	if um.cursor != 2 {
		t.Errorf("expected cursor 2 (clamped), got %d", um.cursor)
	}

	// Move up
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	um = updated.(Model)
	if um.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", um.cursor)
	}

	// Move up to top
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	um = updated.(Model)
	if um.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", um.cursor)
	}

	// Move up at top should not go below 0
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	um = updated.(Model)
	if um.cursor != 0 {
		t.Errorf("expected cursor 0 (clamped), got %d", um.cursor)
	}
}

func TestArrowKeys(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.files = []obsidian.FileInfo{
		{Path: "/a.md", CardsCount: 1},
		{Path: "/b.md", CardsCount: 2},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	um := updated.(Model)
	if um.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", um.cursor)
	}

	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyUp})
	um = updated.(Model)
	if um.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", um.cursor)
	}
}

func TestEnterTogglePreview(t *testing.T) {
	cfg, notesDir := testConfig(t)
	writeTestFile(t, notesDir, "words.md", `| Front | Back |
|-------|------|
| hello | hola |
`)

	m := New(cfg, false)
	files, _ := obsidian.ScanFiles(notesDir)
	m.files = files

	// Enter to preview
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um := updated.(Model)
	if um.mode != modePreview {
		t.Error("expected mode to be preview")
	}
	if len(um.preview) != 1 {
		t.Errorf("expected 1 preview card, got %d", len(um.preview))
	}

	// Enter to go back to list
	updated, _ = um.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um = updated.(Model)
	if um.mode != modeList {
		t.Error("expected mode to be list")
	}
	if um.preview != nil {
		t.Error("expected preview to be nil")
	}
}

func TestQuitKey(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.files = []obsidian.FileInfo{}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	// Execute the command to check it produces a quit message
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

func TestCtrlCQuit(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.files = []obsidian.FileInfo{}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestViewShowsLoadingBeforeScan(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	// files is nil (no scan done yet)

	view := m.View()
	if !strings.Contains(view, "Scanning") {
		t.Errorf("expected 'Scanning' in view, got: %s", view)
	}
}

func TestViewShowsError(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.err = os.ErrNotExist

	view := m.View()
	if !strings.Contains(view, "Error") {
		t.Errorf("expected 'Error' in view, got: %s", view)
	}
}

func TestViewShowsEmptyList(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.files = []obsidian.FileInfo{}

	view := m.View()
	if !strings.Contains(view, "No markdown files found") {
		t.Errorf("expected empty message, got: %s", view)
	}
}

func TestViewShowsFileList(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.files = []obsidian.FileInfo{
		{Path: "/vault/vocab/words.md", Synced: false, HasTable: true, CardsCount: 5},
		{Path: "/vault/vocab/done.md", Synced: true, HasTable: true, CardsCount: 3},
	}

	view := m.View()
	if !strings.Contains(view, "words.md") {
		t.Errorf("expected 'words.md' in view, got: %s", view)
	}
	if !strings.Contains(view, "done.md") {
		t.Errorf("expected 'done.md' in view, got: %s", view)
	}
	if !strings.Contains(view, "5 cards") {
		t.Errorf("expected '5 cards' in view, got: %s", view)
	}
}

func TestViewShowsDryRunIndicator(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, true)
	m.files = []obsidian.FileInfo{}

	view := m.View()
	if !strings.Contains(view, "dry-run") {
		t.Errorf("expected 'dry-run' in view, got: %s", view)
	}
}

func TestViewShowsHelpInListMode(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.files = []obsidian.FileInfo{}

	view := m.View()
	if !strings.Contains(view, "sync all") {
		t.Errorf("expected list help text, got: %s", view)
	}
}

func TestViewShowsHelpInPreviewMode(t *testing.T) {
	cfg, notesDir := testConfig(t)
	writeTestFile(t, notesDir, "a.md", `| Front | Back |
|-------|------|
| hi | bye |
`)

	m := New(cfg, false)
	files, _ := obsidian.ScanFiles(notesDir)
	m.files = files
	m.mode = modePreview
	m.loadPreview()

	view := m.View()
	if !strings.Contains(view, "back to list") {
		t.Errorf("expected preview help text, got: %s", view)
	}
}

func TestPreviewShowsCards(t *testing.T) {
	cfg, notesDir := testConfig(t)
	writeTestFile(t, notesDir, "words.md", `| Front | Back |
|-------|------|
| hello | hola |
| cat   | gato |
`)

	m := New(cfg, false)
	files, _ := obsidian.ScanFiles(notesDir)
	m.files = files
	m.mode = modePreview
	m.loadPreview()

	view := m.View()
	if !strings.Contains(view, "hello") {
		t.Errorf("expected 'hello' in preview, got: %s", view)
	}
	if !strings.Contains(view, "hola") {
		t.Errorf("expected 'hola' in preview, got: %s", view)
	}
	if !strings.Contains(view, "2 of 2") {
		t.Errorf("expected '2 of 2' in preview, got: %s", view)
	}
}

func TestPreviewTruncatesLongList(t *testing.T) {
	cfg, notesDir := testConfig(t)
	var rows strings.Builder
	rows.WriteString("| Front | Back |\n|-------|------|\n")
	for i := 0; i < 15; i++ {
		rows.WriteString("| word" + strings.Repeat("x", i) + " | def |\n")
	}
	writeTestFile(t, notesDir, "many.md", rows.String())

	m := New(cfg, false)
	files, _ := obsidian.ScanFiles(notesDir)
	m.files = files
	m.mode = modePreview
	m.loadPreview()

	view := m.View()
	if !strings.Contains(view, "10 of 15") {
		t.Errorf("expected '10 of 15' in preview, got: %s", view)
	}
	if !strings.Contains(view, "5 more") {
		t.Errorf("expected '5 more' in preview, got: %s", view)
	}
}

func TestNavigationDoesNotMoveInPreviewMode(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.files = []obsidian.FileInfo{
		{Path: "/a.md"},
		{Path: "/b.md"},
	}
	m.mode = modePreview
	m.cursor = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	um := updated.(Model)
	if um.cursor != 0 {
		t.Errorf("expected cursor to stay at 0 in preview mode, got %d", um.cursor)
	}
}

func TestSyncFileMsgUpdatesLog(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.files = []obsidian.FileInfo{}

	updated, _ := m.Update(syncFileMsg{path: "/words.md", err: nil})
	um := updated.(Model)
	if len(um.syncLog) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(um.syncLog))
	}
	if !strings.Contains(um.syncLog[0], "OK") {
		t.Errorf("expected 'OK' in log, got: %s", um.syncLog[0])
	}
}

func TestSyncFileMsgErrorUpdatesLog(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.files = []obsidian.FileInfo{}

	updated, _ := m.Update(syncFileMsg{path: "/words.md", err: os.ErrPermission})
	um := updated.(Model)
	if len(um.syncLog) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(um.syncLog))
	}
	if !strings.Contains(um.syncLog[0], "FAIL") {
		t.Errorf("expected 'FAIL' in log, got: %s", um.syncLog[0])
	}
}

func TestSyncAllDoneMsg(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.syncing = true

	updated, cmd := m.Update(syncAllDoneMsg{})
	um := updated.(Model)
	if um.syncing {
		t.Error("expected syncing to be false after syncAllDoneMsg")
	}
	if um.syncStatus != "sync complete" {
		t.Errorf("expected 'sync complete', got: %s", um.syncStatus)
	}
	if cmd == nil {
		t.Error("expected rescan command after sync all done")
	}
}

func TestWindowSizeMsg(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	um := updated.(Model)
	if um.width != 80 {
		t.Errorf("expected width 80, got %d", um.width)
	}
	if um.height != 24 {
		t.Errorf("expected height 24, got %d", um.height)
	}
}

func TestSyncBlocksKeysDuringSyncing(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.files = []obsidian.FileInfo{
		{Path: "/a.md"},
		{Path: "/b.md"},
	}
	m.syncing = true
	m.cursor = 0

	// Navigation should be blocked
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	um := updated.(Model)
	if um.cursor != 0 {
		t.Errorf("expected cursor to stay at 0 during sync, got %d", um.cursor)
	}

	// Sync should be blocked
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd != nil {
		t.Error("expected no command during syncing")
	}
}

func TestRescanKey(t *testing.T) {
	cfg, notesDir := testConfig(t)
	writeTestFile(t, notesDir, "words.md", `| Front | Back |
|-------|------|
| hello | hola |
`)

	m := New(cfg, false)
	m.files = []obsidian.FileInfo{}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	um := updated.(Model)
	if um.syncStatus != "rescanning..." {
		t.Errorf("expected 'rescanning...' status, got: %s", um.syncStatus)
	}
	if cmd == nil {
		t.Error("expected rescan command")
	}
}

func TestSyncSelectedSkipsSynced(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.files = []obsidian.FileInfo{
		{Path: "/a.md", Synced: true, CardsCount: 5},
	}
	m.cursor = 0

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd != nil {
		t.Error("expected no command when syncing already-synced file")
	}
}

func TestViewShowsSyncProgress(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.files = []obsidian.FileInfo{}
	m.syncStatus = "syncing all files..."

	view := m.View()
	if !strings.Contains(view, "syncing all files...") {
		t.Errorf("expected sync status in view, got: %s", view)
	}
}

func TestViewShowsSyncLog(t *testing.T) {
	cfg, _ := testConfig(t)
	m := New(cfg, false)
	m.files = []obsidian.FileInfo{}
	m.syncLog = []string{"OK   words.md", "FAIL bad.md: some error"}

	view := m.View()
	if !strings.Contains(view, "OK   words.md") {
		t.Errorf("expected sync log in view, got: %s", view)
	}
	if !strings.Contains(view, "FAIL bad.md") {
		t.Errorf("expected sync log in view, got: %s", view)
	}
}

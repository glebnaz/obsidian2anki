package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/glebnaz/obsidian2anki/internal/anki"
	"github.com/glebnaz/obsidian2anki/internal/config"
	"github.com/glebnaz/obsidian2anki/internal/csvout"
	"github.com/glebnaz/obsidian2anki/internal/obsidian"
	"github.com/glebnaz/obsidian2anki/internal/parser"
)

const previewMaxCards = 10

// view modes
const (
	modeList    = iota
	modePreview
)

// Styles
var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	syncedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("34"))
	unsyncedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	progressStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("45"))
)

// Messages

type scanDoneMsg struct {
	files []obsidian.FileInfo
	err   error
}

type syncFileMsg struct {
	path string
	err  error
}

type syncAllDoneMsg struct{}

// Model is the Bubble Tea model for the TUI.
type Model struct {
	cfg    *config.Config
	dryRun bool

	files    []obsidian.FileInfo
	cursor   int
	mode     int
	preview  []parser.Card
	previewW []parser.Warning

	syncing    bool
	syncStatus string
	syncLog    []string

	err    error
	width  int
	height int
}

// New creates a new TUI model.
func New(cfg *config.Config, dryRun bool) Model {
	return Model{
		cfg:    cfg,
		dryRun: dryRun,
	}
}

// Init performs the initial scan.
func (m Model) Init() tea.Cmd {
	return scanCmd(m.cfg.NotesDir)
}

func scanCmd(notesDir string) tea.Cmd {
	return func() tea.Msg {
		files, err := obsidian.ScanFiles(notesDir)
		return scanDoneMsg{files: files, err: err}
	}
}

func syncSingleFile(cfg *config.Config, f obsidian.FileInfo, dryRun bool) error {
	data, err := os.ReadFile(f.Path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	parsed := parser.ParseTable(string(data))
	if len(parsed.Cards) == 0 {
		return fmt.Errorf("no cards found")
	}

	baseName := strings.TrimSuffix(filepath.Base(f.Path), filepath.Ext(f.Path))
	now := time.Now()

	if _, err := csvout.Export(parsed.Cards, cfg.CSVDir, baseName, now); err != nil {
		return fmt.Errorf("csv export: %w", err)
	}

	if dryRun {
		return nil
	}

	client := anki.NewClient(cfg.AnkiEndpoint, cfg.RequestTimeoutMs)

	notes := make([]anki.Note, len(parsed.Cards))
	for i, c := range parsed.Cards {
		notes[i] = anki.Note{
			DeckName:  cfg.Deck,
			ModelName: cfg.Model,
			Front:     c.Front,
			Back:      c.Back,
			Tags:      cfg.Tags,
			AllowDup:  cfg.AllowDuplicates,
		}
	}

	if err := client.AddNotes(notes, cfg.BatchSize); err != nil {
		return fmt.Errorf("import: %w", err)
	}

	if err := obsidian.MarkSynced(f.Path, obsidian.MarkSyncedOptions{
		Deck:         cfg.Deck,
		Model:        cfg.Model,
		MarkCheckbox: cfg.MarkCheckbox,
		Now:          now,
	}); err != nil {
		return fmt.Errorf("mark synced: %w", err)
	}

	return nil
}

// syncAllCmd ensures deck/model exist then syncs all unsynced files sequentially.
func (m Model) syncAllCmd() tea.Cmd {
	cfg := m.cfg
	files := m.files
	dryRun := m.dryRun
	return func() tea.Msg {
		if !dryRun {
			client := anki.NewClient(cfg.AnkiEndpoint, cfg.RequestTimeoutMs)
			if _, err := client.Version(); err != nil {
				return syncFileMsg{path: "", err: fmt.Errorf("AnkiConnect unreachable: %w", err)}
			}
			if err := client.EnsureDeck(cfg.Deck); err != nil {
				return syncFileMsg{path: "", err: fmt.Errorf("ensure deck: %w", err)}
			}
			if err := client.EnsureModel(cfg.Model); err != nil {
				return syncFileMsg{path: "", err: fmt.Errorf("ensure model: %w", err)}
			}
		}

		for _, f := range files {
			if f.Synced || f.CardsCount == 0 {
				continue
			}
			if err := syncSingleFile(cfg, f, dryRun); err != nil {
				return syncFileMsg{path: f.Path, err: err}
			}
		}
		return syncAllDoneMsg{}
	}
}

// syncSelectedCmd ensures deck/model exist then syncs the selected file.
func (m Model) syncSelectedCmd() tea.Cmd {
	if m.cursor >= len(m.files) {
		return nil
	}
	f := m.files[m.cursor]
	if f.Synced {
		return nil
	}
	cfg := m.cfg
	dryRun := m.dryRun
	return func() tea.Msg {
		if !dryRun {
			client := anki.NewClient(cfg.AnkiEndpoint, cfg.RequestTimeoutMs)
			if _, err := client.Version(); err != nil {
				return syncFileMsg{path: f.Path, err: fmt.Errorf("AnkiConnect unreachable: %w", err)}
			}
			if err := client.EnsureDeck(cfg.Deck); err != nil {
				return syncFileMsg{path: f.Path, err: fmt.Errorf("ensure deck: %w", err)}
			}
			if err := client.EnsureModel(cfg.Model); err != nil {
				return syncFileMsg{path: f.Path, err: fmt.Errorf("ensure model: %w", err)}
			}
		}
		err := syncSingleFile(cfg, f, dryRun)
		return syncFileMsg{path: f.Path, err: err}
	}
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case scanDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.files = msg.files
		m.cursor = 0
		m.syncing = false
		m.syncStatus = ""
		return m, nil

	case syncFileMsg:
		if msg.err != nil {
			m.syncLog = append(m.syncLog, fmt.Sprintf("FAIL %s: %v", filepath.Base(msg.path), msg.err))
		} else if msg.path != "" {
			m.syncLog = append(m.syncLog, fmt.Sprintf("OK   %s", filepath.Base(msg.path)))
		}
		return m, nil

	case syncAllDoneMsg:
		m.syncing = false
		m.syncStatus = "sync complete"
		return m, scanCmd(m.cfg.NotesDir)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.syncing {
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.mode == modeList && m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j":
		if m.mode == modeList && m.cursor < len(m.files)-1 {
			m.cursor++
		}
		return m, nil

	case "enter":
		if m.mode == modePreview {
			m.mode = modeList
			m.preview = nil
			m.previewW = nil
			return m, nil
		}
		if m.cursor < len(m.files) {
			m.mode = modePreview
			m.loadPreview()
		}
		return m, nil

	case "s":
		if m.mode == modeList && m.cursor < len(m.files) && !m.files[m.cursor].Synced {
			m.syncing = true
			m.syncStatus = "syncing " + filepath.Base(m.files[m.cursor].Path) + "..."
			m.syncLog = nil
			return m, tea.Batch(m.syncSelectedCmd(), func() tea.Msg {
				return syncAllDoneMsg{}
			})
		}
		return m, nil

	case "a":
		if m.mode == modeList {
			m.syncing = true
			m.syncStatus = "syncing all files..."
			m.syncLog = nil
			return m, m.syncAllCmd()
		}
		return m, nil

	case "r":
		if m.mode == modeList {
			m.syncStatus = "rescanning..."
			return m, scanCmd(m.cfg.NotesDir)
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) loadPreview() {
	if m.cursor >= len(m.files) {
		return
	}
	data, err := os.ReadFile(m.files[m.cursor].Path)
	if err != nil {
		m.preview = nil
		return
	}
	result := parser.ParseTable(string(data))
	m.preview = result.Cards
	m.previewW = result.Warnings
}

// View renders the TUI.
func (m Model) View() string {
	if m.err != nil {
		return errorStyle.Render("Error: "+m.err.Error()) + "\n\nPress q to quit.\n"
	}

	if m.files == nil {
		return "Scanning files...\n"
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("obs2anki TUI"))
	if m.dryRun {
		b.WriteString("  (dry-run)")
	}
	b.WriteString("\n\n")

	if m.mode == modePreview {
		m.renderPreview(&b)
	} else {
		m.renderList(&b)
	}

	if m.syncStatus != "" {
		b.WriteString("\n")
		b.WriteString(progressStyle.Render(m.syncStatus))
		b.WriteString("\n")
	}

	for _, l := range m.syncLog {
		b.WriteString(l)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.mode == modePreview {
		b.WriteString(helpStyle.Render("enter: back to list | q: quit"))
	} else {
		b.WriteString(helpStyle.Render("j/k: navigate | enter: preview | s: sync selected | a: sync all | r: rescan | q: quit"))
	}
	b.WriteString("\n")

	return b.String()
}

func (m Model) renderList(b *strings.Builder) {
	if len(m.files) == 0 {
		b.WriteString("No markdown files found.\n")
		return
	}

	for i, f := range m.files {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		name := filepath.Base(f.Path)
		status := unsyncedStyle.Render("[ ]")
		if f.Synced {
			status = syncedStyle.Render("[x]")
		}

		cards := fmt.Sprintf("%d cards", f.CardsCount)

		line := fmt.Sprintf("%s%s %s  %s", cursor, status, name, cards)
		if i == m.cursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func (m Model) renderPreview(b *strings.Builder) {
	if m.cursor >= len(m.files) {
		return
	}

	name := filepath.Base(m.files[m.cursor].Path)
	b.WriteString(titleStyle.Render("Preview: " + name))
	b.WriteString("\n\n")

	if len(m.preview) == 0 {
		b.WriteString("No cards found in this file.\n")
		return
	}

	limit := previewMaxCards
	if limit > len(m.preview) {
		limit = len(m.preview)
	}

	fmt.Fprintf(b, "Showing %d of %d cards:\n\n", limit, len(m.preview))

	for i := 0; i < limit; i++ {
		c := m.preview[i]
		fmt.Fprintf(b, "  %d. %s  ->  %s\n", i+1, c.Front, c.Back)
	}

	if len(m.preview) > previewMaxCards {
		fmt.Fprintf(b, "\n  ... and %d more\n", len(m.preview)-previewMaxCards)
	}

	if len(m.previewW) > 0 {
		fmt.Fprintf(b, "\n  %d warning(s)\n", len(m.previewW))
	}
}

// Run starts the Bubble Tea program.
func Run(cfg *config.Config, dryRun bool) error {
	m := New(cfg, dryRun)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

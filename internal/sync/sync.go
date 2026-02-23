package sync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebnaz/obsidian2anki/internal/anki"
	"github.com/glebnaz/obsidian2anki/internal/config"
	"github.com/glebnaz/obsidian2anki/internal/csvout"
	"github.com/glebnaz/obsidian2anki/internal/obsidian"
	"github.com/glebnaz/obsidian2anki/internal/parser"
)

// Result holds the outcome of a sync run.
type Result struct {
	Synced  int
	Skipped int
	Failed  int
}

// Run executes the sync pipeline:
// 1. Verify AnkiConnect is reachable
// 2. Ensure deck and model exist
// 3. For each unsynced file: parse cards, export CSV, import to Anki, mark synced
//
// If dryRun is true, CSV is exported but cards are not imported and files are not marked.
// Returns ExitPartial (2) if any file failed but others succeeded, ExitSuccess (0) otherwise.
func Run(cfg *config.Config, dryRun, verbose bool, out io.Writer) (int, error) {
	client := anki.NewClient(cfg.AnkiEndpoint, cfg.RequestTimeoutMs)

	// Step 1: Verify AnkiConnect is reachable.
	if !dryRun {
		ver, err := client.Version()
		if err != nil {
			return 0, fmt.Errorf("cannot connect to AnkiConnect: %w", err)
		}
		if verbose {
			_, _ = fmt.Fprintf(out, "AnkiConnect version: %d\n", ver)
		}
	}

	// Step 2: Ensure deck and model exist.
	if !dryRun {
		if err := client.EnsureDeck(cfg.Deck); err != nil {
			return 0, fmt.Errorf("ensure deck %q: %w", cfg.Deck, err)
		}
		if err := client.EnsureModel(cfg.Model); err != nil {
			return 0, fmt.Errorf("ensure model %q: %w", cfg.Model, err)
		}
		if verbose {
			_, _ = fmt.Fprintf(out, "deck %q and model %q ready\n", cfg.Deck, cfg.Model)
		}
	}

	// Step 3: Scan files.
	files, err := obsidian.ScanFiles(cfg.NotesDir)
	if err != nil {
		return 0, fmt.Errorf("scanning files: %w", err)
	}

	var result Result
	now := time.Now()

	for _, f := range files {
		if f.Synced {
			result.Skipped++
			if verbose {
				_, _ = fmt.Fprintf(out, "skip (synced): %s\n", f.Path)
			}
			continue
		}

		// Read file content and parse cards.
		data, err := os.ReadFile(f.Path)
		if err != nil {
			_, _ = fmt.Fprintf(out, "error reading %s: %v\n", f.Path, err)
			result.Failed++
			continue
		}

		parsed := parser.ParseTable(string(data))
		if len(parsed.Cards) == 0 {
			result.Skipped++
			if verbose {
				_, _ = fmt.Fprintf(out, "skip (no cards): %s\n", f.Path)
			}
			continue
		}

		if verbose {
			for _, w := range parsed.Warnings {
				_, _ = fmt.Fprintf(out, "warning: %s line %d: %s\n", f.Path, w.Line, w.Message)
			}
		}

		// Export CSV.
		baseName := strings.TrimSuffix(filepath.Base(f.Path), filepath.Ext(f.Path))
		csvPath, err := csvout.Export(parsed.Cards, cfg.CSVDir, baseName, now)
		if err != nil {
			_, _ = fmt.Fprintf(out, "error exporting CSV for %s: %v\n", f.Path, err)
			result.Failed++
			continue
		}
		if verbose {
			_, _ = fmt.Fprintf(out, "exported: %s\n", csvPath)
		}

		if dryRun {
			_, _ = fmt.Fprintf(out, "dry-run: %s (%d cards)\n", f.Path, len(parsed.Cards))
			result.Synced++
			continue
		}

		// Import cards to Anki.
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
			_, _ = fmt.Fprintf(out, "error importing %s: %v\n", f.Path, err)
			result.Failed++
			continue
		}

		// Mark file as synced.
		if err := obsidian.MarkSynced(f.Path, obsidian.MarkSyncedOptions{
			Deck:  cfg.Deck,
			Model: cfg.Model,
			Now:   now,
		}); err != nil {
			_, _ = fmt.Fprintf(out, "error marking %s as synced: %v\n", f.Path, err)
			result.Failed++
			continue
		}

		_, _ = fmt.Fprintf(out, "synced: %s (%d cards)\n", f.Path, len(parsed.Cards))
		result.Synced++
	}

	if verbose {
		_, _ = fmt.Fprintf(out, "done: synced=%d skipped=%d failed=%d\n", result.Synced, result.Skipped, result.Failed)
	}

	if result.Failed > 0 {
		return 2, nil
	}
	return 0, nil
}

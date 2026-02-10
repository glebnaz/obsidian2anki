package sync

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebnaz/obsidian2anki/internal/config"
)

// ankiHandler is a test HTTP handler that responds to AnkiConnect requests.
type ankiHandler struct {
	decks  []string
	models []string
}

func mustEncode(w http.ResponseWriter, v any) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "encode error", 500)
	}
}

func (h *ankiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string          `json:"action"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch req.Action {
	case "version":
		mustEncode(w, map[string]any{"result": 6, "error": nil})
	case "deckNames":
		mustEncode(w, map[string]any{"result": h.decks, "error": nil})
	case "createDeck":
		var p struct {
			Deck string `json:"deck"`
		}
		_ = json.Unmarshal(req.Params, &p)
		h.decks = append(h.decks, p.Deck)
		mustEncode(w, map[string]any{"result": 12345, "error": nil})
	case "modelNames":
		mustEncode(w, map[string]any{"result": h.models, "error": nil})
	case "createModel":
		var p struct {
			ModelName string `json:"modelName"`
		}
		_ = json.Unmarshal(req.Params, &p)
		h.models = append(h.models, p.ModelName)
		mustEncode(w, map[string]any{"result": map[string]any{"id": 67890}, "error": nil})
	case "addNotes":
		var p struct {
			Notes []json.RawMessage `json:"notes"`
		}
		_ = json.Unmarshal(req.Params, &p)
		ids := make([]*int64, len(p.Notes))
		for i := range p.Notes {
			id := int64(1000 + i)
			ids[i] = &id
		}
		mustEncode(w, map[string]any{"result": ids, "error": nil})
	default:
		mustEncode(w, map[string]any{"result": nil, "error": "unsupported action"})
	}
}

func setupTestEnv(t *testing.T, handler http.Handler) (*config.Config, string) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

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
		AnkiEndpoint:     srv.URL,
		Tags:             []string{"test"},
		AllowDuplicates:  false,
		MarkCheckbox:     false,
		RequestTimeoutMs: 5000,
		BatchSize:        100,
	}

	return cfg, notesDir
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSyncBasicFlow(t *testing.T) {
	h := &ankiHandler{decks: []string{"Default"}, models: []string{"Basic"}}
	cfg, notesDir := setupTestEnv(t, h)

	writeFile(t, notesDir, "words.md", `---
title: Words
---

| Front | Back |
|-------|------|
| hello | hola |
| cat   | gato |
`)

	var buf bytes.Buffer
	code, err := Run(cfg, false, false, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	output := buf.String()
	if !strings.Contains(output, "synced:") {
		t.Errorf("expected 'synced:' in output, got: %s", output)
	}

	// Verify deck and model were created (not in initial lists).
	found := false
	for _, d := range h.decks {
		if d == "TestDeck" {
			found = true
		}
	}
	if !found {
		t.Error("expected TestDeck to be created")
	}

	// Verify file was marked as synced.
	data, err := os.ReadFile(filepath.Join(notesDir, "words.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "anki_synced: true") {
		t.Error("expected file to be marked as synced")
	}
}

func TestSyncSkipsSyncedFiles(t *testing.T) {
	h := &ankiHandler{decks: []string{"TestDeck"}, models: []string{"TestModel"}}
	cfg, notesDir := setupTestEnv(t, h)

	writeFile(t, notesDir, "already.md", `---
anki_synced: true
---

| Front | Back |
|-------|------|
| hello | hola |
`)

	var buf bytes.Buffer
	code, err := Run(cfg, false, true, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	output := buf.String()
	if !strings.Contains(output, "skip (synced)") {
		t.Errorf("expected 'skip (synced)' in verbose output, got: %s", output)
	}
}

func TestSyncSkipsNoCards(t *testing.T) {
	h := &ankiHandler{decks: []string{"TestDeck"}, models: []string{"TestModel"}}
	cfg, notesDir := setupTestEnv(t, h)

	writeFile(t, notesDir, "empty.md", `---
title: No Table
---

Just some text without a table.
`)

	var buf bytes.Buffer
	code, err := Run(cfg, false, true, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	output := buf.String()
	if !strings.Contains(output, "skip (no cards)") {
		t.Errorf("expected 'skip (no cards)' in verbose output, got: %s", output)
	}
}

func TestSyncDryRun(t *testing.T) {
	h := &ankiHandler{decks: []string{"Default"}, models: []string{"Basic"}}
	cfg, notesDir := setupTestEnv(t, h)

	writeFile(t, notesDir, "words.md", `| Front | Back |
|-------|------|
| hello | hola |
`)

	var buf bytes.Buffer
	code, err := Run(cfg, true, false, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	output := buf.String()
	if !strings.Contains(output, "dry-run:") {
		t.Errorf("expected 'dry-run:' in output, got: %s", output)
	}

	// Verify file was NOT marked as synced.
	data, err := os.ReadFile(filepath.Join(notesDir, "words.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "anki_synced") {
		t.Error("dry-run should not mark file as synced")
	}
}

func TestSyncDryRunSkipsAnkiConnect(t *testing.T) {
	// Use a server that always fails - dry-run should never call it.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", 500)
	}))
	defer srv.Close()

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
		AnkiEndpoint:     srv.URL,
		Tags:             []string{"test"},
		RequestTimeoutMs: 5000,
		BatchSize:        100,
	}

	writeFile(t, notesDir, "words.md", `| Front | Back |
|-------|------|
| hello | hola |
`)

	var buf bytes.Buffer
	code, err := Run(cfg, true, false, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestSyncAnkiConnectUnreachable(t *testing.T) {
	cfg := &config.Config{
		VaultPath:        "/tmp",
		NotesDir:         "/tmp",
		Deck:             "Test",
		Model:            "Test",
		CSVDir:           "/tmp",
		AnkiEndpoint:     "http://127.0.0.1:1", // unreachable
		Tags:             []string{"test"},
		RequestTimeoutMs: 100,
		BatchSize:        100,
	}

	var buf bytes.Buffer
	_, err := Run(cfg, false, false, &buf)
	if err == nil {
		t.Fatal("expected error when AnkiConnect is unreachable")
	}
	if !strings.Contains(err.Error(), "cannot connect to AnkiConnect") {
		t.Errorf("expected connection error, got: %v", err)
	}
}

func TestSyncPartialFailure(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Action string          `json:"action"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		switch req.Action {
		case "version":
			mustEncode(w, map[string]any{"result": 6, "error": nil})
		case "deckNames":
			mustEncode(w, map[string]any{"result": []string{"TestDeck"}, "error": nil})
		case "modelNames":
			mustEncode(w, map[string]any{"result": []string{"TestModel"}, "error": nil})
		case "addNotes":
			callCount++
			if callCount == 1 {
				// First file succeeds.
				var p struct {
					Notes []json.RawMessage `json:"notes"`
				}
				_ = json.Unmarshal(req.Params, &p)
				ids := make([]*int64, len(p.Notes))
				for i := range p.Notes {
					id := int64(1000 + i)
					ids[i] = &id
				}
				mustEncode(w, map[string]any{"result": ids, "error": nil})
			} else {
				// Second file fails (null in result).
				mustEncode(w, map[string]any{"result": []*int64{nil}, "error": nil})
			}
		}
	}))
	defer srv.Close()

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
		AnkiEndpoint:     srv.URL,
		Tags:             []string{"test"},
		RequestTimeoutMs: 5000,
		BatchSize:        100,
	}

	writeFile(t, notesDir, "a_good.md", `| Front | Back |
|-------|------|
| hello | hola |
`)
	writeFile(t, notesDir, "b_bad.md", `| Front | Back |
|-------|------|
| world | mundo |
`)

	var buf bytes.Buffer
	code, err := Run(cfg, false, false, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 2 {
		t.Errorf("expected exit code 2 for partial failure, got %d", code)
	}
}

func TestSyncMultipleFiles(t *testing.T) {
	h := &ankiHandler{decks: []string{"TestDeck"}, models: []string{"TestModel"}}
	cfg, notesDir := setupTestEnv(t, h)

	writeFile(t, notesDir, "a_words.md", `| Front | Back |
|-------|------|
| hello | hola |
`)
	writeFile(t, notesDir, "b_words.md", `| Front | Back |
|-------|------|
| cat   | gato |
| dog   | perro |
`)

	var buf bytes.Buffer
	code, err := Run(cfg, false, false, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	output := buf.String()
	if strings.Count(output, "synced:") != 2 {
		t.Errorf("expected 2 synced lines, got output: %s", output)
	}

	// Both files should be marked as synced.
	for _, name := range []string{"a_words.md", "b_words.md"} {
		data, err := os.ReadFile(filepath.Join(notesDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "anki_synced: true") {
			t.Errorf("expected %s to be marked as synced", name)
		}
	}
}

func TestSyncVerboseOutput(t *testing.T) {
	h := &ankiHandler{decks: []string{"Default"}, models: []string{"Basic"}}
	cfg, notesDir := setupTestEnv(t, h)

	writeFile(t, notesDir, "words.md", `| Front | Back |
|-------|------|
| hello | hola |
`)

	var buf bytes.Buffer
	code, err := Run(cfg, false, true, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	output := buf.String()
	if !strings.Contains(output, "AnkiConnect version:") {
		t.Errorf("expected verbose output to contain AnkiConnect version, got: %s", output)
	}
	if !strings.Contains(output, "done:") {
		t.Errorf("expected verbose output to contain done summary, got: %s", output)
	}
}

func TestSyncCSVExported(t *testing.T) {
	h := &ankiHandler{decks: []string{"TestDeck"}, models: []string{"TestModel"}}
	cfg, notesDir := setupTestEnv(t, h)

	writeFile(t, notesDir, "vocab.md", `| Front | Back |
|-------|------|
| hello | hola |
`)

	var buf bytes.Buffer
	code, err := Run(cfg, false, true, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	// Check that CSV file was created.
	csvFiles, err := filepath.Glob(filepath.Join(cfg.CSVDir, "vocab-*.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(csvFiles) == 0 {
		t.Error("expected CSV file to be created")
	}
}

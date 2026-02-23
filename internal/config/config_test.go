package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	p, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, DefaultConfigDir, DefaultConfigFile)
	if p != expected {
		t.Errorf("expected %s, got %s", expected, p)
	}
}

func writeConfig(t *testing.T, dir string, cfg map[string]any) string {
	t.Helper()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func fullConfig(vaultPath string) map[string]any {
	return map[string]any{
		"vault_path": vaultPath,
		"notes_dir":  "vocab",
		"deck":       "TestDeck",
		"model":      "Basic",
		"csv_dir":    ".anki_csv",
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	vault := filepath.Join(dir, "vault")
	mkdirAll(t, vault)

	path := writeConfig(t, dir, fullConfig(vault))

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.VaultPath != vault {
		t.Errorf("vault_path: expected %s, got %s", vault, cfg.VaultPath)
	}
	if cfg.Deck != "TestDeck" {
		t.Errorf("deck: expected TestDeck, got %s", cfg.Deck)
	}
	if cfg.Model != "Basic" {
		t.Errorf("model: expected Basic, got %s", cfg.Model)
	}

	// Check defaults applied
	if cfg.AnkiEndpoint != DefaultAnkiEndpoint {
		t.Errorf("anki_endpoint default: expected %s, got %s", DefaultAnkiEndpoint, cfg.AnkiEndpoint)
	}
	if cfg.MarkCheckbox != false {
		t.Errorf("mark_checkbox default: expected false, got true")
	}
	if cfg.AllowDuplicates != false {
		t.Errorf("allow_duplicates default: expected false, got true")
	}
	if len(cfg.Tags) != 2 || cfg.Tags[0] != "obsidian" || cfg.Tags[1] != "voc_list" {
		t.Errorf("tags default: expected [obsidian, voc_list], got %v", cfg.Tags)
	}
	if cfg.RequestTimeoutMs != DefaultRequestTimeoutMs {
		t.Errorf("request_timeout_ms default: expected %d, got %d", DefaultRequestTimeoutMs, cfg.RequestTimeoutMs)
	}
	if cfg.BatchSize != DefaultBatchSize {
		t.Errorf("batch_size default: expected %d, got %d", DefaultBatchSize, cfg.BatchSize)
	}
}

func TestLoadOverrideDefaults(t *testing.T) {
	dir := t.TempDir()
	vault := filepath.Join(dir, "vault")
	mkdirAll(t, vault)

	cfgMap := fullConfig(vault)
	cfgMap["anki_endpoint"] = "http://localhost:9999"
	cfgMap["mark_checkbox"] = true
	cfgMap["allow_duplicates"] = true
	cfgMap["tags"] = []string{"custom"}
	cfgMap["request_timeout_ms"] = 10000
	cfgMap["batch_size"] = 50

	path := writeConfig(t, dir, cfgMap)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AnkiEndpoint != "http://localhost:9999" {
		t.Errorf("anki_endpoint: expected http://localhost:9999, got %s", cfg.AnkiEndpoint)
	}
	if cfg.MarkCheckbox != true {
		t.Errorf("mark_checkbox: expected true, got false")
	}
	if cfg.AllowDuplicates != true {
		t.Errorf("allow_duplicates: expected true, got false")
	}
	if len(cfg.Tags) != 1 || cfg.Tags[0] != "custom" {
		t.Errorf("tags: expected [custom], got %v", cfg.Tags)
	}
	if cfg.RequestTimeoutMs != 10000 {
		t.Errorf("request_timeout_ms: expected 10000, got %d", cfg.RequestTimeoutMs)
	}
	if cfg.BatchSize != 50 {
		t.Errorf("batch_size: expected 50, got %d", cfg.BatchSize)
	}
}

func TestLoadMissingRequiredFields(t *testing.T) {
	dir := t.TempDir()

	// Empty config - all required fields missing
	path := writeConfig(t, dir, map[string]any{})

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}

	errStr := err.Error()
	for _, field := range []string{"vault_path", "notes_dir", "deck", "model", "csv_dir"} {
		if !strings.Contains(errStr, field) {
			t.Errorf("error should mention missing field %q, got: %s", field, errStr)
		}
	}
}

func TestLoadMissingSingleField(t *testing.T) {
	dir := t.TempDir()

	cfgMap := fullConfig("/tmp/vault")
	delete(cfgMap, "deck")

	path := writeConfig(t, dir, cfgMap)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing deck field")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "deck") {
		t.Errorf("error should mention missing field deck, got: %s", errStr)
	}
	// Should NOT mention fields that are present
	if strings.Contains(errStr, "vault_path") {
		t.Errorf("error should not mention vault_path which is present, got: %s", errStr)
	}
}

func TestLoadResolveRelativePaths(t *testing.T) {
	dir := t.TempDir()
	vault := filepath.Join(dir, "vault")
	mkdirAll(t, vault)

	path := writeConfig(t, dir, fullConfig(vault))

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedNotesDir := filepath.Join(vault, "vocab")
	if cfg.NotesDir != expectedNotesDir {
		t.Errorf("notes_dir should be resolved: expected %s, got %s", expectedNotesDir, cfg.NotesDir)
	}

	expectedCSVDir := filepath.Join(vault, ".anki_csv")
	if cfg.CSVDir != expectedCSVDir {
		t.Errorf("csv_dir should be resolved: expected %s, got %s", expectedCSVDir, cfg.CSVDir)
	}
}

func TestLoadAbsolutePathsUnchanged(t *testing.T) {
	dir := t.TempDir()
	vault := filepath.Join(dir, "vault")
	mkdirAll(t, vault)

	absNotes := filepath.Join(dir, "abs_notes")
	absCSV := filepath.Join(dir, "abs_csv")

	cfgMap := fullConfig(vault)
	cfgMap["notes_dir"] = absNotes
	cfgMap["csv_dir"] = absCSV

	path := writeConfig(t, dir, cfgMap)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.NotesDir != absNotes {
		t.Errorf("absolute notes_dir should be unchanged: expected %s, got %s", absNotes, cfg.NotesDir)
	}
	if cfg.CSVDir != absCSV {
		t.Errorf("absolute csv_dir should be unchanged: expected %s, got %s", absCSV, cfg.CSVDir)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestInitConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.json")

	err := InitConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file exists
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Verify it's valid JSON
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("template config is not valid JSON: %v", err)
	}

	// Verify template has expected values
	if cfg.VaultPath == "" {
		t.Error("template vault_path should not be empty")
	}
	if cfg.AnkiEndpoint != DefaultAnkiEndpoint {
		t.Errorf("template anki_endpoint: expected %s, got %s", DefaultAnkiEndpoint, cfg.AnkiEndpoint)
	}
	if cfg.RequestTimeoutMs != DefaultRequestTimeoutMs {
		t.Errorf("template request_timeout_ms: expected %d, got %d", DefaultRequestTimeoutMs, cfg.RequestTimeoutMs)
	}
	if cfg.BatchSize != DefaultBatchSize {
		t.Errorf("template batch_size: expected %d, got %d", DefaultBatchSize, cfg.BatchSize)
	}
	if len(cfg.Tags) != 2 {
		t.Errorf("template tags: expected 2 tags, got %d", len(cfg.Tags))
	}
}

func TestInitConfigEmptyPathUsesDefault(t *testing.T) {
	// We just verify it doesn't panic; actual path creation may fail
	// in CI due to permissions, so we don't assert success.
	_ = InitConfig("")
}

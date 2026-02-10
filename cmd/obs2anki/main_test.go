package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunNoArgs(t *testing.T) {
	code := run(nil)
	if code != ExitFatal {
		t.Errorf("expected exit code %d, got %d", ExitFatal, code)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	code := run([]string{"foobar"})
	if code != ExitFatal {
		t.Errorf("expected exit code %d, got %d", ExitFatal, code)
	}
}

func TestRunHelp(t *testing.T) {
	for _, cmd := range []string{"help", "--help", "-h"} {
		code := run([]string{cmd})
		if code != ExitSuccess {
			t.Errorf("command %q: expected exit code %d, got %d", cmd, ExitSuccess, code)
		}
	}
}

func TestRunInitConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	code := run([]string{"init-config", "--config", cfgPath})
	if code != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, code)
	}
}

func writeScanConfig(t *testing.T, dir string) string {
	t.Helper()
	vault := filepath.Join(dir, "vault")
	notesDir := filepath.Join(vault, "vocab")
	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := map[string]any{
		"vault_path": vault,
		"notes_dir":  "vocab",
		"deck":       "TestDeck",
		"model":      "Basic",
		"csv_dir":    ".anki_csv",
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return cfgPath
}

func TestRunScan(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeScanConfig(t, dir)
	code := run([]string{"scan", "--config", cfgPath})
	if code != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, code)
	}
}

func TestRunScanNoConfig(t *testing.T) {
	code := run([]string{"scan", "--config", "/nonexistent/config.json"})
	if code != ExitFatal {
		t.Errorf("expected exit code %d, got %d", ExitFatal, code)
	}
}

func TestRunSync(t *testing.T) {
	code := run([]string{"sync"})
	if code != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, code)
	}
}

func TestRunTUI(t *testing.T) {
	code := run([]string{"tui"})
	if code != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, code)
	}
}

func TestGlobalFlags(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeScanConfig(t, dir)
	code := run([]string{"scan", "--config", cfgPath, "--dry-run", "--verbose"})
	if code != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, code)
	}
}

func TestExitCodes(t *testing.T) {
	if ExitSuccess != 0 {
		t.Errorf("ExitSuccess should be 0, got %d", ExitSuccess)
	}
	if ExitFatal != 1 {
		t.Errorf("ExitFatal should be 1, got %d", ExitFatal)
	}
	if ExitPartial != 2 {
		t.Errorf("ExitPartial should be 2, got %d", ExitPartial)
	}
}

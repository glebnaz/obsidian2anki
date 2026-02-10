package main

import (
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

func TestRunScan(t *testing.T) {
	code := run([]string{"scan"})
	if code != ExitSuccess {
		t.Errorf("expected exit code %d, got %d", ExitSuccess, code)
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
	// Test that flags are parsed without error
	code := run([]string{"scan", "--config", "/tmp/test.json", "--dry-run", "--verbose"})
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

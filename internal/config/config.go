package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultConfigDir = ".config/obs2anki"
const DefaultConfigFile = "config.json"
const DefaultAnkiEndpoint = "http://127.0.0.1:8765"
const DefaultRequestTimeoutMs = 5000
const DefaultBatchSize = 100

// Config holds all configuration for obs2anki.
type Config struct {
	// Required fields
	VaultPath string `json:"vault_path"`
	NotesDir  string `json:"notes_dir"`
	Deck      string `json:"deck"`
	Model     string `json:"model"`
	CSVDir    string `json:"csv_dir"`

	// Optional fields
	AnkiEndpoint    string   `json:"anki_endpoint"`
	MarkCheckbox    bool     `json:"mark_checkbox"`
	AllowDuplicates bool     `json:"allow_duplicates"`
	Tags            []string `json:"tags"`
	RequestTimeoutMs int     `json:"request_timeout_ms"`
	BatchSize       int      `json:"batch_size"`
}

// DefaultConfigPath returns the default config file path: ~/.config/obs2anki/config.json.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, DefaultConfigDir, DefaultConfigFile), nil
}

// Load reads and parses a config file from the given path.
// If path is empty, it uses the default config path.
func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse config file %s: %w", path, err)
	}

	applyDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	resolvePaths(&cfg)

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.AnkiEndpoint == "" {
		cfg.AnkiEndpoint = DefaultAnkiEndpoint
	}
	if cfg.Tags == nil {
		cfg.Tags = []string{"obsidian", "voc_list"}
	}
	if cfg.RequestTimeoutMs == 0 {
		cfg.RequestTimeoutMs = DefaultRequestTimeoutMs
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = DefaultBatchSize
	}
}

func validate(cfg *Config) error {
	var missing []string
	if cfg.VaultPath == "" {
		missing = append(missing, "vault_path")
	}
	if cfg.NotesDir == "" {
		missing = append(missing, "notes_dir")
	}
	if cfg.Deck == "" {
		missing = append(missing, "deck")
	}
	if cfg.Model == "" {
		missing = append(missing, "model")
	}
	if cfg.CSVDir == "" {
		missing = append(missing, "csv_dir")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config fields: %s", strings.Join(missing, ", "))
	}
	return nil
}

func resolvePaths(cfg *Config) {
	if !filepath.IsAbs(cfg.NotesDir) {
		cfg.NotesDir = filepath.Join(cfg.VaultPath, cfg.NotesDir)
	}
	if !filepath.IsAbs(cfg.CSVDir) {
		cfg.CSVDir = filepath.Join(cfg.VaultPath, cfg.CSVDir)
	}
}

// InitConfig creates the default config directory and writes a template config file.
// If path is empty, it uses the default config path.
func InitConfig(path string) error {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return err
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create config directory %s: %w", dir, err)
	}

	template := Config{
		VaultPath:        "/path/to/your/obsidian/vault",
		NotesDir:         "vocab",
		Deck:             "Default",
		Model:            "Basic",
		CSVDir:           ".anki_csv",
		AnkiEndpoint:     DefaultAnkiEndpoint,
		MarkCheckbox:     false,
		AllowDuplicates:  false,
		Tags:             []string{"obsidian", "voc_list"},
		RequestTimeoutMs: DefaultRequestTimeoutMs,
		BatchSize:        DefaultBatchSize,
	}

	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal template config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("cannot write config file %s: %w", path, err)
	}

	return nil
}

package obsidian

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// MarkSyncedOptions holds parameters for marking a file as synced.
type MarkSyncedOptions struct {
	Deck         string
	Model        string
	MarkCheckbox bool
	Now          time.Time // If zero, uses time.Now().
}

// MarkSynced updates the given markdown file's YAML frontmatter to record
// anki_synced: true, anki_synced_at (RFC3339), anki_deck, and anki_model.
// If mark_checkbox is true, it also replaces "- [ ] anki_synced" with
// "- [x] anki_synced" in the body, or appends "- [x] anki_synced" if absent.
// The file is written atomically via a temp file and rename.
func MarkSynced(path string, opts MarkSyncedOptions) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	content := string(data)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	result := updateFrontmatter(content, opts.Deck, opts.Model, now)

	if opts.MarkCheckbox {
		result = updateCheckbox(result)
	}

	return atomicWrite(path, []byte(result))
}

// updateFrontmatter updates or creates YAML frontmatter with sync metadata.
func updateFrontmatter(content, deck, model string, now time.Time) string {
	fm := extractFrontmatter(content)
	timestamp := now.Format(time.RFC3339)

	if fm == "" {
		// No frontmatter exists: create one.
		newFM := buildFrontmatter(nil, deck, model, timestamp)
		return "---\n" + newFM + "---\n" + content
	}

	// Parse existing frontmatter into an ordered map.
	var meta yaml.Node
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		// If unparseable, create new frontmatter and prepend.
		newFM := buildFrontmatter(nil, deck, model, timestamp)
		return "---\n" + newFM + "---\n" + content
	}

	setYAMLKey(&meta, "anki_synced", "true")
	setYAMLKey(&meta, "anki_synced_at", timestamp)
	setYAMLKey(&meta, "anki_deck", deck)
	setYAMLKey(&meta, "anki_model", model)

	out, err := yaml.Marshal(&meta)
	if err != nil {
		newFM := buildFrontmatter(nil, deck, model, timestamp)
		return "---\n" + newFM + "---\n" + content
	}

	newFM := strings.TrimRight(string(out), "\n") + "\n"

	// Replace old frontmatter in original content.
	// Content starts with "---\n", then frontmatter, then "---\n".
	afterFM := stripFrontmatter(content)
	return "---\n" + newFM + "---\n" + afterFM
}

// buildFrontmatter creates frontmatter YAML string from scratch.
func buildFrontmatter(existing map[string]any, deck, model, timestamp string) string {
	lines := []string{
		"anki_synced: true",
		"anki_synced_at: " + timestamp,
		"anki_deck: " + deck,
		"anki_model: " + model,
	}
	return strings.Join(lines, "\n") + "\n"
}

// stripFrontmatter returns the content after the frontmatter block.
// Assumes content has been normalized (\r\n -> \n).
func stripFrontmatter(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return content
	}
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			// Return everything after the closing ---
			if i+1 < len(lines) {
				return strings.Join(lines[i+1:], "\n")
			}
			return ""
		}
	}
	return content
}

// setYAMLKey sets or updates a key in a yaml.Node document.
func setYAMLKey(doc *yaml.Node, key, value string) {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return
	}
	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return
	}

	newVal := newScalarNode(key, value)

	// Search for existing key.
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = newVal
			return
		}
	}

	// Key not found: append.
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	mapping.Content = append(mapping.Content, keyNode, newVal)
}

// newScalarNode creates a yaml.Node for a value. Boolean keys get bool tag,
// all others are untagged so yaml.Marshal uses the most natural representation.
func newScalarNode(key, value string) *yaml.Node {
	if key == "anki_synced" {
		return &yaml.Node{Kind: yaml.ScalarNode, Value: "true", Tag: "!!bool"}
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Value: value}
}

// updateCheckbox replaces "- [ ] anki_synced" with "- [x] anki_synced"
// in the body (after frontmatter). If no such checkbox exists, appends one.
func updateCheckbox(content string) string {
	const unchecked = "- [ ] anki_synced"
	const checked = "- [x] anki_synced"

	if strings.Contains(content, unchecked) {
		return strings.Replace(content, unchecked, checked, 1)
	}

	// If already checked, leave as is.
	if strings.Contains(content, checked) {
		return content
	}

	// Append the checkbox.
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + checked + "\n"
}

// atomicWrite writes data to a temp file in the same directory and renames it.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".obs2anki-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	// Preserve original file permissions if possible.
	info, statErr := os.Stat(path)
	if statErr == nil {
		if chErr := tmp.Chmod(info.Mode()); chErr != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
			return fmt.Errorf("setting permissions: %w", chErr)
		}
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

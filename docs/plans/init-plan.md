# Plan: Build obs2anki CLI to sync Obsidian markdown vocab tables into Anki

## Overview

Implement a Go CLI tool `obs2anki` that scans all `.md` files in `{vault_path}/{notes_dir}`, skips files with `anki_synced: true` in YAML frontmatter, extracts a markdown table with exactly two columns `Front` and `Back`, exports cards to CSV using `;` with exactly one `;` per data line, imports cards into Anki via AnkiConnect (HTTP JSON API), and marks each file as synced only after all cards from that file are successfully imported. Provide CLI commands `init-config`, `scan`, `sync`, and a Bubble Tea TUI command `tui`.

## Validation Commands

* `go test ./...`
* `golangci-lint run`

### Task 1: Initialize project and CLI skeleton

* [ ] Create Go module
* [ ] Create directories `cmd/obs2anki`, `internal/config`, `internal/obsidian`, `internal/parser`, `internal/csvout`, `internal/anki`, `internal/sync`, `internal/tui`
* [ ] Implement command routing using `os.Args`
* [ ] Implement `flag.FlagSet` for `init-config`, `scan`, `sync`, `tui`
* [ ] Add flags `--config`, `--dry-run`, `--verbose`
* [ ] Define exit codes `0` success, `1` fatal, `2` partial
* [ ] Mark completed

### Task 2: Implement JSON config and loading rules

* [ ] Set default config path `~/.config/obs2anki/config.json`
* [ ] Define required fields `vault_path`, `notes_dir`, `deck`, `model`, `csv_dir`
* [ ] Define optional fields `anki_endpoint` default `http://127.0.0.1:8765`
* [ ] Define optional fields `mark_checkbox` default false
* [ ] Define optional fields `allow_duplicates` default false
* [ ] Define optional fields `tags` default `["obsidian","voc_list"]`
* [ ] Define optional fields `request_timeout_ms` default 5000
* [ ] Define optional fields `batch_size` default 100
* [ ] Load config via `encoding/json`
* [ ] Resolve `notes_dir` relative to `vault_path`
* [ ] Resolve `csv_dir` relative to `vault_path` if not absolute
* [ ] Validate required fields and return error listing all missing fields
* [ ] Implement `init-config` to create `~/.config/obs2anki` and write a full template JSON
* [ ] Mark completed

### Task 3: Scan markdown files and detect synced state

* [ ] Recursively list all `.md` files under `join(vault_path, notes_dir)`
* [ ] Sort file paths lexicographically
* [ ] Read each file as UTF-8
* [ ] Extract YAML frontmatter only if the first line is `---`
* [ ] End frontmatter at the next line that is exactly `---`
* [ ] Parse frontmatter using `yaml.v3`
* [ ] Treat file as synced only if `anki_synced` exists and is boolean true
* [ ] Implement `scan` output: path, synced true/false, has_table true/false, cards_count integer
* [ ] Mark completed

### Task 4: Parse markdown table `Front | Back` with exactly two columns

* [ ] Find the first table whose header row has exactly `Front` and `Back` (case-sensitive)
* [ ] Require a valid separator row after the header
* [ ] Parse each data row into exactly two cells
* [ ] Trim whitespace around cells
* [ ] Preserve cell content as-is including `<br>`
* [ ] Skip row with warning if not exactly two cells
* [ ] Skip row with warning if Front is empty
* [ ] Skip row with warning if Back is empty
* [ ] Return cards and warnings
* [ ] Mark completed

### Task 5: Export CSV using `;` with exactly one delimiter per data line

* [ ] Write header `Front;Back\n`
* [ ] For each card write `<Front>;<Back>\n`
* [ ] Replace all `;` in Front with `,`
* [ ] Replace all `;` in Back with `,`
* [ ] Normalize `\r\n` and `\r` to `\n` inside fields
* [ ] Do not quote fields
* [ ] Write CSV to `<csv_dir>/<md_base_name>-<timestamp_rfc3339_basic>.csv`
* [ ] Validate each data line contains exactly one `;`
* [ ] Mark completed

### Task 6: Implement AnkiConnect HTTP client

* [ ] Use HTTP POST to `anki_endpoint` with `Content-Type: application/json`
* [ ] Use request `version` value 6 in every call
* [ ] Use timeout `request_timeout_ms`
* [ ] Send request JSON with fields `action`, `version`, and optional `params`
* [ ] Parse response JSON with fields `result` and `error`
* [ ] Treat non-null `error` as Go error
* [ ] Treat non-200 HTTP status as Go error including a short body snippet
* [ ] Implement `Version()` using action `version` returning int
* [ ] Implement `DeckNames()` using action `deckNames` returning []string
* [ ] Implement `CreateDeck(name)` using action `createDeck` with params `{ "deck": name }` returning deck id
* [ ] Implement `ModelNames()` using action `modelNames` returning []string
* [ ] Add `httptest.Server` tests for success, anki error, non-JSON, and HTTP 500
* [ ] Mark completed

### Task 7: Ensure deck and model exist

* [ ] Implement `EnsureDeck(deck)` by calling `deckNames` and `createDeck` if missing
* [ ] Implement `EnsureModel(model)` by calling `modelNames` and `createModel` if missing
* [ ] Implement `createModel` request with params `modelName`, `inOrderFields` `["Front","Back"]`, and one template `Name` `Card 1`, `Front` `{{Front}}`, `Back` `{{Front}}<hr>{{Back}}`
* [ ] Add tests verifying `createDeck` and `createModel` are called when missing
* [ ] Mark completed

### Task 8: Import cards via AnkiConnect addNotes in batches

* [ ] Implement `AddNotes(cards)` using action `addNotes`
* [ ] Send notes in chunks of `batch_size`
* [ ] Build each note with `deckName`, `modelName`, `fields` `{ "Front": "...", "Back": "..." }`, `tags`, and `options.allowDuplicate`
* [ ] Treat any null element in `addNotes` result array as failure and return error
* [ ] Mark completed

### Task 9: Mark markdown file as synced

* [ ] Update or create YAML frontmatter keys `anki_synced`, `anki_synced_at`, `anki_deck`, `anki_model`
* [ ] Use RFC3339 timestamp for `anki_synced_at`
* [ ] Write file atomically using temp file then rename
* [ ] If `mark_checkbox` is true, replace `- [ ] anki_synced` with `- [x] anki_synced` or append it
* [ ] Mark completed

### Task 10: Implement sync pipeline for CLI command `sync`

* [ ] Call `Version()` and fail without modifying files if it fails
* [ ] Call `EnsureDeck` and `EnsureModel`
* [ ] For each markdown file in sorted order, skip if synced
* [ ] Parse cards; skip file if zero cards
* [ ] Export CSV for the file
* [ ] If `--dry-run`, do not import and do not mark synced
* [ ] Otherwise import all cards and mark synced only if all batches succeed
* [ ] Return exit code 2 if any file failed, else 0
* [ ] Mark completed

### Task 11: Implement Bubble Tea TUI

* [ ] Show list of files with synced status and card count
* [ ] Show preview of first N cards for selected file
* [ ] Run sync for selected file or all files with progress display
* [ ] Key bindings `q` exit, `enter` toggle preview, `s` sync selected, `a` sync all, `r` rescan
* [ ] Reuse the same scan/sync logic as CLI
* [ ] Mark completed


package anki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client communicates with Anki via the AnkiConnect HTTP JSON API.
type Client struct {
	endpoint   string
	httpClient *http.Client
}

// NewClient creates a new AnkiConnect client.
func NewClient(endpoint string, timeoutMs int) *Client {
	return &Client{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutMs) * time.Millisecond,
		},
	}
}

type request struct {
	Action  string      `json:"action"`
	Version int         `json:"version"`
	Params  interface{} `json:"params,omitempty"`
}

type response struct {
	Result json.RawMessage `json:"result"`
	Error  *string         `json:"error"`
}

func (c *Client) do(action string, params interface{}) (json.RawMessage, error) {
	req := request{
		Action:  action,
		Version: 6,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("anki: marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("anki: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anki: send request: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("anki: read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		snippet := string(respBody)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("anki: HTTP %d: %s", httpResp.StatusCode, snippet)
	}

	var resp response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("anki: parse response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("anki: %s", *resp.Error)
	}

	return resp.Result, nil
}

// Version returns the AnkiConnect API version.
func (c *Client) Version() (int, error) {
	raw, err := c.do("version", nil)
	if err != nil {
		return 0, err
	}

	var version int
	if err := json.Unmarshal(raw, &version); err != nil {
		return 0, fmt.Errorf("anki: parse version result: %w", err)
	}

	return version, nil
}

// DeckNames returns the list of deck names in Anki.
func (c *Client) DeckNames() ([]string, error) {
	raw, err := c.do("deckNames", nil)
	if err != nil {
		return nil, err
	}

	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return nil, fmt.Errorf("anki: parse deckNames result: %w", err)
	}

	return names, nil
}

// CreateDeck creates a new deck and returns its ID.
func (c *Client) CreateDeck(name string) (int64, error) {
	raw, err := c.do("createDeck", map[string]string{"deck": name})
	if err != nil {
		return 0, err
	}

	var id int64
	if err := json.Unmarshal(raw, &id); err != nil {
		return 0, fmt.Errorf("anki: parse createDeck result: %w", err)
	}

	return id, nil
}

// ModelNames returns the list of model (note type) names in Anki.
func (c *Client) ModelNames() ([]string, error) {
	raw, err := c.do("modelNames", nil)
	if err != nil {
		return nil, err
	}

	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return nil, fmt.Errorf("anki: parse modelNames result: %w", err)
	}

	return names, nil
}

// CreateModel creates a new note type with Front/Back fields and a single card template.
func (c *Client) CreateModel(name string) (int64, error) {
	params := map[string]interface{}{
		"modelName":     name,
		"inOrderFields": []string{"Front", "Back"},
		"cardTemplates": []map[string]string{
			{
				"Name":  "Card 1",
				"Front": "{{Front}}",
				"Back":  "{{Front}}<hr>{{Back}}",
			},
		},
	}

	raw, err := c.do("createModel", params)
	if err != nil {
		return 0, err
	}

	// createModel returns a model object; extract the id field.
	var model struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(raw, &model); err != nil {
		return 0, fmt.Errorf("anki: parse createModel result: %w", err)
	}

	return model.ID, nil
}

// EnsureDeck checks if a deck exists and creates it if missing.
func (c *Client) EnsureDeck(deck string) error {
	names, err := c.DeckNames()
	if err != nil {
		return fmt.Errorf("anki: ensure deck: %w", err)
	}

	for _, n := range names {
		if n == deck {
			return nil
		}
	}

	_, err = c.CreateDeck(deck)
	if err != nil {
		return fmt.Errorf("anki: ensure deck: %w", err)
	}

	return nil
}

// Note represents a single note to be added via addNotes.
type Note struct {
	DeckName  string
	ModelName string
	Front     string
	Back      string
	Tags      []string
	AllowDup  bool
}

// AddNotes sends notes to Anki in batches using the addNotes action.
// When AllowDup is false, notes already present in Anki are silently skipped
// so that a file with pre-existing cards is still marked as synced.
// Returns an error if any note in any batch fails for a non-duplicate reason.
func (c *Client) AddNotes(notes []Note, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(notes); i += batchSize {
		end := i + batchSize
		if end > len(notes) {
			end = len(notes)
		}
		batch := notes[i:end]

		ankiNotes := make([]map[string]interface{}, len(batch))
		for j, n := range batch {
			ankiNotes[j] = map[string]interface{}{
				"deckName":  n.DeckName,
				"modelName": n.ModelName,
				"fields": map[string]string{
					"Front": n.Front,
					"Back":  n.Back,
				},
				"tags": n.Tags,
				"options": map[string]bool{
					"allowDuplicate": n.AllowDup,
				},
			}
		}

		// When duplicates are not allowed, pre-filter notes that already exist
		// in Anki. This prevents a "cannot create note because it is a duplicate"
		// error from blocking files that were partially or fully synced before.
		if len(batch) > 0 && !batch[0].AllowDup {
			filtered, err := c.filterAddable(ankiNotes)
			if err != nil {
				return fmt.Errorf("anki: canAddNotes batch %d-%d: %w", i, end-1, err)
			}
			if len(filtered) == 0 {
				continue // all notes already in Anki — skip batch
			}
			ankiNotes = filtered
		}

		raw, err := c.do("addNotes", map[string]interface{}{
			"notes": ankiNotes,
		})
		if err != nil {
			return fmt.Errorf("anki: addNotes batch %d-%d: %w", i, end-1, err)
		}

		var results []*int64
		if err := json.Unmarshal(raw, &results); err != nil {
			return fmt.Errorf("anki: parse addNotes result: %w", err)
		}

		for j, id := range results {
			if id == nil {
				return fmt.Errorf("anki: addNotes failed for note %d (Front: %q)", i+j, ankiNotes[j]["fields"].(map[string]string)["Front"])
			}
		}
	}

	return nil
}

// filterAddable calls canAddNotes and returns only the notes that can be added
// (i.e. are not duplicates of existing notes in Anki).
func (c *Client) filterAddable(ankiNotes []map[string]interface{}) ([]map[string]interface{}, error) {
	raw, err := c.do("canAddNotes", map[string]interface{}{"notes": ankiNotes})
	if err != nil {
		return nil, err
	}

	var canAdd []bool
	if err := json.Unmarshal(raw, &canAdd); err != nil {
		return nil, fmt.Errorf("parse canAddNotes result: %w", err)
	}

	var filtered []map[string]interface{}
	for j, ok := range canAdd {
		if ok {
			filtered = append(filtered, ankiNotes[j])
		}
	}
	return filtered, nil
}

// EnsureModel checks if a model exists and creates it if missing.
func (c *Client) EnsureModel(model string) error {
	names, err := c.ModelNames()
	if err != nil {
		return fmt.Errorf("anki: ensure model: %w", err)
	}

	for _, n := range names {
		if n == model {
			return nil
		}
	}

	_, err = c.CreateModel(model)
	if err != nil {
		return fmt.Errorf("anki: ensure model: %w", err)
	}

	return nil
}

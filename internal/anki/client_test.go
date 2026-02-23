package anki

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// helper to create a test server that returns a fixed AnkiConnect response.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := NewClient(srv.URL, 5000)
	return srv, client
}

// helper to decode the incoming AnkiConnect request body.
func decodeRequest(t *testing.T, r *http.Request) request {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	return req
}

func ankiResponse(result interface{}, ankiErr *string) []byte {
	resp := map[string]interface{}{
		"result": result,
		"error":  ankiErr,
	}
	data, _ := json.Marshal(resp)
	return data
}

func strPtr(s string) *string { return &s }

func writeJSON(t *testing.T, w http.ResponseWriter, data []byte) {
	t.Helper()
	if _, err := w.Write(data); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

// --- Version tests ---

func TestVersion_Success(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "version" {
			t.Errorf("expected action version, got %s", req.Action)
		}
		if req.Version != 6 {
			t.Errorf("expected version 6, got %d", req.Version)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		writeJSON(t, w, ankiResponse(6, nil))
	})

	ver, err := client.Version()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != 6 {
		t.Errorf("expected version 6, got %d", ver)
	}
}

func TestVersion_AnkiError(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, ankiResponse(nil, strPtr("unsupported action")))
	})

	_, err := client.Version()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "anki: unsupported action" {
		t.Errorf("expected 'anki: unsupported action', got %q", got)
	}
}

func TestVersion_NonJSON(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []byte("this is not json"))
	})

	_, err := client.Version()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestVersion_HTTP500(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(t, w, []byte("internal server error"))
	})

	_, err := client.Version()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "anki: HTTP 500: internal server error" {
		t.Errorf("unexpected error message: %q", got)
	}
}

// --- DeckNames tests ---

func TestDeckNames_Success(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "deckNames" {
			t.Errorf("expected action deckNames, got %s", req.Action)
		}
		writeJSON(t, w, ankiResponse([]string{"Default", "Vocab"}, nil))
	})

	names, err := client.DeckNames()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 decks, got %d", len(names))
	}
	if names[0] != "Default" || names[1] != "Vocab" {
		t.Errorf("unexpected deck names: %v", names)
	}
}

func TestDeckNames_AnkiError(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, ankiResponse(nil, strPtr("collection is not available")))
	})

	_, err := client.DeckNames()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- CreateDeck tests ---

func TestCreateDeck_Success(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "createDeck" {
			t.Errorf("expected action createDeck, got %s", req.Action)
		}
		// Verify params contains deck name
		params, err := json.Marshal(req.Params)
		if err != nil {
			t.Fatalf("marshal params: %v", err)
		}
		var p map[string]string
		if err := json.Unmarshal(params, &p); err != nil {
			t.Fatalf("unmarshal params: %v", err)
		}
		if p["deck"] != "MyDeck" {
			t.Errorf("expected deck MyDeck, got %s", p["deck"])
		}
		writeJSON(t, w, ankiResponse(1234567890, nil))
	})

	id, err := client.CreateDeck("MyDeck")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 1234567890 {
		t.Errorf("expected id 1234567890, got %d", id)
	}
}

func TestCreateDeck_AnkiError(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, ankiResponse(nil, strPtr("deck creation failed")))
	})

	_, err := client.CreateDeck("Bad")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- ModelNames tests ---

func TestModelNames_Success(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "modelNames" {
			t.Errorf("expected action modelNames, got %s", req.Action)
		}
		writeJSON(t, w, ankiResponse([]string{"Basic", "Cloze"}, nil))
	})

	names, err := client.ModelNames()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 models, got %d", len(names))
	}
	if names[0] != "Basic" || names[1] != "Cloze" {
		t.Errorf("unexpected model names: %v", names)
	}
}

func TestModelNames_AnkiError(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, ankiResponse(nil, strPtr("collection is not available")))
	})

	_, err := client.ModelNames()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- CreateModel tests ---

func TestCreateModel_Success(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "createModel" {
			t.Errorf("expected action createModel, got %s", req.Action)
		}
		params, err := json.Marshal(req.Params)
		if err != nil {
			t.Fatalf("marshal params: %v", err)
		}
		var p map[string]interface{}
		if err := json.Unmarshal(params, &p); err != nil {
			t.Fatalf("unmarshal params: %v", err)
		}
		if p["modelName"] != "MyModel" {
			t.Errorf("expected modelName MyModel, got %v", p["modelName"])
		}
		fields := p["inOrderFields"].([]interface{})
		if len(fields) != 2 || fields[0] != "Front" || fields[1] != "Back" {
			t.Errorf("unexpected inOrderFields: %v", fields)
		}
		templates := p["cardTemplates"].([]interface{})
		if len(templates) != 1 {
			t.Fatalf("expected 1 template, got %d", len(templates))
		}
		tmpl := templates[0].(map[string]interface{})
		if tmpl["Name"] != "Card 1" {
			t.Errorf("expected template name Card 1, got %v", tmpl["Name"])
		}
		if tmpl["Front"] != "{{Front}}" {
			t.Errorf("unexpected front template: %v", tmpl["Front"])
		}
		if tmpl["Back"] != "{{Front}}<hr>{{Back}}" {
			t.Errorf("unexpected back template: %v", tmpl["Back"])
		}
		result := map[string]interface{}{"id": 1609876543210}
		writeJSON(t, w, ankiResponse(result, nil))
	})

	id, err := client.CreateModel("MyModel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 1609876543210 {
		t.Errorf("expected id 1609876543210, got %d", id)
	}
}

func TestCreateModel_AnkiError(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, ankiResponse(nil, strPtr("model already exists")))
	})

	_, err := client.CreateModel("Bad")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- EnsureDeck tests ---

func TestEnsureDeck_AlreadyExists(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "deckNames" {
			t.Errorf("expected only deckNames call, got %s", req.Action)
		}
		writeJSON(t, w, ankiResponse([]string{"Default", "Vocab"}, nil))
	})

	err := client.EnsureDeck("Vocab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureDeck_CreateWhenMissing(t *testing.T) {
	callCount := 0
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		callCount++
		switch req.Action {
		case "deckNames":
			writeJSON(t, w, ankiResponse([]string{"Default"}, nil))
		case "createDeck":
			params, _ := json.Marshal(req.Params)
			var p map[string]string
			_ = json.Unmarshal(params, &p)
			if p["deck"] != "NewDeck" {
				t.Errorf("expected deck NewDeck, got %s", p["deck"])
			}
			writeJSON(t, w, ankiResponse(1234567890, nil))
		default:
			t.Errorf("unexpected action: %s", req.Action)
		}
	})

	err := client.EnsureDeck("NewDeck")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls (deckNames + createDeck), got %d", callCount)
	}
}

func TestEnsureDeck_DeckNamesError(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, ankiResponse(nil, strPtr("collection unavailable")))
	})

	err := client.EnsureDeck("Vocab")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- EnsureModel tests ---

func TestEnsureModel_AlreadyExists(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "modelNames" {
			t.Errorf("expected only modelNames call, got %s", req.Action)
		}
		writeJSON(t, w, ankiResponse([]string{"Basic", "MyModel"}, nil))
	})

	err := client.EnsureModel("MyModel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureModel_CreateWhenMissing(t *testing.T) {
	callCount := 0
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		callCount++
		switch req.Action {
		case "modelNames":
			writeJSON(t, w, ankiResponse([]string{"Basic"}, nil))
		case "createModel":
			params, _ := json.Marshal(req.Params)
			var p map[string]interface{}
			_ = json.Unmarshal(params, &p)
			if p["modelName"] != "NewModel" {
				t.Errorf("expected modelName NewModel, got %v", p["modelName"])
			}
			result := map[string]interface{}{"id": 9876543210}
			writeJSON(t, w, ankiResponse(result, nil))
		default:
			t.Errorf("unexpected action: %s", req.Action)
		}
	})

	err := client.EnsureModel("NewModel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls (modelNames + createModel), got %d", callCount)
	}
}

func TestEnsureModel_ModelNamesError(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, ankiResponse(nil, strPtr("collection unavailable")))
	})

	err := client.EnsureModel("MyModel")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- General error handling tests ---

func TestDo_HTTP500_LongBody(t *testing.T) {
	longBody := make([]byte, 300)
	for i := range longBody {
		longBody[i] = 'x'
	}
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(t, w, longBody)
	})

	_, err := client.Version()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Body snippet should be truncated to 200 chars
	errMsg := err.Error()
	if len(errMsg) > 250 {
		t.Errorf("error message too long, expected truncation: len=%d", len(errMsg))
	}
}

func TestDo_RequestVersion6(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Version != 6 {
			t.Errorf("expected version 6, got %d", req.Version)
		}
		writeJSON(t, w, ankiResponse(6, nil))
	})

	_, _ = client.Version()
}

func TestDo_ContentTypeJSON(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
		writeJSON(t, w, ankiResponse(6, nil))
	})

	_, _ = client.Version()
}

// --- AddNotes tests ---

func TestAddNotes_Success(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		if req.Action != "addNotes" {
			t.Errorf("expected action addNotes, got %s", req.Action)
		}
		// Verify params structure
		params, _ := json.Marshal(req.Params)
		var p map[string]interface{}
		_ = json.Unmarshal(params, &p)
		notes := p["notes"].([]interface{})
		if len(notes) != 2 {
			t.Fatalf("expected 2 notes, got %d", len(notes))
		}
		note0 := notes[0].(map[string]interface{})
		if note0["deckName"] != "TestDeck" {
			t.Errorf("expected deckName TestDeck, got %v", note0["deckName"])
		}
		if note0["modelName"] != "TestModel" {
			t.Errorf("expected modelName TestModel, got %v", note0["modelName"])
		}
		fields := note0["fields"].(map[string]interface{})
		if fields["Front"] != "hello" {
			t.Errorf("expected Front hello, got %v", fields["Front"])
		}
		if fields["Back"] != "world" {
			t.Errorf("expected Back world, got %v", fields["Back"])
		}
		tags := note0["tags"].([]interface{})
		if len(tags) != 1 || tags[0] != "vocab" {
			t.Errorf("unexpected tags: %v", tags)
		}
		opts := note0["options"].(map[string]interface{})
		if opts["allowDuplicate"] != false {
			t.Errorf("expected allowDuplicate false, got %v", opts["allowDuplicate"])
		}

		// Return two successful note IDs
		var id1 int64 = 111
		var id2 int64 = 222
		writeJSON(t, w, ankiResponse([]interface{}{id1, id2}, nil))
	})

	notes := []Note{
		{DeckName: "TestDeck", ModelName: "TestModel", Front: "hello", Back: "world", Tags: []string{"vocab"}, AllowDup: false},
		{DeckName: "TestDeck", ModelName: "TestModel", Front: "foo", Back: "bar", Tags: []string{"vocab"}, AllowDup: false},
	}
	err := client.AddNotes(notes, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddNotes_PartialFailure(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Second note fails (null in result)
		writeJSON(t, w, ankiResponse([]interface{}{111, nil}, nil))
	})

	notes := []Note{
		{DeckName: "D", ModelName: "M", Front: "ok", Back: "ok", Tags: []string{}, AllowDup: false},
		{DeckName: "D", ModelName: "M", Front: "bad", Back: "bad", Tags: []string{}, AllowDup: false},
	}
	err := client.AddNotes(notes, 100)
	if err == nil {
		t.Fatal("expected error for null result element, got nil")
	}
	if got := err.Error(); got != `anki: addNotes failed for note 1 (Front: "bad")` {
		t.Errorf("unexpected error message: %q", got)
	}
}

func TestAddNotes_Batching(t *testing.T) {
	batchCount := 0
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		batchCount++
		req := decodeRequest(t, r)
		params, _ := json.Marshal(req.Params)
		var p map[string]interface{}
		_ = json.Unmarshal(params, &p)
		notes := p["notes"].([]interface{})

		// First batch should have 2 notes, second batch should have 1
		results := make([]interface{}, len(notes))
		for i := range results {
			results[i] = int64(100 + i)
		}
		writeJSON(t, w, ankiResponse(results, nil))
	})

	notes := []Note{
		{DeckName: "D", ModelName: "M", Front: "a", Back: "1", Tags: []string{}, AllowDup: false},
		{DeckName: "D", ModelName: "M", Front: "b", Back: "2", Tags: []string{}, AllowDup: false},
		{DeckName: "D", ModelName: "M", Front: "c", Back: "3", Tags: []string{}, AllowDup: false},
	}
	err := client.AddNotes(notes, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if batchCount != 2 {
		t.Errorf("expected 2 batches, got %d", batchCount)
	}
}

func TestAddNotes_AnkiError(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, ankiResponse(nil, strPtr("collection unavailable")))
	})

	notes := []Note{
		{DeckName: "D", ModelName: "M", Front: "a", Back: "1", Tags: []string{}, AllowDup: false},
	}
	err := client.AddNotes(notes, 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAddNotes_AllowDuplicate(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r)
		params, _ := json.Marshal(req.Params)
		var p map[string]interface{}
		_ = json.Unmarshal(params, &p)
		notes := p["notes"].([]interface{})
		note0 := notes[0].(map[string]interface{})
		opts := note0["options"].(map[string]interface{})
		if opts["allowDuplicate"] != true {
			t.Errorf("expected allowDuplicate true, got %v", opts["allowDuplicate"])
		}
		writeJSON(t, w, ankiResponse([]interface{}{111}, nil))
	})

	notes := []Note{
		{DeckName: "D", ModelName: "M", Front: "a", Back: "1", Tags: []string{}, AllowDup: true},
	}
	err := client.AddNotes(notes, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddNotes_EmptySlice(t *testing.T) {
	// Should succeed immediately with no API calls
	callCount := 0
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		writeJSON(t, w, ankiResponse(nil, nil))
	})

	err := client.AddNotes([]Note{}, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 0 {
		t.Errorf("expected 0 API calls for empty notes, got %d", callCount)
	}
}

func TestAddNotes_BatchFailureStopsEarly(t *testing.T) {
	batchCount := 0
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		batchCount++
		// First batch fails with a null
		writeJSON(t, w, ankiResponse([]interface{}{nil}, nil))
	})

	notes := []Note{
		{DeckName: "D", ModelName: "M", Front: "a", Back: "1", Tags: []string{}, AllowDup: false},
		{DeckName: "D", ModelName: "M", Front: "b", Back: "2", Tags: []string{}, AllowDup: false},
	}
	err := client.AddNotes(notes, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should stop after first batch failure
	if batchCount != 1 {
		t.Errorf("expected 1 batch call before failure, got %d", batchCount)
	}
}

func TestNewClient_Timeout(t *testing.T) {
	client := NewClient("http://localhost:12345", 100)
	if client.httpClient.Timeout.Milliseconds() != 100 {
		t.Errorf("expected timeout 100ms, got %v", client.httpClient.Timeout)
	}
}

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

func TestNewClient_Timeout(t *testing.T) {
	client := NewClient("http://localhost:12345", 100)
	if client.httpClient.Timeout.Milliseconds() != 100 {
		t.Errorf("expected timeout 100ms, got %v", client.httpClient.Timeout)
	}
}

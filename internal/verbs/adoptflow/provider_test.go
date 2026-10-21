package adoptflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/procrastivity/lds/internal/opencode"
)

type localTransport func(*http.Request) (*http.Response, error)

func (f localTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func localProviderClient(t *testing.T, server *httptest.Server) *opencode.Client {
	t.Helper()
	target, err := url.Parse(server.URL + "/v1/chat/completions")
	if err != nil {
		t.Fatal(err)
	}
	return opencode.NewClient(localTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://opencode.ai/zen/go/v1/chat/completions" {
			t.Errorf("unexpected provider endpoint: %s", r.URL)
		}
		local := r.Clone(r.Context())
		local.URL = target
		local.Host = target.Host
		return server.Client().Transport.RoundTrip(local)
	}))
}

func TestRequestAdvisoryAtSendsBoundedFlowFactsAndAggregateEvidence(t *testing.T) {
	const (
		dummyKey = "test-only-not-a-credential"
		flow     = "LIVE FLOW MARKER"
		facts    = `{"DisplayName":"Fixture"}`
	)
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Store(true)
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("provider request did not use the expected POST JSON protocol")
		}
		if r.Header.Get("Authorization") != "Bearer "+dummyKey {
			t.Error("provider request did not use the supplied test credential")
		}
		if r.Header.Get("User-Agent") != providerUserAgent || r.Header.Get("x-opencode-session") != providerSession {
			t.Error("provider request omitted adoption identity headers")
		}
		var request map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode provider request: %v", err)
			return
		}
		if len(request) != 4 || string(request["model"]) != `"deepseek-v4-flash"` || string(request["max_tokens"]) != `768` || string(request["stream"]) != `false` {
			t.Errorf("provider request settings are not the fixed non-streaming model request")
		}
		if _, hasTools := request["tools"]; hasTools {
			t.Error("provider request unexpectedly enabled tools")
		}
		var messages []struct{ Role, Content string }
		if err := json.Unmarshal(request["messages"], &messages); err != nil || len(messages) != 2 {
			t.Errorf("provider messages count=%d decode=%v; want system and user only", len(messages), err)
			return
		}
		userMessage := messages[1].Content
		for _, want := range []string{flow, facts, `"inventoryFiles":3`} {
			if !strings.Contains(userMessage, want) {
				t.Errorf("bounded provider message omits %q", want)
			}
		}
		for _, forbidden := range []string{"/private/target", "source-file-secret", "rendered-output-secret", "sha256:plan-secret"} {
			if strings.Contains(userMessage, forbidden) {
				t.Errorf("provider message contains forbidden target/source/plan content")
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"deepseek-v4-flash","choices":[{"message":{"content":"Maps the supplied display name to the overview."}}]}`))
	}))
	defer server.Close()

	evidence := providerEvidence{InventoryFiles: 3, OutputClassifications: map[string]int{"create": 4}}
	got, err := requestAdvisoryWithClient(context.Background(), localProviderClient(t, server), dummyKey, flow, []byte(facts), evidence)
	if err != nil || got != "Maps the supplied display name to the overview." || !called.Load() {
		t.Fatalf("provider advisory = %q, called=%t, err=%v", got, called.Load(), err)
	}
}

func TestRequestAdvisoryAtRejectsWrongModelAndDoesNotFollowRedirects(t *testing.T) {
	var redirected atomic.Bool
	wrongModel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/wrong-model" {
			redirected.Store(true)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"model":"different-model","choices":[{"message":{"content":"Looks fine."}}]}`))
			return
		}
		w.Header().Set("Location", "/wrong-model")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer wrongModel.Close()

	_, err := requestAdvisoryWithClient(context.Background(), localProviderClient(t, wrongModel), "test-only-not-a-credential", "flow", []byte(`{}`), providerEvidence{})
	if err == nil || !strings.Contains(err.Error(), "unexpected HTTP status") || redirected.Load() {
		t.Fatalf("redirect result err=%v redirected=%t; want refusal without following", err, redirected.Load())
	}

	wrongResponse := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"different-model","choices":[{"message":{"content":"Looks fine."}}]}`))
	}))
	defer wrongResponse.Close()
	_, err = requestAdvisoryWithClient(context.Background(), localProviderClient(t, wrongResponse), "test-only-not-a-credential", "flow", []byte(`{}`), providerEvidence{})
	if err == nil || !strings.Contains(err.Error(), "selected model contract") {
		t.Fatalf("wrong-model response err=%v, want fixed-model refusal", err)
	}
}

func TestRequestAdvisoryRetains16KiBLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := map[string]any{
			"model":   "deepseek-v4-flash",
			"choices": []any{map[string]any{"message": map[string]string{"content": strings.Repeat("a", 16*1024+1)}}},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	got, err := requestAdvisoryWithClient(context.Background(), localProviderClient(t, server), "test-only-not-a-credential", "flow", []byte(`{}`), providerEvidence{})
	if got != "" || err == nil || !strings.Contains(err.Error(), "unsafe text") {
		t.Fatalf("oversized adoption advisory got length=%d, err=%v", len(got), err)
	}
}

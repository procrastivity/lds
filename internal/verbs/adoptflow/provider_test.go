package adoptflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRequestAdvisoryAtSendsBoundedFlowFactsAndAggregateEvidence(t *testing.T) {
	if providerEndpoint != "https://opencode.ai/zen/go/v1/chat/completions" || providerModel != "deepseek-v4-flash" {
		t.Fatal("adoption provider endpoint/model differ from the owner-selected OpenCode Go route")
	}
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
		var messages []providerMessage
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
	got, err := requestAdvisoryAt(context.Background(), server.Client(), server.URL+"/v1/chat/completions", dummyKey, flow, []byte(facts), evidence)
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

	_, err := requestAdvisoryAt(context.Background(), wrongModel.Client(), wrongModel.URL, "test-only-not-a-credential", "flow", []byte(`{}`), providerEvidence{})
	if err == nil || !strings.Contains(err.Error(), "unexpected HTTP status") || redirected.Load() {
		t.Fatalf("redirect result err=%v redirected=%t; want refusal without following", err, redirected.Load())
	}

	wrongResponse := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"different-model","choices":[{"message":{"content":"Looks fine."}}]}`))
	}))
	defer wrongResponse.Close()
	_, err = requestAdvisoryAt(context.Background(), wrongResponse.Client(), wrongResponse.URL, "test-only-not-a-credential", "flow", []byte(`{}`), providerEvidence{})
	if err == nil || !strings.Contains(err.Error(), "selected model contract") {
		t.Fatalf("wrong-model response err=%v, want fixed-model refusal", err)
	}
}

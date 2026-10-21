package opencode_test

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

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func localClient(t *testing.T, server *httptest.Server) *opencode.Client {
	t.Helper()
	target, err := url.Parse(server.URL + "/v1/chat/completions")
	if err != nil {
		t.Fatal(err)
	}
	return opencode.NewClient(transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://opencode.ai/zen/go/v1/chat/completions" {
			t.Errorf("unexpected endpoint: %s", r.URL)
		}
		local := r.Clone(r.Context())
		local.URL = target
		local.Host = target.Host
		return server.Client().Transport.RoundTrip(local)
	}))
}

func reply(model, content string) string {
	encoded, _ := json.Marshal(map[string]any{
		"model":   model,
		"choices": []any{map[string]any{"message": map[string]string{"content": content}}},
	})
	return string(encoded)
}

func TestCompleteRequestAndContentBounds(t *testing.T) {
	const key = "dummy-test-key"
	var responseText string
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Authorization") != "Bearer "+key || r.Header.Get("User-Agent") != "assessment/1" || r.Header.Get("x-opencode-session") != "assessment-session" {
			t.Error("incorrect request method, endpoint, or headers")
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if len(body) != 4 || string(body["model"]) != `"deepseek-v4-flash"` || string(body["max_tokens"]) != "4096" || string(body["stream"]) != "false" {
			t.Errorf("wrong model, token limit, or streaming/tools settings: %v", body)
		}
		var messages []map[string]string
		if err := json.Unmarshal(body["messages"], &messages); err != nil || len(messages) != 2 || messages[0]["role"] != "system" || messages[0]["content"] != "system prompt" || messages[1]["role"] != "user" || messages[1]["content"] != "user prompt" {
			t.Errorf("wrong messages: %v (%v)", messages, err)
		}
		_, _ = w.Write([]byte(responseText))
	}))
	defer server.Close()
	client := localClient(t, server)
	request := opencode.Request{UserAgent: "assessment/1", Session: "assessment-session", MaxTokens: 4096, System: "system prompt", User: "user prompt"}
	for _, tc := range []struct {
		name, content string
		valid         bool
	}{
		{"newlines tabs", "safe\n\tresponse", true},
		{"64KiB", strings.Repeat("x", 64*1024), true},
		{"too long", strings.Repeat("x", 64*1024+1), false},
		{"unsafe control", "unsafe\rresponse", false},
		{"empty", " \n\t", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			responseText = reply("deepseek-v4-flash", tc.content)
			got, err := client.Complete(context.Background(), key, request)
			if tc.valid && (err != nil || got != tc.content) || !tc.valid && (err == nil || got != "") {
				t.Fatalf("got length=%d, err=%v; valid=%t", len(got), err, tc.valid)
			}
		})
	}
	if calls.Load() != 5 {
		t.Fatalf("calls = %d; want exactly one per completion", calls.Load())
	}
}

func TestCompleteRejectsMalformedResponsesWithoutLeakingBody(t *testing.T) {
	const secret = "response-secret-must-not-leak"
	for _, tc := range []struct {
		name  string
		serve func(http.ResponseWriter, *http.Request)
	}{
		{"wrong model", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(reply("other", "okay"))) }},
		{"missing choice", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"model":"deepseek-v4-flash","choices":[]}`))
		}},
		{"malformed", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(secret)) }},
		{"invalid UTF-8", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"model":"deepseek-v4-flash","choices":[{"message":{"content":"`))
			_, _ = w.Write([]byte{0xff})
			_, _ = w.Write([]byte(`"}}]}`))
		}},
		{"oversized body", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(strings.Repeat("x", 1<<20+1))) }},
		{"truncated body", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", "9999")
			_, _ = w.Write([]byte(secret))
		}},
		{"HTTP refusal", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(secret))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tc.serve))
			defer server.Close()
			got, err := localClient(t, server).Complete(context.Background(), "dummy-test-key", opencode.Request{MaxTokens: 4096})
			if err == nil || got != "" || strings.Contains(err.Error(), secret) {
				t.Fatalf("got %q, err=%v; want sanitized refusal", got, err)
			}
		})
	}
}

func TestCompleteDoesNotFollowRedirect(t *testing.T) {
	var followed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/next" {
			followed.Store(true)
			_, _ = w.Write([]byte(reply("deepseek-v4-flash", "okay")))
			return
		}
		w.Header().Set("Location", "/next")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	got, err := localClient(t, server).Complete(context.Background(), "dummy-test-key", opencode.Request{MaxTokens: 4096})
	if err == nil || got != "" || followed.Load() || !strings.Contains(err.Error(), "unexpected HTTP status") {
		t.Fatalf("redirect got %q, err=%v, followed=%t", got, err, followed.Load())
	}
}

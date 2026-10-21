// Package opencode provides a single bounded OpenCode Go chat completion.
package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	endpoint   = "https://opencode.ai/zen/go/v1/chat/completions"
	model      = "deepseek-v4-flash"
	maxReply   = 1 << 20
	maxContent = 64 * 1024
)

// Request supplies the caller's identity and two messages. The model, endpoint,
// streaming policy, and HTTP bounds are fixed by this package.
type Request struct {
	UserAgent string
	Session   string
	MaxTokens int
	System    string
	User      string
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type completionRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Stream    bool      `json:"stream"`
	Messages  []message `json:"messages"`
}

type completionResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Client permits a custom HTTP transport (for example, a local test server).
// It cannot change the production endpoint or relax the timeout or redirect policy.
type Client struct {
	transport http.RoundTripper
}

// NewClient uses the supplied transport, or http.DefaultTransport when nil.
func NewClient(transport http.RoundTripper) *Client {
	return &Client{transport: transport}
}

// Complete makes exactly one non-streaming request, without tools or retries.
func Complete(ctx context.Context, key string, request Request) (string, error) {
	return NewClient(nil).Complete(ctx, key, request)
}

// Complete makes one request through the client's transport with fixed bounds.
func (c *Client) Complete(ctx context.Context, key string, request Request) (string, error) {
	if key == "" {
		return "", errors.New("provider credential unavailable")
	}
	if request.MaxTokens <= 0 {
		return "", errors.New("provider token limit is invalid")
	}
	encoded, err := json.Marshal(completionRequest{
		Model: model, MaxTokens: request.MaxTokens, Stream: false,
		Messages: []message{{Role: "system", Content: request.System}, {Role: "user", Content: request.User}},
	})
	if err != nil {
		return "", errors.New("could not encode provider request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return "", errors.New("could not construct provider request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("User-Agent", request.UserAgent)
	req.Header.Set("x-opencode-session", request.Session)
	client := &http.Client{
		Transport:     c.transport,
		Timeout:       60 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	response, err := client.Do(req)
	if err != nil {
		return "", errors.New("provider request failed")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return "", errors.New("provider returned an unexpected HTTP status")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxReply+1))
	if err != nil || len(body) > maxReply {
		return "", errors.New("provider response could not be read safely")
	}
	var decoded completionResponse
	if !utf8.Valid(body) || json.Unmarshal(body, &decoded) != nil || decoded.Model != model || len(decoded.Choices) != 1 {
		return "", errors.New("provider response did not match the selected model contract")
	}
	content := decoded.Choices[0].Message.Content
	if !safeContent(content) {
		return "", errors.New("provider advisory was empty or contained unsafe text")
	}
	return content, nil
}

func safeContent(content string) bool {
	if len(content) == 0 || len(content) > maxContent || !utf8.ValidString(content) || strings.TrimSpace(content) == "" {
		return false
	}
	for _, r := range content {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return false
		}
	}
	return true
}

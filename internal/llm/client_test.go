package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewLLMClientAnthropic(t *testing.T) {
	client, err := NewLLMClient("anthropic", "test-key", "")
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	client.Close()
}

func TestNewLLMClientOpenAI(t *testing.T) {
	client, err := NewLLMClient("openai", "test-key", "")
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	client.Close()
}

func TestNewLLMClientUnknown(t *testing.T) {
	_, err := NewLLMClient("unknown", "key", "")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestAnthropicChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Error("missing or wrong API key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Error("missing anthropic-version header")
		}

		resp := map[string]any{
			"id":          "msg_001",
			"model":       "claude-sonnet-4",
			"stop_reason": "end_turn",
			"content": []map[string]any{
				{"type": "text", "text": "Hello from Claude"},
			},
			"usage": map[string]int{
				"input_tokens":  10,
				"output_tokens": 5,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key", server.URL)
	defer client.Close()

	req := &ChatRequest{
		Model:     "claude-sonnet-4",
		System:    "You are helpful.",
		Messages:  []Message{{Role: "user", Content: []ContentBlock{{Type: "text", Text: "hi"}}}},
		MaxTokens: 100,
	}

	resp, err := client.Chat(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("input tokens: got %d, want 10", resp.Usage.InputTokens)
	}
	if len(resp.Messages) == 0 || resp.Messages[0].Content[0].Text != "Hello from Claude" {
		t.Error("unexpected response content")
	}
}

func TestOpenAIChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing or wrong auth header")
		}

		resp := map[string]any{
			"id":    "chatcmpl-001",
			"model": "gpt-4o",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello from GPT",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient("test-key", server.URL)
	defer client.Close()

	req := &ChatRequest{
		Model:     "gpt-4o",
		System:    "You are helpful.",
		Messages:  []Message{{Role: "user", Content: []ContentBlock{{Type: "text", Text: "hi"}}}},
		MaxTokens: 100,
	}

	resp, err := client.Chat(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Usage.OutputTokens != 5 {
		t.Errorf("output tokens: got %d, want 5", resp.Usage.OutputTokens)
	}
}

func TestAPIErrorIsRetryable(t *testing.T) {
	tests := []struct {
		status   int
		retryable bool
	}{
		{429, true},
		{529, true},
		{503, true},
		{400, false},
		{401, false},
		{404, false},
	}
	for _, tt := range tests {
		err := &APIError{Status: tt.status, Type: "test", Message: "test"}
		if err.IsRetryable() != tt.retryable {
			t.Errorf("status %d: expected retryable=%v", tt.status, tt.retryable)
		}
	}
}

func TestRetryWithBackoff(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	fn := func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", &APIError{Status: 429, Type: "rate_limit", Message: "try later"}
		}
		return "success", nil
	}

	result, err := RetryWithBackoff(ctx, 3, fn, func(err error) bool {
		if apiErr, ok := err.(*APIError); ok {
			return apiErr.IsRetryable()
		}
		return false
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "success" {
		t.Errorf("got %q, want 'success'", result)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const anthropicVersion = "2023-06-01"

func init() {
	Register(anthropicProvider{})
}

type anthropicProvider struct{}

func (anthropicProvider) Name() string               { return "anthropic" }
func (anthropicProvider) DefaultBaseURL() string      { return "https://api.anthropic.com/v1" }
func (anthropicProvider) NewClient(apiKey, baseURL string) LLMClient {
	return NewAnthropicClient(apiKey, baseURL)
}

// AnthropicClient implements LLMClient for the Anthropic Messages API.
type AnthropicClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewAnthropicClient creates a new Anthropic API client.
func NewAnthropicClient(apiKey, baseURL string) *AnthropicClient {
	return &AnthropicClient{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// Chat sends a non-streaming request to the Anthropic API.
func (c *AnthropicClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	reqBody := c.buildRequest(req)
	reqBody["stream"] = false

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp, body)
	}

	var result anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return c.toChatResponse(&result), nil
}

// ChatStream sends a streaming request to the Anthropic API.
func (c *AnthropicClient) ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
	reqBody := c.buildRequest(req)
	reqBody["stream"] = true

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, c.parseError(resp, body)
	}

	events := make(chan StreamEvent, 64)
	go c.readSSEStream(ctx, resp.Body, events)
	return events, nil
}

func (c *AnthropicClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// buildRequest converts a ChatRequest to the Anthropic API format.
func (c *AnthropicClient) buildRequest(req *ChatRequest) map[string]interface{} {
	apiReq := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
	}

	if req.Temperature > 0 {
		apiReq["temperature"] = req.Temperature
	}

	if req.System != "" {
		apiReq["system"] = req.System
	}

	// Convert messages
	messages := make([]map[string]interface{}, 0, len(req.Messages))
	for _, msg := range req.Messages {
		m := map[string]interface{}{
			"role":    msg.Role,
			"content": c.convertContent(msg.Content),
		}
		messages = append(messages, m)
	}
	apiReq["messages"] = messages

	// Convert tools
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, 0, len(req.Tools))
		for _, t := range req.Tools {
			tool := map[string]interface{}{
				"name":         t.Name,
				"description":  t.Description,
				"input_schema": json.RawMessage(t.InputSchema),
			}
			tools = append(tools, tool)
		}
		apiReq["tools"] = tools
	}

	return apiReq
}

func (c *AnthropicClient) convertContent(blocks []ContentBlock) interface{} {
	// If it's a single text block, return as string for simplicity
	if len(blocks) == 1 && blocks[0].Type == "text" {
		return blocks[0].Text
	}

	result := make([]map[string]interface{}, 0, len(blocks))
	for _, b := range blocks {
		item := map[string]interface{}{
			"type": b.Type,
		}
		switch b.Type {
		case "text":
			item["text"] = b.Text
			if b.Cached != nil {
				item["cache_control"] = b.Cached
			}
		case "tool_use":
			item["id"] = b.ID
			item["name"] = b.Name
			item["input"] = b.Input
		case "tool_result":
			item["tool_use_id"] = b.ToolUseID
			item["content"] = b.Content
			if b.IsError {
				item["is_error"] = true
			}
		case "image":
			item["source"] = map[string]interface{}{
				"type":       "base64",
				"media_type": "image/png",
				"data":       b.Text, // base64 data in Text field
			}
		}
		result = append(result, item)
	}
	return result
}

func (c *AnthropicClient) setHeaders(req *http.Request) {
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("Content-Type", "application/json")
}

// readSSEStream reads Server-Sent Events from the response body.
func (c *AnthropicClient) readSSEStream(ctx context.Context, body io.ReadCloser, events chan<- StreamEvent) {
	defer close(events)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024) // 2MB max

	var currentEvent string
	var currentData strings.Builder
	var messageStopSeen bool

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			events <- StreamEvent{Type: EventError, Error: ctx.Err()}
			return
		default:
		}

		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "event: "):
			currentEvent = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			currentData.WriteString(strings.TrimPrefix(line, "data: "))
		case line == "":
			// Empty line marks end of an event
			if currentData.Len() == 0 {
				continue
			}

			data := currentData.String()

			switch currentEvent {
			case "message_start":
				var evt anthropicMessageStartEvent
				if err := json.Unmarshal([]byte(data), &evt); err == nil {
					rawMsg, _ := json.Marshal(evt.Message)
					events <- StreamEvent{Type: EventMessageStart, Data: rawMsg}
				}
			case "content_block_start":
				var evt anthropicContentBlockStartEvent
				if err := json.Unmarshal([]byte(data), &evt); err == nil {
					if evt.ContentBlock.Type == "tool_use" {
						raw, _ := json.Marshal(map[string]interface{}{
							"id":    evt.ContentBlock.ID,
							"name":  evt.ContentBlock.Name,
							"index": evt.Index,
						})
						events <- StreamEvent{Type: EventToolUseStart, Data: raw}
					}
				}
			case "content_block_delta":
				var evt anthropicContentBlockDeltaEvent
				if err := json.Unmarshal([]byte(data), &evt); err == nil {
					switch evt.Delta.Type {
					case "text_delta":
						raw, _ := json.Marshal(map[string]string{"text": evt.Delta.Text})
						events <- StreamEvent{Type: EventText, Data: raw}
						case "thinking_delta":
							raw, _ := json.Marshal(map[string]string{"thinking": evt.Delta.Thinking})
							events <- StreamEvent{Type: EventThinking, Data: raw}
					case "input_json_delta":
						raw, _ := json.Marshal(map[string]string{"partial_json": evt.Delta.PartialJSON})
						events <- StreamEvent{Type: EventToolUseEnd, Data: raw}
					}
				}
			case "content_block_stop":
				var evt anthropicContentBlockStopEvent
				if err := json.Unmarshal([]byte(data), &evt); err == nil {
					raw, _ := json.Marshal(evt)
					events <- StreamEvent{Type: EventToolUseEnd, Data: raw}
				}
			case "message_delta":
				messageStopSeen = true
				var evt anthropicMessageDeltaEvent
				if err := json.Unmarshal([]byte(data), &evt); err == nil {
					raw, _ := json.Marshal(evt)
					events <- StreamEvent{Type: EventMessageDelta, Data: raw}
				}
			case "message_stop":
				if !messageStopSeen {
					// message_stop without message_delta: emit a synthetic done
					events <- StreamEvent{Type: EventDone, Data: json.RawMessage(`{"stop_reason": "end_turn"}`)}
				}
			case "error":
				var evt anthropicErrorEvent
				if err := json.Unmarshal([]byte(data), &evt); err == nil {
					events <- StreamEvent{Type: EventError,
						Error: &APIError{
							Type:    evt.Error.Type,
							Message: evt.Error.Message,
							Status:  respStatus(evt.Error.Type),
						},
					}
				}
			case "ping":
				// Ignore ping events
			default:
				// Unknown event type, skip
			}

			currentEvent = ""
			currentData.Reset()
		}
	}

	if err := scanner.Err(); err != nil {
		events <- StreamEvent{Type: EventError, Error: fmt.Errorf("scan stream: %w", err)}
		return
	}

	events <- StreamEvent{Type: EventDone, Data: json.RawMessage(`{"stop_reason": "end_turn"}`)}
}

// Anthropic API response/event types
type anthropicResponse struct {
	ID         string                `json:"id"`
	Model      string                `json:"model"`
	StopReason string                `json:"stop_reason"`
	Content    []anthropicContent    `json:"content"`
	Usage      anthropicUsage        `json:"usage"`
}

type anthropicContent struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicMessageStartEvent struct {
	Message anthropicResponse `json:"message"`
}

type anthropicContentBlockStartEvent struct {
	Index        int              `json:"index"`
	ContentBlock anthropicContent `json:"content_block"`
}

type anthropicContentBlockDeltaEvent struct {
	Index int               `json:"index"`
	Delta anthropicDelta    `json:"delta"`
}

type anthropicDelta struct {
	Type         string `json:"type"`
	Text         string `json:"text,omitempty"`
	Thinking     string `json:"thinking,omitempty"`
	PartialJSON  string `json:"partial_json,omitempty"`
}

type anthropicContentBlockStopEvent struct {
	Index int `json:"index"`
}

type anthropicMessageDeltaEvent struct {
	Delta struct {
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Usage anthropicUsage `json:"usage"`
}

type anthropicErrorEvent struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *AnthropicClient) toChatResponse(resp *anthropicResponse) *ChatResponse {
	result := &ChatResponse{
		ID:         resp.ID,
		Model:      resp.Model,
		StopReason: resp.StopReason,
		Usage: Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}

	var blocks []ContentBlock
	for _, content := range resp.Content {
		block := ContentBlock{Type: content.Type}
		switch content.Type {
		case "text":
			block.Text = content.Text
		case "tool_use":
			block.ID = content.ID
			block.Name = content.Name
			block.Input = content.Input
		}
		blocks = append(blocks, block)
	}

	result.Messages = []Message{{Role: "assistant", Content: blocks}}
	return result
}

func (c *AnthropicClient) parseError(resp *http.Response, reqBody []byte) error {
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))

	var errResp struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error.Message != "" {
		return &APIError{
			Type:    errResp.Error.Type,
			Message: fmt.Sprintf("%s (req: %s)", errResp.Error.Message, truncateBytes(reqBody, 200)),
			Status:  resp.StatusCode,
		}
	}

	return &APIError{
		Type:    fmt.Sprintf("http_%d", resp.StatusCode),
		Message: fmt.Sprintf("HTTP %d: %s (req: %s)", resp.StatusCode, string(respBody), truncateBytes(reqBody, 200)),
		Status:  resp.StatusCode,
	}
}

func truncateBytes(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}

func respStatus(errType string) int {
	switch {
	case strings.Contains(errType, "rate_limit"):
		return 429
	case strings.Contains(errType, "overloaded"):
		return 529
	case strings.Contains(errType, "authentication"):
		return 401
	case strings.Contains(errType, "permission"):
		return 403
	case strings.Contains(errType, "not_found"):
		return 404
	case strings.Contains(errType, "invalid_request"):
		return 400
	default:
		return 500
	}
}

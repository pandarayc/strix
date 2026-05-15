package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/raydraw/ergate/internal/llm"
	"github.com/raydraw/ergate/internal/tool"
)

const agentSchema = `{
  "type": "object",
  "properties": {
    "description": {"type": "string", "description": "Short description of what this sub-agent should do"},
    "prompt":      {"type": "string", "description": "The full prompt/task for the sub-agent"},
    "model":       {"type": "string", "description": "Optional model override"}
  },
  "required": ["description", "prompt"]
}`

// AgentTool lets the model spawn sub-agents.
type AgentTool struct {
	reg    *Registry
	client llm.LLMClient
	model  string
	tools  *tool.Registry
}

// NewAgentTool creates a sub-agent tool.
func NewAgentTool(reg *Registry, client llm.LLMClient, model string, tools *tool.Registry) *AgentTool {
	return &AgentTool{reg: reg, client: client, model: model, tools: tools}
}

func (t *AgentTool) Name() string                { return "Agent" }
func (t *AgentTool) Description() string         { return "Spawn a sub-agent to work on a task in the background. The sub-agent has access to Read/Grep/Glob/WebSearch tools." }
func (t *AgentTool) InputSchema() json.RawMessage { return json.RawMessage(agentSchema) }
func (t *AgentTool) IsEnabled() bool             { return true }
func (t *AgentTool) IsReadOnly(input json.RawMessage) bool { return false }
func (t *AgentTool) IsConcurrencySafe() bool     { return true }

func (t *AgentTool) ValidateInput(ctx context.Context, input json.RawMessage) *tool.ValidationResult {
	return &tool.ValidationResult{Valid: true}
}

func (t *AgentTool) CheckPermissions(ctx context.Context, input json.RawMessage, permCtx tool.PermissionContext) tool.PermissionResult {
	return tool.AllowAll(input)
}

func (t *AgentTool) Execute(ctx context.Context, input json.RawMessage, execCtx *tool.ExecContext) (*tool.ToolResult, error) {
	var in struct {
		Description string `json:"description"`
		Prompt      string `json:"prompt"`
		Model       string `json:"model"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return &tool.ToolResult{Success: false, Content: fmt.Sprintf("Invalid input: %v", err)}, nil
	}

	model := t.model
	if in.Model != "" {
		model = in.Model
	}

	taskID := t.reg.Register(TypeLocalAgent, in.Description)
	t.reg.SetStatus(taskID, StatusRunning)

	// Run sub-agent in background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.reg.SetStatus(taskID, StatusFailed)
			}
		}()

		result := t.runAgent(in.Prompt, model)
		os.WriteFile("/tmp/ergate_task_"+taskID+".out", []byte(result), 0o644)

		if result != "" {
			t.reg.SetStatus(taskID, StatusCompleted)
		} else {
			t.reg.SetStatus(taskID, StatusFailed)
		}
	}()

	return &tool.ToolResult{
		Success: true,
		Content: fmt.Sprintf("Sub-agent started: %s (ID: %s)", in.Description, taskID),
		Metadata: map[string]any{
			"task_id":     taskID,
			"task_type":   "local_agent",
		},
	}, nil
}

func (t *AgentTool) runAgent(prompt, model string) string {
	ctx := context.Background()
	req := &llm.ChatRequest{
		Model:     model,
		System:    "You are a focused sub-agent. Complete the task and return results concisely. Use available tools to read and search. Do not write or edit files.",
		Messages:  []llm.Message{llm.NewUserMessage(prompt)},
		MaxTokens: 4096,
	}

	// Filter to read-only tools for sub-agents
	toolConfigs := []llm.ToolConfig{}
	readOnlyNames := map[string]bool{"Read": true, "Grep": true, "Glob": true, "WebSearch": true, "WebFetch": true}
	for _, t := range t.tools.List() {
		if readOnlyNames[t.Name()] && t.IsEnabled() {
			toolConfigs = append(toolConfigs, llm.ToolConfig{
				Name:        t.Name(),
				Description: t.Description(),
				InputSchema: t.InputSchema(),
			})
		}
	}
	req.Tools = toolConfigs

	// Two-turn agent: one call
	resp, err := t.client.Chat(ctx, req)
	if err != nil {
		return fmt.Sprintf("Sub-agent error: %v", err)
	}

	var result string
	for _, msg := range resp.Messages {
		for _, block := range msg.Content {
			if block.Type == "text" {
				result += block.Text
			}
		}
	}
	return result
}

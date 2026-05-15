package planmode

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/raydraw/ergate/internal/tool"
)

const enterPlanSchema = `{
  "type": "object",
  "properties": {},
  "required": []
}`

const exitPlanSchema = `{
  "type": "object",
  "properties": {},
  "required": []
}`

// EnterPlanTool lets the model enter plan mode.
type EnterPlanTool struct {
	mgr *Manager
}

func NewEnterPlanTool(mgr *Manager) *EnterPlanTool { return &EnterPlanTool{mgr: mgr} }
func (t *EnterPlanTool) Name() string              { return "EnterPlanMode" }
func (t *EnterPlanTool) Description() string       { return "Enter plan mode to explore and design before implementing. Only read-only tools are allowed." }
func (t *EnterPlanTool) InputSchema() json.RawMessage { return json.RawMessage(enterPlanSchema) }
func (t *EnterPlanTool) IsEnabled() bool           { return true }
func (t *EnterPlanTool) IsReadOnly(input json.RawMessage) bool { return true }
func (t *EnterPlanTool) IsConcurrencySafe() bool   { return true }

func (t *EnterPlanTool) ValidateInput(ctx context.Context, input json.RawMessage) *tool.ValidationResult {
	return &tool.ValidationResult{Valid: true}
}

func (t *EnterPlanTool) CheckPermissions(ctx context.Context, input json.RawMessage, permCtx tool.PermissionContext) tool.PermissionResult {
	return tool.AllowAll(input)
}

func (t *EnterPlanTool) Execute(ctx context.Context, input json.RawMessage, execCtx *tool.ExecContext) (*tool.ToolResult, error) {
	if t.mgr.InPlanMode() {
		return &tool.ToolResult{Success: true, Content: "Already in plan mode."}, nil
	}
	t.mgr.EnterPlan()
	return &tool.ToolResult{
		Success: true,
		Content: fmt.Sprintf("Entered plan mode.\n%s", PlanSystemPrompt()),
	}, nil
}

// ExitPlanTool lets the model exit plan mode and start implementing.
type ExitPlanTool struct {
	mgr *Manager
}

func NewExitPlanTool(mgr *Manager) *ExitPlanTool { return &ExitPlanTool{mgr: mgr} }
func (t *ExitPlanTool) Name() string              { return "ExitPlanMode" }
func (t *ExitPlanTool) Description() string       { return "Exit plan mode to begin implementation. Present your plan first." }
func (t *ExitPlanTool) InputSchema() json.RawMessage { return json.RawMessage(exitPlanSchema) }
func (t *ExitPlanTool) IsEnabled() bool           { return t.mgr.InPlanMode() }
func (t *ExitPlanTool) IsReadOnly(input json.RawMessage) bool { return false }
func (t *ExitPlanTool) IsConcurrencySafe() bool   { return true }

func (t *ExitPlanTool) ValidateInput(ctx context.Context, input json.RawMessage) *tool.ValidationResult {
	return &tool.ValidationResult{Valid: true}
}

func (t *ExitPlanTool) CheckPermissions(ctx context.Context, input json.RawMessage, permCtx tool.PermissionContext) tool.PermissionResult {
	return tool.AllowAll(input)
}

func (t *ExitPlanTool) Execute(ctx context.Context, input json.RawMessage, execCtx *tool.ExecContext) (*tool.ToolResult, error) {
	if !t.mgr.InPlanMode() {
		return &tool.ToolResult{Success: true, Content: "Not in plan mode."}, nil
	}
	t.mgr.ExitPlan()
	return &tool.ToolResult{
		Success: true,
		Content: "Exited plan mode. You may now use all tools to implement the plan.",
	}, nil
}

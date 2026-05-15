package skill

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/raydraw/ergate/internal/tool"
)

const loadSkillSchema = `{
  "type": "object",
  "properties": {
    "skill": {
      "type": "string",
      "description": "The skill name to load"
    }
  },
  "required": ["skill"]
}`

// LoadSkillTool lets the model load a skill on demand.
type LoadSkillTool struct {
	registry *Registry
}

// NewLoadSkillTool creates a new load_skill tool.
func NewLoadSkillTool(reg *Registry) *LoadSkillTool {
	return &LoadSkillTool{registry: reg}
}

func (t *LoadSkillTool) Name() string             { return "Skill" }
func (t *LoadSkillTool) Description() string      { return "Load a skill to get detailed instructions for a specific task type." }
func (t *LoadSkillTool) InputSchema() json.RawMessage { return json.RawMessage(loadSkillSchema) }
func (t *LoadSkillTool) IsEnabled() bool          { return true }
func (t *LoadSkillTool) IsReadOnly(input json.RawMessage) bool { return true }
func (t *LoadSkillTool) IsConcurrencySafe() bool  { return true }

func (t *LoadSkillTool) ValidateInput(ctx context.Context, input json.RawMessage) *tool.ValidationResult {
	return &tool.ValidationResult{Valid: true}
}

func (t *LoadSkillTool) CheckPermissions(ctx context.Context, input json.RawMessage, permCtx tool.PermissionContext) tool.PermissionResult {
	return tool.AllowAll(input)
}

func (t *LoadSkillTool) Execute(ctx context.Context, input json.RawMessage, exec *tool.ExecContext) (*tool.ToolResult, error) {
	var in struct {
		Skill string `json:"skill"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return &tool.ToolResult{Success: false, Content: fmt.Sprintf("Invalid input: %v", err)}, nil
	}

	skill, ok := t.registry.Get(in.Skill)
	if !ok {
		available := t.registry.Descriptions()
		return &tool.ToolResult{
			Success: false,
			Content: fmt.Sprintf("Skill %q not found.\nAvailable skills:\n%s", in.Skill, available),
		}, nil
	}

	content := fmt.Sprintf("# %s\n\n%s\n\n%s", skill.Name, skill.Description, skill.Body)
	return &tool.ToolResult{
		Success: true,
		Content: content,
		Metadata: map[string]any{
			"skill_name": skill.Name,
			"skill_path": skill.Path,
		},
	}, nil
}

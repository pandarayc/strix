package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/raydraw/ergate/internal/tool"
)

const writeMemorySchema = `{
  "type": "object",
  "properties": {
    "name":      {"type": "string", "description": "Short kebab-case slug for the memory file"},
    "type":      {"type": "string", "enum": ["user", "feedback", "project", "reference"]},
    "content":   {"type": "string", "description": "Memory content (markdown)"},
    "description": {"type": "string", "description": "One-line summary"}
  },
  "required": ["name", "type", "content"]
}`

// WriteTool lets the model save memories to disk.
type WriteTool struct {
	memoryDir string
	onWrite   func(path string) // called on successful write
}

// NewWriteTool creates a memory write tool.
func NewWriteTool(dir string, onWrite func(path string)) *WriteTool {
	return &WriteTool{memoryDir: dir, onWrite: onWrite}
}

func (t *WriteTool) Name() string             { return "SaveMemory" }
func (t *WriteTool) Description() string      { return "Save a memory to project memory for future sessions. Use for important decisions, user preferences, or learned patterns." }
func (t *WriteTool) InputSchema() json.RawMessage { return json.RawMessage(writeMemorySchema) }
func (t *WriteTool) IsEnabled() bool          { return true }
func (t *WriteTool) IsReadOnly(input json.RawMessage) bool { return false }
func (t *WriteTool) IsConcurrencySafe() bool  { return true }

func (t *WriteTool) ValidateInput(ctx context.Context, input json.RawMessage) *tool.ValidationResult {
	return &tool.ValidationResult{Valid: true}
}

func (t *WriteTool) CheckPermissions(ctx context.Context, input json.RawMessage, permCtx tool.PermissionContext) tool.PermissionResult {
	return tool.AllowAll(input)
}

func (t *WriteTool) Execute(ctx context.Context, input json.RawMessage, execCtx *tool.ExecContext) (*tool.ToolResult, error) {
	var in struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Content     string `json:"content"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return &tool.ToolResult{Success: false, Content: fmt.Sprintf("Invalid input: %v", err)}, nil
	}

	os.MkdirAll(t.memoryDir, 0o700)

	// Build frontmatter
	var fm strings.Builder
	fm.WriteString("---\n")
	fm.WriteString(fmt.Sprintf("name: %s\n", in.Name))
	fm.WriteString(fmt.Sprintf("description: %s\n", in.Description))
	fm.WriteString("metadata:\n")
	fmt.Fprintf(&fm, "  type: %s\n", in.Type)
	fm.WriteString("---\n\n")
	fm.WriteString(in.Content)

	fname := filepath.Join(t.memoryDir, in.Name+".md")
	if err := os.WriteFile(fname, []byte(fm.String()), 0o644); err != nil {
		return &tool.ToolResult{Success: false, Content: fmt.Sprintf("Failed to write memory: %v", err)}, nil
	}

	if t.onWrite != nil {
		t.onWrite(fname)
	}

	return &tool.ToolResult{
		Success: true,
		Content: fmt.Sprintf("Memory saved: %s (%d bytes)", fname, len(in.Content)),
		Metadata: map[string]any{
			"memory_path": fname,
			"memory_name": in.Name,
		},
	}, nil
}

// UpdateMEMORYMD appends a reference to a new memory file in MEMORY.md.
func UpdateMEMORYMD(memoryDir, filename string) error {
	memFile := filepath.Join(memoryDir, "MEMORY.md")
	f, err := os.OpenFile(memFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	entry := fmt.Sprintf("- [%s](%s)\n", strings.TrimSuffix(filename, ".md"), filename)
	_, err = f.WriteString(entry)
	return err
}

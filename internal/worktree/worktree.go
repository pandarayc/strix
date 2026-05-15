package worktree

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/raydraw/ergate/internal/tool"
)

// Manager tracks git worktrees.
type Manager struct {
	mu         sync.Mutex
	active     string            // current worktree path
	original   string            // original working directory
	worktrees  map[string]string // name -> path
}

// NewManager creates a worktree manager.
func NewManager() *Manager {
	cwd, _ := os.Getwd()
	return &Manager{
		original:  cwd,
		worktrees: make(map[string]string),
	}
}

// IsGitRepo checks if the current directory is in a git repo.
func IsGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	return cmd.Run() == nil
}

// Create creates a new worktree on a new branch.
func (m *Manager) Create(name string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !IsGitRepo(m.original) {
		return "", fmt.Errorf("not in a git repository")
	}

	baseRef := "HEAD"
	// Try origin/main first, fall back to HEAD
	cmd := exec.Command("git", "rev-parse", "--verify", "origin/main")
	cmd.Dir = m.original
	if cmd.Run() == nil {
		baseRef = "origin/main"
	}

	branchName := "ergate/" + name
	wtPath := filepath.Join(m.original, ".ergate", "worktrees", name)

	// Create the worktree
	cmd = exec.Command("git", "worktree", "add", "-b", branchName, wtPath, baseRef)
	cmd.Dir = m.original
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git worktree add: %v\n%s", err, string(out))
	}

	m.worktrees[name] = wtPath
	m.active = wtPath
	return wtPath, nil
}

// Remove removes a worktree and its branch.
func (m *Manager) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	wtPath, ok := m.worktrees[name]
	if !ok {
		return fmt.Errorf("worktree %q not found", name)
	}

	// Remove the worktree
	cmd := exec.Command("git", "worktree", "remove", wtPath, "--force")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove: %v\n%s", err, string(out))
	}

	// Delete the branch
	branchName := "ergate/" + name
	cmd = exec.Command("git", "branch", "-D", branchName)
	cmd.Dir = m.original
	cmd.Run() // ignore errors (branch might already be gone)

	delete(m.worktrees, name)
	if m.active == wtPath {
		m.active = m.original
	}
	return nil
}

// Active returns the current active worktree path.
func (m *Manager) Active() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active != "" {
		return m.active
	}
	return m.original
}

// List returns all worktree names.
func (m *Manager) List() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.worktrees))
	for name := range m.worktrees {
		names = append(names, name)
	}
	return names
}

// EnterWorktreeTool lets the model create an isolated worktree.
type EnterWorktreeTool struct {
	mgr *Manager
}

func NewEnterWorktreeTool(mgr *Manager) *EnterWorktreeTool { return &EnterWorktreeTool{mgr: mgr} }

const enterWTSchema = `{
  "type": "object",
  "properties": {
    "name": {"type": "string", "description": "Short name for the worktree (alphanumeric)"}
  },
  "required": ["name"]
}`

func (t *EnterWorktreeTool) Name() string              { return "EnterWorktree" }
func (t *EnterWorktreeTool) Description() string       { return "Create an isolated git worktree for experimental changes. Use before making complex edits." }
func (t *EnterWorktreeTool) InputSchema() json.RawMessage   { return json.RawMessage(enterWTSchema) }
func (t *EnterWorktreeTool) IsEnabled() bool               { return IsGitRepo(t.mgr.original) }
func (t *EnterWorktreeTool) IsReadOnly(input json.RawMessage) bool { return false }
func (t *EnterWorktreeTool) IsConcurrencySafe() bool       { return false }

func (t *EnterWorktreeTool) ValidateInput(ctx context.Context, input json.RawMessage) *tool.ValidationResult {
	return &tool.ValidationResult{Valid: true}
}

func (t *EnterWorktreeTool) CheckPermissions(ctx context.Context, input json.RawMessage, permCtx tool.PermissionContext) tool.PermissionResult {
	return tool.AllowAll(input)
}

func (t *EnterWorktreeTool) Execute(ctx context.Context, input json.RawMessage, execCtx *tool.ExecContext) (*tool.ToolResult, error) {
	// Simple JSON parsing since we use []byte
	var in struct {
		Name string `json:"name"`
	}
	json.Unmarshal(input, &in)
	if in.Name == "" {
		return &tool.ToolResult{Success: false, Content: "name is required"}, nil
	}

	path, err := t.mgr.Create(in.Name)
	if err != nil {
		return &tool.ToolResult{Success: false, Content: fmt.Sprintf("Failed to create worktree: %v", err)}, nil
	}

	return &tool.ToolResult{
		Success: true,
		Content: fmt.Sprintf("Worktree created: %s\nBranch: ergate/%s\nUse ExitWorktree to clean up when done.", path, in.Name),
		Metadata: map[string]any{"worktree_path": path, "worktree_name": in.Name},
	}, nil
}

// ExitWorktreeTool lets the model clean up a worktree.
type ExitWorktreeTool struct {
	mgr *Manager
}

func NewExitWorktreeTool(mgr *Manager) *ExitWorktreeTool { return &ExitWorktreeTool{mgr: mgr} }

const exitWTSchema = `{
  "type": "object",
  "properties": {
    "name": {"type": "string", "description": "Worktree name to remove"}
  },
  "required": ["name"]
}`

func (t *ExitWorktreeTool) Name() string              { return "ExitWorktree" }
func (t *ExitWorktreeTool) Description() string       { return "Remove a worktree and delete its branch. Use after changes are committed." }
func (t *ExitWorktreeTool) InputSchema() json.RawMessage   { return json.RawMessage(exitWTSchema) }
func (t *ExitWorktreeTool) IsEnabled() bool               { return len(t.mgr.List()) > 0 }
func (t *ExitWorktreeTool) IsReadOnly(input json.RawMessage) bool { return false }
func (t *ExitWorktreeTool) IsConcurrencySafe() bool       { return false }

func (t *ExitWorktreeTool) ValidateInput(ctx context.Context, input json.RawMessage) *tool.ValidationResult {
	return &tool.ValidationResult{Valid: true}
}

func (t *ExitWorktreeTool) CheckPermissions(ctx context.Context, input json.RawMessage, permCtx tool.PermissionContext) tool.PermissionResult {
	return tool.AllowAll(input)
}

func (t *ExitWorktreeTool) Execute(ctx context.Context, input json.RawMessage, execCtx *tool.ExecContext) (*tool.ToolResult, error) {
	var in struct {
		Name string `json:"name"`
	}
	json.Unmarshal(input, &in)
	if in.Name == "" {
		return &tool.ToolResult{Success: false, Content: "name is required"}, nil
	}

	if err := t.mgr.Remove(in.Name); err != nil {
		return &tool.ToolResult{Success: false, Content: fmt.Sprintf("Failed to remove worktree: %v", err)}, nil
	}

	return &tool.ToolResult{
		Success: true,
		Content: fmt.Sprintf("Worktree %q removed.", in.Name),
	}, nil
}

package hooks

import (
	"context"
	"encoding/json"
	"sync"
)

// Event marks the point at which a hook fires.
type Event string

const (
	PreToolUse  Event = "preToolUse"
	PostToolUse Event = "postToolUse"
	OnStop      Event = "onStop"
)

// Data carries context about the hook invocation.
type Data struct {
	ToolName string          `json:"tool_name"`
	Input    json.RawMessage `json:"input"`
	Output   string          `json:"output,omitempty"`  // PostToolUse only
	IsError  bool            `json:"is_error,omitempty"` // PostToolUse only
	Duration int64           `json:"duration_ms,omitempty"`
}

// Result controls what happens after the hook runs.
type Result struct {
	Continue bool   // false blocks the operation
	Message  string // feedback to the model
}

// Hook is the interface for individual hooks.
type Hook interface {
	Name() string
	Run(ctx context.Context, event Event, data Data) (Result, error)
}

// Manager registers and dispatches hooks.
type Manager struct {
	mu    sync.RWMutex
	hooks []Hook
}

// NewManager creates a new hook manager.
func NewManager() *Manager {
	return &Manager{}
}

// Register adds a hook to the manager.
func (m *Manager) Register(h Hook) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks = append(m.hooks, h)
}

// Fire invokes all registered hooks for the given event.
// Returns the first blocking result or the combined messages.
func (m *Manager) Fire(ctx context.Context, event Event, data Data) (Result, error) {
	m.mu.RLock()
	hooks := make([]Hook, len(m.hooks))
	copy(hooks, m.hooks)
	m.mu.RUnlock()

	var messages []string
	for _, h := range hooks {
		result, err := h.Run(ctx, event, data)
		if err != nil {
			return Result{Continue: false, Message: err.Error()}, err
		}
		if !result.Continue {
			return result, nil
		}
		if result.Message != "" {
			messages = append(messages, "["+h.Name()+"] "+result.Message)
		}
	}

	msg := ""
	if len(messages) > 0 {
		for _, m := range messages {
			msg += m + "\n"
		}
	}
	return Result{Continue: true, Message: msg}, nil
}

// HasHooks returns true if any hooks are registered.
func (m *Manager) HasHooks() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.hooks) > 0
}

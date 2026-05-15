package task

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// Type identifies the kind of task.
type Type string

const (
	TypeLocalBash  Type = "local_bash"
	TypeLocalAgent Type = "local_agent"
)

// Status tracks the task lifecycle.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusKilled    Status = "killed"
)

// IsTerminal returns true for final states.
func IsTerminal(s Status) bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusKilled
}

// ID prefixes per task type.
var typePrefixes = map[Type]string{
	TypeLocalBash:  "b",
	TypeLocalAgent: "a",
}

const idAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"

func generateID(t Type) string {
	b := make([]byte, 8)
	rand.Read(b)
	for i, v := range b {
		b[i] = idAlphabet[int(v)%len(idAlphabet)]
	}
	return typePrefixes[t] + string(b)
}

// State holds the base fields shared by all tasks.
type State struct {
	ID          string    `json:"id"`
	Type        Type      `json:"type"`
	Status      Status    `json:"status"`
	Description string    `json:"description"`
	ToolUseID   string    `json:"tool_use_id,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time,omitempty"`
	outputFile  string
	notified    bool
}

// BashTaskState extends State for shell tasks.
type BashTaskState struct {
	State
	Command string `json:"command"`
	Result  *BashResult `json:"result,omitempty"`
}

// BashResult holds the outcome of a bash task.
type BashResult struct {
	ExitCode    int    `json:"exit_code"`
	Stdout      string `json:"stdout"`
	Stderr      string `json:"stderr"`
	Interrupted bool   `json:"interrupted"`
}

// AgentTaskState extends State for agent tasks.
type AgentTaskState struct {
	State
	Prompt string `json:"prompt"`
	Model  string `json:"model,omitempty"`
	Result string `json:"result,omitempty"`
}

// Task is the interface for type-specific kill logic.
type Task interface {
	Kill() error
}

// Notification is sent when a task completes.
type Notification struct {
	TaskID      string
	Type        Type
	Description string
	Status      Status
	Output      string
}

// Registry manages all active tasks.
type Registry struct {
	mu       sync.RWMutex
	tasks    map[string]*State
	notifyCh chan Notification
}

// NewRegistry creates a new task registry.
func NewRegistry() *Registry {
	return &Registry{
		tasks:    make(map[string]*State),
		notifyCh: make(chan Notification, 64),
	}
}

// NotifyChan returns the notification channel.
func (r *Registry) NotifyChan() <-chan Notification {
	return r.notifyCh
}

// Register adds a task with generated ID.
func (r *Registry) Register(t Type, description string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	state := &State{
		ID:          generateID(t),
		Type:        t,
		Status:      StatusPending,
		Description: description,
		StartTime:   time.Now(),
	}
	r.tasks[state.ID] = state
	return state.ID
}

// Get returns a task by ID.
func (r *Registry) Get(id string) (*State, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tasks[id]
	return t, ok
}

// SetStatus updates the task status and sends notification on terminal state.
func (r *Registry) SetStatus(id string, status Status) {
	r.mu.Lock()
	t, ok := r.tasks[id]
	if !ok {
		r.mu.Unlock()
		return
	}
	t.Status = status
	if IsTerminal(status) {
		t.EndTime = time.Now()
	}
	desc := t.Description
	taskType := t.Type
	r.mu.Unlock()

	if IsTerminal(status) {
		select {
		case r.notifyCh <- Notification{
			TaskID:      id,
			Type:        taskType,
			Description: desc,
			Status:      status,
		}:
		default:
		}
	}
}

// List returns all tasks.
func (r *Registry) List() []State {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]State, 0, len(r.tasks))
	for _, t := range r.tasks {
		result = append(result, *t)
	}
	return result
}

// EvictTerminal removes completed/failed/killed tasks.
func (r *Registry) EvictTerminal() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, t := range r.tasks {
		if IsTerminal(t.Status) {
			delete(r.tasks, id)
		}
	}
}

// FormatList returns a human-readable task listing.
func (r *Registry) FormatList() string {
	tasks := r.List()
	if len(tasks) == 0 {
		return "No active tasks."
	}
	var result string
	for _, t := range tasks {
		result += fmt.Sprintf("[%s] %s — %s (%s)\n", t.ID, t.Type, t.Description, t.Status)
	}
	return result
}

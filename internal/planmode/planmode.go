package planmode

// State tracks whether the engine is in plan or implement mode.
type State int

const (
	StateNormal State = iota
	StatePlan
)

// Manager maintains the plan/implement state machine.
type Manager struct {
	state State
}

// NewManager creates a plan mode manager.
func NewManager() *Manager {
	return &Manager{state: StateNormal}
}

// EnterPlan transitions to plan mode. No-op if already in plan mode.
func (m *Manager) EnterPlan() {
	m.state = StatePlan
}

// ExitPlan transitions to normal mode. No-op if already normal.
func (m *Manager) ExitPlan() {
	m.state = StateNormal
}

// InPlanMode returns true when in plan mode.
func (m *Manager) InPlanMode() bool {
	return m.state == StatePlan
}

// PlanSystemPrompt returns the system prompt extension for plan mode.
func PlanSystemPrompt() string {
	return `

## PLAN MODE

You are in PLAN MODE. In this mode:
- You may ONLY use read-only tools (Read, Grep, Glob, WebSearch)
- You may NOT use write tools (Write, Edit) or shell commands (Bash)
- Your goal is to explore the codebase and design an implementation plan
- Produce a clear, structured plan before asking to exit plan mode
- When ready, use the ExitPlanMode tool to request plan approval

Follow the design phase carefully. Do not implement anything yet.`
}

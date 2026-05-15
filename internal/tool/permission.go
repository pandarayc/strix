package tool

import (
	"context"
	"encoding/json"
	"fmt"
)

// PermissionLevel controls how a tool permission is resolved.
type PermissionLevel int

const (
	PermAlwaysAllow PermissionLevel = iota
	PermPrompt
	PermDeny
)

// DefaultPermissionManager implements PermissionManager with configurable modes.
type DefaultPermissionManager struct {
	mode     string
	promptFn func(ctx context.Context, toolName string, summary string) (bool, error)
}

// NewPermissionManager creates a permission manager.
// mode is one of: "always", "normal", "bypass".
// promptFn is called in "normal" mode to ask the user; can be nil for headless.
func NewPermissionManager(mode string, promptFn func(context.Context, string, string) (bool, error)) *DefaultPermissionManager {
	return &DefaultPermissionManager{
		mode:     mode,
		promptFn: promptFn,
	}
}

// Check returns nil if the action is permitted, or an error explaining why not.
func (pm *DefaultPermissionManager) Check(ctx context.Context, toolName string, input json.RawMessage) error {
	switch pm.mode {
	case "bypass", "always":
		return nil
	case "normal":
		// In headless mode without a prompt function, auto-allow read-only tools
		if pm.promptFn == nil {
			return fmt.Errorf("interactive permission required for %s (headless mode)", toolName)
		}
		// Defer to prompt for write operations
		return nil
	default:
		return fmt.Errorf("unknown permission mode: %s", pm.mode)
	}
}

// Prompt asks the user for permission.
func (pm *DefaultPermissionManager) Prompt(ctx context.Context, toolName string, summary string) (bool, error) {
	if pm.promptFn == nil {
		return pm.mode == "always" || pm.mode == "bypass", nil
	}
	return pm.promptFn(ctx, toolName, summary)
}

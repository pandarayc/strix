package cli

import (
	"fmt"
	"sort"
	"strings"
)

// CommandType indicates how a command executes.
type CommandType int

const (
	// CommandLocal runs synchronously in-process and returns text output.
	CommandLocal CommandType = iota
	// CommandPrompt expands to text sent to the model (not yet used).
	CommandPrompt
)

// CommandContext holds state available to command handlers.
type CommandContext struct {
	Config    interface{}
	Engine    interface{}
	ModelName string
}

// Command is a slash command definition.
type Command struct {
	Name        string
	Aliases     []string
	Description string
	Type        CommandType
	// ArgumentHint is shown after the command name in help.
	ArgumentHint string
	// IsEnabled returns false to hide the command.
	IsEnabled func() bool
	// IsHidden hides from help listing.
	IsHidden bool
	// Call handles the command. Returns output text and an optional quit signal.
	Call func(args []string, ctx *CommandContext) (string, bool)
}

// CommandRegistry holds and dispatches commands.
type CommandRegistry struct {
	commands map[string]*Command
	names    []string // insertion order
}

// Register adds a command, including its aliases.
func (r *CommandRegistry) Register(cmd *Command) {
	r.commands[cmd.Name] = cmd
	r.names = append(r.names, cmd.Name)
	for _, alias := range cmd.Aliases {
		if _, exists := r.commands[alias]; !exists {
			r.commands[alias] = cmd
		}
	}
}

// Find looks up a command by name or alias.
func (r *CommandRegistry) Find(name string) (*Command, bool) {
	cmd, ok := r.commands[name]
	return cmd, ok
}

// Dispatch runs the command and returns (output, shouldQuit, found).
func (r *CommandRegistry) Dispatch(name string, args []string, ctx *CommandContext) (string, bool, bool) {
	cmd, ok := r.Find(name)
	if !ok {
		return "", false, false
	}
	if cmd.IsEnabled != nil && !cmd.IsEnabled() {
		return "", false, false
	}
	out, quit := cmd.Call(args, ctx)
	return out, quit, true
}

// Help returns formatted help text for all visible commands.
func (r *CommandRegistry) Help() string {
	var b strings.Builder
	b.WriteString("Available commands:\n")

	names := make([]string, 0)
	for _, name := range r.names {
		cmd := r.commands[name]
		if cmd.IsHidden {
			continue
		}
		if cmd.IsEnabled != nil && !cmd.IsEnabled() {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		cmd := r.commands[name]
		line := fmt.Sprintf("  /%s", name)
		if cmd.ArgumentHint != "" {
			line += " " + cmd.ArgumentHint
		}
		line += " — " + cmd.Description
		b.WriteString(line + "\n")
	}
	return b.String()
}

// handleREPLCommand processes input that starts with "/".
// Returns (output, shouldQuit, wasHandled).
func (r *CommandRegistry) HandleREPL(input string, ctx *CommandContext) (string, bool, bool) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", false, false
	}
	name := strings.TrimPrefix(parts[0], "/")
	return r.Dispatch(name, parts[1:], ctx)
}

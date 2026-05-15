package cli

import "github.com/raydraw/ergate/internal/engine"

func ClearCmd(eng *engine.Engine) *Command {
	return &Command{
		Name:        "clear",
		Description: "Clear conversation history",
		Type:        CommandLocal,
		Call: func(args []string, ctx *CommandContext) (string, bool) {
			eng.Clear()
			return "Conversation cleared.", false
		},
	}
}

package cli

import (
	"fmt"

	"github.com/raydraw/ergate/internal/engine"
)

func UsageCmd(eng *engine.Engine) *Command {
	return &Command{
		Name:        "usage",
		Description: "Show token usage statistics",
		Type:        CommandLocal,
		Call: func(args []string, ctx *CommandContext) (string, bool) {
			in, out := eng.TotalUsage()
			return fmt.Sprintf("Input tokens:  %d\nOutput tokens: %d\nTotal tokens:  %d", in, out, in+out), false
		},
	}
}

package cli

import (
	"fmt"

	"github.com/raydraw/ergate/internal/config"
	"github.com/raydraw/ergate/internal/engine"
)

func StatusCmd(cfg *config.Config, eng *engine.Engine) *Command {
	return &Command{
		Name:        "status",
		Description: "Show engine status",
		Type:        CommandLocal,
		Call: func(args []string, ctx *CommandContext) (string, bool) {
			msgs := eng.Messages()
			in, out := eng.TotalUsage()
			return fmt.Sprintf("Model: %s  Messages: %d  Tokens (in:%d out:%d)",
				cfg.Model, len(msgs), in, out), false
		},
	}
}

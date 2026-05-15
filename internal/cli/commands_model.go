package cli

import "github.com/raydraw/ergate/internal/config"

func ModelCmd(cfg *config.Config) *Command {
	return &Command{
		Name:         "model",
		Description:  "Show or change the current model",
		Type:         CommandLocal,
		ArgumentHint: "[model]",
		Call: func(args []string, ctx *CommandContext) (string, bool) {
			if len(args) > 0 {
				cfg.Model = args[0]
				return "Model changed to: " + cfg.Model, false
			}
			return "Current model: " + cfg.Model, false
		},
	}
}

package cli

import (
	"fmt"

	"github.com/raydraw/ergate/internal/config"
)

func ConfigCmd(cfg *config.Config) *Command {
	return &Command{
		Name:        "config",
		Description: "Show current configuration",
		Type:        CommandLocal,
		Call: func(args []string, ctx *CommandContext) (string, bool) {
			return fmt.Sprintf("Provider:  %s\nModel:     %s\nBase URL:  %s\nMax turns: %d",
				cfg.APIProvider, cfg.Model, cfg.BaseURL, cfg.MaxTurns), false
		},
	}
}

package cli

import (
	"fmt"

	"github.com/raydraw/ergate/internal/engine"
)

func CostCmd(eng *engine.Engine) *Command {
	return &Command{
		Name:        "cost",
		Description: "Show estimated API cost",
		Type:        CommandLocal,
		Call: func(args []string, ctx *CommandContext) (string, bool) {
			in, out := eng.TotalUsage()
			cost := float64(in)/1e6*3.0 + float64(out)/1e6*15.0
			return fmt.Sprintf("Est. cost: $%.4f  (in:%d out:%d)", cost, in, out), false
		},
	}
}

package cli

import "github.com/raydraw/ergate/internal/tui"

func ThemeCmd() *Command {
	return &Command{
		Name:         "theme",
		Description:  "Switch theme (dark or light)",
		Type:         CommandLocal,
		ArgumentHint: "[dark|light]",
		Call: func(args []string, ctx *CommandContext) (string, bool) {
			if len(args) == 0 {
				return "Current theme: " + string(tui.CurrentTheme.Name), false
			}
			switch args[0] {
			case "light":
				tui.SetTheme(tui.ThemeLight)
				return "Theme changed to light.", false
			case "dark":
				tui.SetTheme(tui.ThemeDark)
				return "Theme changed to dark.", false
			default:
				return "Usage: /theme [dark|light]", false
			}
		},
	}
}

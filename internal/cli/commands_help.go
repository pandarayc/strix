package cli

func HelpCmd() *Command {
	return &Command{
		Name:        "help",
		Description: "Show help and available commands",
		Type:        CommandLocal,
		Call: func(args []string, ctx *CommandContext) (string, bool) {
			return "Available commands:\n" +
				"  /help /exit /quit /clear /model /usage /config\n" +
				"  /cost /status /save /load /sessions /version", false
		},
	}
}

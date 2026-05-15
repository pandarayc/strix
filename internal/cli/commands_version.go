package cli

func VersionCmd() *Command {
	return &Command{
		Name:        "version",
		Description: "Show Ergate version",
		Type:        CommandLocal,
		Call: func(args []string, ctx *CommandContext) (string, bool) {
			return "Ergate v0.1.0", false
		},
	}
}

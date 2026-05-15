package cli

func ExitCmd() *Command {
	return &Command{
		Name: "exit",
		Aliases: []string{"quit"},
		Description: "Exit Ergate",
		Type: CommandLocal,
		Call: func(args []string, ctx *CommandContext) (string, bool) {
			return "Goodbye!", true
		},
	}
}

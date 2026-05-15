package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/raydraw/ergate/internal/engine"
)

// RunOneShot executes a single prompt and prints the result to stdout.
func RunOneShot(eng *engine.Engine, prompt string) error {
	events := make(chan engine.Event, 128)
	ctx := context.Background()

	go func() {
		// eng.Run closes events internally, don't close again
		if err := eng.Run(ctx, prompt, events); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}()

	for event := range events {
		switch event.Type {
		case engine.EventText:
			if text, ok := event.Data.(string); ok {
				fmt.Print(text)
			}
		case engine.EventToolUse:
			if data, ok := event.Data.(map[string]any); ok {
				name, _ := data["name"].(string)
				fmt.Fprintf(os.Stderr, "[Tool: %s] ", name)
			}
		case engine.EventToolResult:
			if data, ok := event.Data.(map[string]any); ok {
				isErr, _ := data["is_error"].(bool)
				if isErr {
					content, _ := data["content"].(string)
					fmt.Fprintf(os.Stderr, "[Error: %s]\n", content)
				}
			}
		case engine.EventError:
			if err, ok := event.Data.(error); ok {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		case engine.EventDone:
			fmt.Println()
			return nil
		}
	}
	return nil
}

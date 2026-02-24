package shell

import (
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
)

// Start starts the interactive shell
func Start(rootCmd *cobra.Command) {
	fmt.Println("PikPak CLI Interactive Shell")
	fmt.Println("Type 'help' for available commands, 'exit' to quit")
	fmt.Println()

	// Create readline instance
	// TODO: we can add path here: pikpak {path} >.
	l, err := readline.New("pikpak > ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing readline: %v\n", err)
		return
	}
	defer l.Close()

	for {
		input, err := l.Readline()

		// Handle EOF (Ctrl+D)
		if err == readline.ErrInterrupt {
			continue
		}

		if err != nil {
			// This is EOF
			fmt.Println("\nBye~!")
			break
		}

		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Bye~!")
			break
		}

		if input == "help" {
			rootCmd.Help()
			continue
		}

		// Parse the args and set them to rootCmd
		args := parseShellArgs(input)
		rootCmd.SetArgs(args)

		// Directly use pre-defined Execute function
		// TODO: Need to be updated if cd command is supported.
		if err := rootCmd.Execute(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		rootCmd.SetArgs([]string{}) // Reset for next iteration
	}
}

// parseShellArgs parses shell-like arguments
func parseShellArgs(input string) []string {
	var args []string
	var current strings.Builder
	inDoubleQuote := false
	inSingleQuote := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		switch ch {
		case '"':
			if inSingleQuote {
				current.WriteByte(ch)
			} else {
				inDoubleQuote = !inDoubleQuote
			}
		case '\'':
			if inDoubleQuote {
				current.WriteByte(ch)
			} else {
				inSingleQuote = !inSingleQuote
			}
		case ' ', '\t':
			if inDoubleQuote || inSingleQuote {
				current.WriteByte(ch)
			} else {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

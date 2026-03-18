package shell

import (
	"fmt"
	"os"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Start starts the interactive shell
func Start(rootCmd *cobra.Command) {
	fmt.Println("PikPak CLI Interactive Shell")
	fmt.Println("Type 'help' for available commands, 'exit' to quit")
	fmt.Println()

	currentPath := "/"
	globalPath := "/"
	// globalPath := currentPath

	// Create readline instance
	// TODO: we can add path here: pikpak {path} >.
	l, err := readline.New(fmt.Sprintf("pikpak %s > ", currentPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing readline: %v\n", err)
		return
	}
	defer l.Close()

	for {
		input, err := l.Readline()

		// l.SetPrompt(fmt.Sprintf("pikpak %s > ", currentPath))

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

		// Handle cd command
		if len(args) > 0 && args[0] == "cd" {
			var path string
			if len(args) > 1 {
				path = args[1]
			}
			// Go back to root if target path is empty, ~ or /
			switch path {
			case "", "~", "/":
				currentPath = "/"
				globalPath = currentPath
			case "..":
				// Go back to parent directory
				if currentPath != "/" {
					currentPath = currentPath[:strings.LastIndex(currentPath, "/")]
					globalPath = currentPath + "/"
					if currentPath == "" {
						currentPath = "/"
						globalPath = currentPath
					}
				}

				// Update the prompt with the new path
				l.SetPrompt(fmt.Sprintf("pikpak %s > ", globalPath))
			default:
				// Handle relative and absolute paths
				var targetPath string
				if strings.HasPrefix(path, "/") {
					// Absolute path
					targetPath = path
				} else {
					// Relative path
					if currentPath == "/" {
						targetPath = "/" + path
					} else {
						targetPath = currentPath + "/" + path
					}
				}
				// Normalize path: remove duplicate slashes and trailing slash
				targetPath = strings.ReplaceAll(targetPath, "//", "/")
				if targetPath != "/" {
					targetPath = strings.TrimSuffix(targetPath, "/")
				}
				// Check if the path exists and is a directory
				p := pikpak.NewPikPak(conf.Config.Username, conf.Config.Password)
				err := p.Login()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
					continue
				}
				_, err = p.GetPathFolderId(targetPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "cd: %s: No such directory\n", targetPath)
					continue
				}
				currentPath = targetPath
				globalPath = currentPath + "/"
				// Update the prompt with the new path
				l.SetPrompt(fmt.Sprintf("pikpak %s > ", globalPath))
			}

			continue
		}

		// For ls command, if no path specified, add current path
		if len(args) == 1 && args[0] == "ls" {
			args = append(args, "-p", currentPath)
		}

		rootCmd.SetArgs(args)

		// Directly use pre-defined Execute function
		// TODO: Need to be updated if cd command is supported.
		if err := rootCmd.Execute(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		rootCmd.SetArgs([]string{}) // Reset for next iteration

		// Reset flags to default values to prevent state retention between commands
		resetFlags(rootCmd)
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

// resetFlags recursively resets all flags in the command tree to their default values
func resetFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue)
	})
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue)
	})
	for _, subCmd := range cmd.Commands() {
		resetFlags(subCmd)
	}
}

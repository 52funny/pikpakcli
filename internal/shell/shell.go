package shell

import (
	"fmt"
	"os"
	"path"
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

	p := pikpak.NewPikPak(conf.Config.Username, conf.Config.Password)
	if err := p.Login(); err != nil {
		fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
		return
	}

	l, err := readline.New(promptForPath(currentPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing readline: %v\n", err)
		return
	}
	defer l.Close()

	for {
		input, err := l.Readline()

		if err == readline.ErrInterrupt {
			continue
		}

		if err != nil {
			fmt.Println("\nBye~!")
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		switch input {
		case "exit", "quit":
			fmt.Println("Bye~!")
			return
		case "help":
			rootCmd.Help()
			continue
		}

		args := parseShellArgs(input)
		if len(args) == 0 {
			continue
		}

		if args[0] == "cd" {
			nextPath, err := changeDirectory(&p, currentPath, args[1:])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			currentPath = nextPath
			l.SetPrompt(promptForPath(currentPath))
			continue
		}

		if len(args) == 1 && args[0] == "ls" {
			args = append(args, "-p", currentPath)
		}

		rootCmd.SetArgs(args)
		if err := rootCmd.Execute(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		rootCmd.SetArgs([]string{})
		resetFlags(rootCmd)
	}
}

func promptForPath(currentPath string) string {
	if currentPath == "/" {
		return "pikpak / > "
	}
	return fmt.Sprintf("pikpak %s/ > ", currentPath)
}

func changeDirectory(p *pikpak.PikPak, currentPath string, args []string) (string, error) {
	target := "/"
	if len(args) > 0 {
		target = args[0]
	}

	targetPath := resolveShellPath(currentPath, target)
	if targetPath == "/" {
		return targetPath, nil
	}

	if _, err := p.GetPathFolderId(targetPath); err != nil {
		return "", fmt.Errorf("cd: %s: no such directory", targetPath)
	}

	return targetPath, nil
}

func resolveShellPath(currentPath string, target string) string {
	switch strings.TrimSpace(target) {
	case "", "~", "/":
		return "/"
	}

	if strings.HasPrefix(target, "/") {
		return path.Clean(target)
	}

	return path.Clean(path.Join(currentPath, target))
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
			} else if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
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

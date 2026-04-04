package shell

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"slices"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/logx"
	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var builtInCommands = []string{"cd", "exit", "help", "quit"}

type fileStatProvider interface {
	GetPathFolderId(dirPath string) (string, error)
	GetFolderFileStatList(parentId string) ([]api.FileStat, error)
}

type shellAutoCompleter struct {
	rootCmd        *cobra.Command
	fileStatSource fileStatProvider
	currentPath    func() string
}

// Start starts the interactive shell
func Start(rootCmd *cobra.Command) {
	fmt.Println("PikPak CLI Interactive Shell")
	fmt.Println("Type 'help' for available commands, 'exit' or Ctrl-D to quit")
	fmt.Println()

	currentPath := "/"

	p := api.NewPikPak(conf.Config.Username, conf.Config.Password)
	if err := p.Login(); err != nil {
		fmt.Println("Login failed")
		logx.Error(err)
		return
	}

	l, err := readline.NewEx(&readline.Config{
		Prompt: promptForPath(currentPath),
		AutoComplete: &shellAutoCompleter{
			rootCmd:        rootCmd,
			fileStatSource: &p,
			currentPath: func() string {
				return currentPath
			},
		},
	})
	if err != nil {
		fmt.Println("Initialize readline failed")
		logx.Error(err)
		return
	}
	defer l.Close()

	for {
		input, err := l.Readline()

		if isReadlineInterrupt(err) {
			fmt.Println()
			l.SetPrompt(promptForPath(currentPath))
			continue
		}

		if shouldExitOnReadlineError(err) {
			fmt.Println("\nBye~!")
			return
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
				fmt.Println("Change directory failed")
				logx.Error(err)
				continue
			}
			currentPath = nextPath
			l.SetPrompt(promptForPath(currentPath))
			continue
		}

		if len(args) == 1 && args[0] == "ls" {
			args = append(args, "-p", currentPath)
		}

		cmdCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		setCommandContextTree(rootCmd, cmdCtx)
		rootCmd.SetArgs(args)
		rootCmd.Execute()
		stop()
		setCommandContextTree(rootCmd, context.Background())
		rootCmd.SetArgs([]string{})
		resetFlags(rootCmd)
		if cmdCtx.Err() != nil {
			fmt.Println()
		}
	}
}

func shouldExitOnReadlineError(err error) bool {
	return err == io.EOF
}

func isReadlineInterrupt(err error) bool {
	return err == readline.ErrInterrupt
}

func setCommandContextTree(cmd *cobra.Command, ctx context.Context) {
	cmd.SetContext(ctx)
	for _, child := range cmd.Commands() {
		setCommandContextTree(child, ctx)
	}
}

func (c *shellAutoCompleter) Do(line []rune, pos int) ([][]rune, int) {
	input := string(line[:pos])
	tokens, active, endedWithSpace := splitCompletionLine(input)

	if len(tokens) == 0 {
		return completeFromPrefix(active, commandCandidates(c.rootCmd), true)
	}

	if tokens[0] == "cd" {
		return c.completeRemotePath(active, true)
	}

	cmd, consumed := resolveCommand(c.rootCmd, tokens)

	if consumed == 0 && !endedWithSpace {
		return completeFromPrefix(active, commandCandidates(c.rootCmd), true)
	}

	if cmd == nil {
		return nil, 0
	}

	if len(cmd.Commands()) > 0 && (endedWithSpace || active != "") && len(tokens) == consumed {
		return completeFromPrefix(active, subcommandCandidates(cmd), true)
	}

	if strings.HasPrefix(active, "-") {
		return completeFromPrefix(active, flagCandidates(cmd), true)
	}

	return nil, 0
}

func (c *shellAutoCompleter) completeRemotePath(prefix string, onlyDirs bool) ([][]rune, int) {
	currentPath := c.currentPath()
	targetPath := resolveShellPath(currentPath, prefix)
	basePrefix := prefix
	if strings.TrimSpace(prefix) == "" {
		targetPath = currentPath
		basePrefix = ""
	}

	parentPath := targetPath
	namePrefix := ""
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		parentPath = path.Dir(targetPath)
		if parentPath == "." {
			parentPath = "/"
		}
		namePrefix = path.Base(targetPath)
	}

	parentID := ""
	var err error
	if parentPath != "/" {
		parentID, err = c.fileStatSource.GetPathFolderId(parentPath)
		if err != nil {
			return nil, len([]rune(basePrefix))
		}
	}

	files, err := c.fileStatSource.GetFolderFileStatList(parentID)
	if err != nil {
		return nil, len([]rune(basePrefix))
	}

	candidates := make([]string, 0)
	for _, file := range files {
		if onlyDirs && file.Kind != api.FileKindFolder {
			continue
		}
		if !strings.HasPrefix(file.Name, namePrefix) {
			continue
		}

		remaining := file.Name[len(namePrefix):]
		if file.Kind == api.FileKindFolder {
			remaining += "/"
		}
		candidates = append(candidates, remaining)
	}

	return toRuneCandidates(candidates), len([]rune(basePrefix))
}

func promptForPath(currentPath string) string {
	if currentPath == "/" {
		return "pikpak / > "
	}
	return fmt.Sprintf("pikpak %s/ > ", currentPath)
}

func changeDirectory(p *api.PikPak, currentPath string, args []string) (string, error) {
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

func splitCompletionLine(input string) ([]string, string, bool) {
	args := make([]string, 0)
	var current strings.Builder
	inDoubleQuote := false
	inSingleQuote := false
	endedWithSpace := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		switch ch {
		case '"':
			endedWithSpace = false
			if inSingleQuote {
				current.WriteByte(ch)
			} else {
				inDoubleQuote = !inDoubleQuote
			}
		case '\'':
			endedWithSpace = false
			if inDoubleQuote {
				current.WriteByte(ch)
			} else {
				inSingleQuote = !inSingleQuote
			}
		case ' ', '\t':
			if inDoubleQuote || inSingleQuote {
				current.WriteByte(ch)
				endedWithSpace = false
			} else {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
				endedWithSpace = true
			}
		default:
			current.WriteByte(ch)
			endedWithSpace = false
		}
	}

	if current.Len() > 0 {
		return args, current.String(), false
	}

	return args, "", endedWithSpace
}

func commandCandidates(rootCmd *cobra.Command) []string {
	candidates := append([]string{}, builtInCommands...)
	candidates = append(candidates, subcommandCandidates(rootCmd)...)
	slices.Sort(candidates)
	return slices.Compact(candidates)
}

func subcommandCandidates(cmd *cobra.Command) []string {
	candidates := make([]string, 0)
	for _, sub := range cmd.Commands() {
		if sub.Hidden {
			continue
		}
		candidates = append(candidates, sub.Name())
		candidates = append(candidates, sub.Aliases...)
	}
	slices.Sort(candidates)
	return slices.Compact(candidates)
}

func flagCandidates(cmd *cobra.Command) []string {
	candidates := make([]string, 0)
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		candidates = append(candidates, "--"+f.Name)
		if f.Shorthand != "" {
			candidates = append(candidates, "-"+f.Shorthand)
		}
	})
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		candidates = append(candidates, "--"+f.Name)
		if f.Shorthand != "" {
			candidates = append(candidates, "-"+f.Shorthand)
		}
	})
	slices.Sort(candidates)
	return slices.Compact(candidates)
}

func resolveCommand(rootCmd *cobra.Command, tokens []string) (*cobra.Command, int) {
	current := rootCmd
	consumed := 0

	for _, token := range tokens {
		matched := false
		for _, sub := range current.Commands() {
			if sub.Hidden {
				continue
			}
			if token == sub.Name() || slices.Contains(sub.Aliases, token) {
				current = sub
				consumed++
				matched = true
				break
			}
		}
		if !matched {
			break
		}
	}

	return current, consumed
}

func completeFromPrefix(prefix string, candidates []string, appendSpace bool) ([][]rune, int) {
	matches := make([]string, 0)
	for _, candidate := range candidates {
		if !strings.HasPrefix(candidate, prefix) {
			continue
		}
		suffix := candidate[len(prefix):]
		if appendSpace {
			suffix += " "
		}
		matches = append(matches, suffix)
	}
	return toRuneCandidates(matches), len([]rune(prefix))
}

func toRuneCandidates(candidates []string) [][]rune {
	out := make([][]rune, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, []rune(candidate))
	}
	return out
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

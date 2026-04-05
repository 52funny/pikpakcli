package shell

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/logx"
	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var builtInCommands = []string{"cd", "clear", "exit", "help", "open", "quit"}

const clearScreenSequence = "\033[H\033[2J"

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

		args := parseShellArgs(input)
		if len(args) == 0 {
			continue
		}

		switch args[0] {
		case "exit", "quit":
			fmt.Println("Bye~!")
			return
		case "help":
			rootCmd.Help()
			continue
		case "clear":
			clearScreen(os.Stdout)
			l.SetPrompt(promptForPath(currentPath))
			continue
		case "cd":
			nextPath, err := changeDirectory(&p, currentPath, args[1:])
			if err != nil {
				fmt.Println("Change directory failed")
				logx.Error(err)
				continue
			}
			currentPath = nextPath
			l.SetPrompt(promptForPath(currentPath))
			continue
		case "open":
			cmdCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			if err := handleOpenCommand(p.WithContext(cmdCtx), currentPath, args[1:]); err != nil {
				fmt.Println(err.Error())
			}
			stop()
			if cmdCtx.Err() != nil {
				fmt.Println()
			}
			continue
		}
		args = adaptShellArgs(rootCmd, currentPath, args)

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
	if tokens[0] == "open" {
		return c.completeRemotePath(active, false)
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

	commandKey := canonicalCommandKey(c.rootCmd, cmd)
	if shouldCompleteLocalPathFlagValue(commandKey, tokens, active, endedWithSpace) {
		return completeLocalPath(active, false)
	}
	if shouldCompleteDirectoryPath(commandKey, tokens, active, endedWithSpace, consumed) {
		return c.completeRemotePath(active, true)
	}
	if shouldCompleteRemoteTargetPath(commandKey, tokens, active, consumed) {
		return c.completeRemotePath(active, false)
	}
	if shouldCompleteLocalTargetPath(commandKey, tokens, active, consumed) {
		return completeLocalPath(active, false)
	}

	return nil, 0
}

func shouldCompleteLocalPathFlagValue(commandKey string, tokens []string, active string, endedWithSpace bool) bool {
	if commandKey == "" {
		return false
	}

	switch commandKey {
	case "rubbish":
		return wantsFlagValue(tokens, active, endedWithSpace, "--rules")
	default:
		return false
	}
}

func shouldCompleteDirectoryPath(commandKey string, tokens []string, active string, endedWithSpace bool, consumed int) bool {
	if commandKey == "" {
		return false
	}

	if wantsFlagValue(tokens, active, endedWithSpace, "-p", "--path") {
		switch commandKey {
		case "ls", "empty", "rubbish", "download", "share", "upload", "delete", "new folder", "new url", "new sha":
			return true
		}
	}

	positionalsAfterCommand := positionalTokens(tokens[consumed:], active)

	switch commandKey {
	case "ls", "empty", "rubbish":
		return len(positionalsAfterCommand) <= 1
	}

	return false
}

func shouldCompleteRemoteTargetPath(commandKey string, tokens []string, active string, consumed int) bool {
	if commandKey == "" || active == "" {
		return false
	}

	positionalsAfterCommand := positionalTokens(tokens[consumed:], active)
	switch commandKey {
	case "download", "share", "delete":
		return len(positionalsAfterCommand) >= 1
	case "rename":
		return len(positionalsAfterCommand) == 1
	default:
		return false
	}
}

func shouldCompleteLocalTargetPath(commandKey string, tokens []string, active string, consumed int) bool {
	if commandKey == "" || active == "" {
		return false
	}

	positionalsAfterCommand := positionalTokens(tokens[consumed:], active)
	switch commandKey {
	case "upload":
		return len(positionalsAfterCommand) >= 1
	default:
		return false
	}
}

func wantsFlagValue(tokens []string, active string, endedWithSpace bool, flags ...string) bool {
	if len(tokens) == 0 {
		return false
	}

	last := tokens[len(tokens)-1]
	if endedWithSpace {
		return slices.Contains(flags, last)
	}

	if active != "" {
		return slices.Contains(flags, last)
	}

	return false
}

func positionalTokens(tokens []string, active string) []string {
	positionals := make([]string, 0)
	stopFlags := false

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if stopFlags {
			positionals = append(positionals, token)
			continue
		}

		switch {
		case token == "--":
			stopFlags = true
		case token == "-p" || token == "--path" ||
			token == "-P" || token == "--parent-id" ||
			token == "-o" || token == "--output" ||
			token == "-i" || token == "--input" ||
			token == "-c" || token == "--count" ||
			token == "--rules":
			if i+1 < len(tokens) {
				i++
			}
		case strings.HasPrefix(token, "-"):
		default:
			positionals = append(positionals, token)
		}
	}

	if active != "" {
		positionals = append(positionals, active)
	}

	return positionals
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
		candidates = append(candidates, escapeShellCompletion(remaining))
	}

	return toRuneCandidates(candidates), len([]rune(basePrefix))
}

func completeLocalPath(prefix string, onlyDirs bool) ([][]rune, int) {
	expandedPrefix := utils.ExpandLocalPath(prefix)
	parentPath := "."
	basePrefix := prefix
	namePrefix := expandedPrefix
	hasTrailingSeparator := strings.HasSuffix(prefix, string(filepath.Separator))

	if strings.TrimSpace(prefix) == "" {
		basePrefix = ""
		namePrefix = ""
	} else if !hasTrailingSeparator {
		parentPath = filepath.Dir(expandedPrefix)
		if parentPath == "." && filepath.IsAbs(expandedPrefix) {
			parentPath = string(filepath.Separator)
		}
		namePrefix = filepath.Base(expandedPrefix)
	} else {
		parentPath = expandedPrefix
		namePrefix = ""
	}

	entries, err := os.ReadDir(parentPath)
	if err != nil {
		return nil, len([]rune(basePrefix))
	}

	candidates := make([]string, 0)
	for _, entry := range entries {
		if onlyDirs && !entry.IsDir() {
			continue
		}
		if !strings.HasPrefix(entry.Name(), namePrefix) {
			continue
		}

		remaining := entry.Name()[len(namePrefix):]
		if entry.IsDir() {
			remaining += string(filepath.Separator)
		}
		candidates = append(candidates, escapeShellCompletion(remaining))
	}

	return toRuneCandidates(candidates), len([]rune(basePrefix))
}

func promptForPath(currentPath string) string {
	if currentPath == "/" {
		return "pikpak / > "
	}
	return fmt.Sprintf("pikpak %s/ > ", currentPath)
}

func clearScreen(w io.Writer) {
	fmt.Fprint(w, clearScreenSequence)
}

func adaptShellArgs(rootCmd *cobra.Command, currentPath string, args []string) []string {
	if len(args) == 0 {
		return args
	}

	cmd, consumed := resolveCommand(rootCmd, args)
	if consumed == 0 {
		return args
	}

	commandKey := canonicalCommandKey(rootCmd, cmd)
	rest := append([]string{}, args[consumed:]...)
	flags := inspectShellArgs(rest)

	switch commandKey {
	case "ls", "empty", "rubbish":
		rest = rewritePositionalPaths(rest, currentPath, 1)
		if flags.positionals == 0 && !flags.hasPath && !flags.hasParentID {
			rest = append([]string{"-p", currentPath}, rest...)
		}
	case "download", "share", "upload", "new folder", "new url", "new sha":
		rest = rewritePathFlagValues(rest, currentPath)
		if !flags.hasPath && !flags.hasParentID && flags.positionals > 0 {
			rest = append([]string{"-p", currentPath}, rest...)
		}
		// Handle glob patterns for upload (local paths)
		if commandKey == "upload" {
			expandedRest := make([]string, 0, len(rest))
			for _, arg := range rest {
				if strings.Contains(arg, "*") {
					// Expand glob pattern for local files
					matches, err := filepath.Glob(utils.ExpandLocalPath(arg))
					if err == nil && len(matches) > 0 {
						expandedRest = append(expandedRest, matches...)
					} else {
						expandedRest = append(expandedRest, arg)
					}
				} else {
					expandedRest = append(expandedRest, arg)
				}
			}
			rest = expandedRest
		}
	case "delete":
		if !flags.hasPath {
			rest = rewritePositionalPaths(rest, currentPath, -1)
		}
	case "rename":
		rest = rewritePositionalPaths(rest, currentPath, 1)
	}

	return append(append([]string{}, args[:consumed]...), rest...)
}

func canonicalCommandKey(rootCmd *cobra.Command, cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	path := cmd.CommandPath()
	rootName := rootCmd.Name()
	if path == rootName {
		return ""
	}
	return strings.TrimPrefix(path, rootName+" ")
}

type shellArgFlags struct {
	hasPath     bool
	hasParentID bool
	positionals int
}

func inspectShellArgs(args []string) shellArgFlags {
	var flags shellArgFlags
	stopFlags := false
	for i := 0; i < len(args); i++ {
		token := args[i]
		if stopFlags {
			flags.positionals++
			continue
		}
		switch {
		case token == "--":
			stopFlags = true
		case token == "--path" || token == "-p":
			flags.hasPath = true
			if i+1 < len(args) {
				i++
			}
		case strings.HasPrefix(token, "--path=") || strings.HasPrefix(token, "-p="):
			flags.hasPath = true
		case token == "--parent-id" || token == "-P":
			flags.hasParentID = true
			if i+1 < len(args) {
				i++
			}
		case token == "--rules":
			if i+1 < len(args) {
				i++
			}
		case strings.HasPrefix(token, "--parent-id=") || strings.HasPrefix(token, "-P="):
			flags.hasParentID = true
		case strings.HasPrefix(token, "-"):
		default:
			flags.positionals++
		}
	}
	return flags
}

func rewritePathFlagValues(args []string, currentPath string) []string {
	rewritten := append([]string{}, args...)
	for i := 0; i < len(rewritten); i++ {
		switch token := rewritten[i]; {
		case token == "--path" || token == "-p":
			if i+1 < len(rewritten) {
				rewritten[i+1] = resolveShellPath(currentPath, rewritten[i+1])
				i++
			}
		case strings.HasPrefix(token, "--path="):
			rewritten[i] = "--path=" + resolveShellPath(currentPath, strings.TrimPrefix(token, "--path="))
		case strings.HasPrefix(token, "-p="):
			rewritten[i] = "-p=" + resolveShellPath(currentPath, strings.TrimPrefix(token, "-p="))
		}
	}
	return rewritten
}

func rewritePositionalPaths(args []string, currentPath string, limit int) []string {
	rewritten := append([]string{}, args...)
	stopFlags := false
	rewrittenCount := 0

	for i := 0; i < len(rewritten); i++ {
		token := rewritten[i]
		if stopFlags {
			if limit < 0 || rewrittenCount < limit {
				rewritten[i] = resolveShellPath(currentPath, token)
				rewrittenCount++
			}
			continue
		}

		switch {
		case token == "--":
			stopFlags = true
		case token == "--path" || token == "-p" || token == "--parent-id" || token == "-P" || token == "--output" || token == "-o" || token == "--input" || token == "-i" || token == "--count" || token == "-c" || token == "--rules":
			if i+1 < len(rewritten) {
				i++
			}
		case strings.HasPrefix(token, "-"):
		default:
			if limit >= 0 && rewrittenCount >= limit {
				continue
			}
			rewritten[i] = resolveShellPath(currentPath, token)
			rewrittenCount++
		}
	}

	return rewritten
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
	escaped := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if escaped {
			current.WriteByte(ch)
			escaped = false
			endedWithSpace = false
			continue
		}

		switch ch {
		case '\\':
			if inSingleQuote {
				current.WriteByte(ch)
			} else {
				escaped = true
			}
			endedWithSpace = false
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
	escaped := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}

		switch ch {
		case '\\':
			if inSingleQuote {
				current.WriteByte(ch)
			} else {
				escaped = true
			}
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

func escapeShellCompletion(value string) string {
	var escaped strings.Builder
	escaped.Grow(len(value))
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case ' ', '\\', '"', '\'':
			escaped.WriteByte('\\')
		}
		escaped.WriteByte(value[i])
	}
	return escaped.String()
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

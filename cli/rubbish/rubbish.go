package rubbish

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/logx"
	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/spf13/cobra"
)

var rubbishPath string
var rulesPath string
var rubbishConcurrency int
var rubbishDeleteMode bool
var openRulesFile bool
var openRulesDir bool
var downloadRules bool

const (
	defaultRulesRelativePath = "rules/rubbish_rules.txt"
	defaultRulesDownloadURL  = "https://raw.githubusercontent.com/52funny/pikpakcli/master/rules/rubbish_rules.txt"
)

type rubbishProvider interface {
	GetPathFolderId(dirPath string) (string, error)
	GetFolderFileStatList(parentId string) ([]api.FileStat, error)
	DeleteFile(fileId string) error
}

type compiledRules struct {
	includes []string
	excludes []string
}

type rubbishMatch struct {
	path    string
	pattern string
}

type rubbishWalkState struct {
	sem chan struct{}
	mu  sync.Mutex
}

type rubbishFolderResult struct {
	matches []rubbishMatch
	err     error
}

var RubbishCmd = &cobra.Command{
	Use:   "rubbish [path]",
	Short: "Recursively find rubbish files on the PikPak server using text rules",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := rubbishPath
		if len(args) > 0 {
			path = args[0]
		}

		resolvedRulesPath, err := resolveRulesPath(rulesPath)
		if err != nil {
			fmt.Printf("Resolve rubbish rules failed: %v\n", err)
			return
		}

		if downloadRules {
			if err := downloadDefaultRules(resolvedRulesPath, defaultRulesDownloadURL); err != nil {
				fmt.Printf("Download rubbish rules failed: %v\n", err)
				return
			}
			fmt.Printf("Downloaded rubbish rules to %s\n", resolvedRulesPath)
		}

		if openRulesDir {
			if err := ensureDefaultRulesFile(resolvedRulesPath); err != nil {
				fmt.Printf("Prepare rubbish rules failed: %v\n", err)
				return
			}
			if err := openLocalPath(filepath.Dir(resolvedRulesPath)); err != nil {
				fmt.Printf("Open rules directory failed: %v\n", err)
				return
			}
			fmt.Printf("Opened rules directory: %s\n", filepath.Dir(resolvedRulesPath))
			return
		}

		if openRulesFile {
			if err := ensureDefaultRulesFile(resolvedRulesPath); err != nil {
				fmt.Printf("Prepare rubbish rules failed: %v\n", err)
				return
			}
			if err := openLocalPath(resolvedRulesPath); err != nil {
				fmt.Printf("Open rules file failed: %v\n", err)
				return
			}
			fmt.Printf("Opened rules file: %s\n", resolvedRulesPath)
			return
		}

		if strings.TrimSpace(rulesPath) == "" {
			if err := ensureDefaultRulesFile(resolvedRulesPath); err != nil {
				fmt.Printf("Prepare rubbish rules failed: %v\n", err)
				return
			}
		}

		rules, err := loadRules(resolvedRulesPath)
		if err != nil {
			fmt.Printf("Load rubbish rules failed: %v\n", err)
			return
		}

		p := api.NewPikPakWithContext(cmd.Context(), conf.Config.Username, conf.Config.Password)
		if err := p.Login(); err != nil {
			fmt.Println("Login failed")
			logx.Error(err)
			return
		}

		matches, err := handleRubbish(cmd.Context(), &p, path, rules, rubbishConcurrency, rubbishDeleteMode)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Println("Rubbish scan canceled")
				return
			}
			fmt.Println("Handle rubbish failed")
			logx.Error(err)
			return
		}

		if len(matches) == 0 {
			fmt.Printf("No rubbish files matched under %s\n", path)
			return
		}

		for _, match := range matches {
			if rubbishDeleteMode {
				fmt.Printf("Deleted rubbish: %s (matched %s)\n", match.path, match.pattern)
				continue
			}
			fmt.Printf("Rubbish file: %s (matched %s)\n", match.path, match.pattern)
		}
	},
}

func init() {
	RubbishCmd.Flags().StringVarP(&rubbishPath, "path", "p", "/", "The path where to scan rubbish files recursively")
	RubbishCmd.Flags().StringVar(&rulesPath, "rules", "", "Path or URL to the rubbish rules file")
	RubbishCmd.Flags().IntVarP(&rubbishConcurrency, "concurrency", "c", 8, "number of folders to process concurrently")
	RubbishCmd.Flags().BoolVarP(&rubbishDeleteMode, "delete", "d", false, "delete matched rubbish files instead of only listing them")
	RubbishCmd.Flags().BoolVar(&openRulesFile, "open-rules", false, "Open the rubbish rules file, downloading the default file to the config directory when needed")
	RubbishCmd.Flags().BoolVar(&openRulesDir, "open-rules-dir", false, "Open the rubbish rules directory, downloading the default file to the config directory when needed")
	RubbishCmd.Flags().BoolVar(&downloadRules, "download-rules", false, "Download the default rubbish rules file from GitHub into the config directory before running")
}

func loadRules(path string) (compiledRules, error) {
	expandedPath := utils.ExpandLocalPath(path)
	file, err := os.Open(expandedPath)
	if err != nil {
		return compiledRules{}, err
	}
	defer file.Close()

	var rules compiledRules
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		exclude := strings.HasPrefix(line, "!")
		if exclude {
			line = strings.TrimSpace(strings.TrimPrefix(line, "!"))
		}
		if line == "" {
			continue
		}

		if exclude {
			rules.excludes = append(rules.excludes, line)
			continue
		}
		rules.includes = append(rules.includes, line)
	}
	if err := scanner.Err(); err != nil {
		return compiledRules{}, err
	}
	if len(rules.includes) == 0 {
		return compiledRules{}, fmt.Errorf("no include rules found in %s", expandedPath)
	}
	return rules, nil
}

func resolveRulesPath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultRulesPath()
	}
	if isRemoteRulesSource(trimmed) {
		target, err := defaultRulesPath()
		if err != nil {
			return "", err
		}
		if err := downloadDefaultRules(target, trimmed); err != nil {
			return "", err
		}
		return target, nil
	}

	expanded := utils.ExpandLocalPath(trimmed)
	info, err := os.Stat(expanded)
	if err == nil && info.IsDir() {
		return filepath.Join(expanded, filepath.Base(defaultRulesRelativePath)), nil
	}
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	return expanded, nil
}

func defaultRulesPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get config dir error: %w", err)
	}
	return filepath.Join(configDir, "pikpakcli", defaultRulesRelativePath), nil
}

func ensureDefaultRulesFile(path string) error {
	if path == "" {
		return errors.New("rules path cannot be empty")
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return downloadDefaultRules(path, defaultRulesDownloadURL)
}

func downloadDefaultRules(targetPath string, sourceURL string) error {
	if err := utils.CreateDirIfNotExist(filepath.Dir(targetPath)); err != nil {
		return err
	}

	resp, err := http.Get(sourceURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download rules returned %s", resp.Status)
	}

	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := os.WriteFile(targetPath, bs, 0o644); err != nil {
		return err
	}
	return nil
}

func isRemoteRulesSource(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

func openLocalPath(path string) error {
	name, args, err := buildLocalOpenCommand(runtime.GOOS, path)
	if err != nil {
		return err
	}
	return exec.Command(name, args...).Start()
}

func buildLocalOpenCommand(goos string, path string) (string, []string, error) {
	switch goos {
	case "darwin":
		return "open", []string{path}, nil
	case "linux":
		return "xdg-open", []string{path}, nil
	case "windows":
		return "cmd", []string{"/c", "start", "", path}, nil
	default:
		return "", nil, fmt.Errorf("unsupported platform: %s", goos)
	}
}

func handleRubbish(ctx context.Context, p rubbishProvider, rootPath string, rules compiledRules, concurrency int, deleteMode bool) ([]rubbishMatch, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rootID, err := p.GetPathFolderId(rootPath)
	if err != nil {
		return nil, err
	}
	if concurrency < 1 {
		concurrency = 1
	}

	matches := make([]rubbishMatch, 0)
	state := rubbishWalkState{
		sem: make(chan struct{}, concurrency),
	}
	if err := walkRubbish(ctx, p, rootID, filepath.Clean(rootPath), rules, deleteMode, &matches, &state); err != nil {
		return nil, err
	}
	return matches, nil
}

func walkRubbish(ctx context.Context, p rubbishProvider, folderID, currentPath string, rules compiledRules, deleteMode bool, matches *[]rubbishMatch, state *rubbishWalkState) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	files, err := p.GetFolderFileStatList(folderID)
	if err != nil {
		return err
	}

	results := make(chan rubbishFolderResult, len(files))
	var childFolders int
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}

		childPath := filepath.Join(currentPath, file.Name)
		if file.Kind == api.FileKindFolder {
			childFolders++
			select {
			case <-ctx.Done():
				return ctx.Err()
			case state.sem <- struct{}{}:
				go func(file api.FileStat, childPath string) {
					defer func() {
						<-state.sem
					}()
					localMatches := make([]rubbishMatch, 0)
					err := walkRubbish(ctx, p, file.ID, childPath, rules, deleteMode, &localMatches, state)
					if err == nil {
						if pattern, ok := rules.Match(childPath); ok {
							if deleteMode {
								err = p.DeleteFile(file.ID)
							}
							if err == nil {
								localMatches = append(localMatches, rubbishMatch{path: childPath, pattern: pattern})
							}
						}
					}
					results <- rubbishFolderResult{matches: localMatches, err: err}
				}(file, childPath)
			default:
				localMatches := make([]rubbishMatch, 0)
				if err := walkRubbish(ctx, p, file.ID, childPath, rules, deleteMode, &localMatches, state); err != nil {
					return err
				}
				if pattern, ok := rules.Match(childPath); ok {
					if deleteMode {
						if err := p.DeleteFile(file.ID); err != nil {
							return err
						}
					}
					localMatches = append(localMatches, rubbishMatch{path: childPath, pattern: pattern})
				}
				results <- rubbishFolderResult{matches: localMatches}
			}
			continue
		}

		pattern, ok := rules.Match(childPath)
		if !ok {
			continue
		}
		if deleteMode {
			if err := p.DeleteFile(file.ID); err != nil {
				return err
			}
		}
		state.mu.Lock()
		*matches = append(*matches, rubbishMatch{path: childPath, pattern: pattern})
		state.mu.Unlock()
	}

	for i := 0; i < childFolders; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result := <-results:
			if result.err != nil {
				return result.err
			}
			state.mu.Lock()
			*matches = append(*matches, result.matches...)
			state.mu.Unlock()
		}
	}

	return nil
}

func (r compiledRules) Match(path string) (string, bool) {
	normalizedPath := filepath.Clean(path)
	if normalizedPath == "." {
		normalizedPath = string(filepath.Separator)
	}
	name := filepath.Base(normalizedPath)

	for _, pattern := range r.excludes {
		if patternMatches(pattern, normalizedPath, name) {
			return "", false
		}
	}
	for _, pattern := range r.includes {
		if patternMatches(pattern, normalizedPath, name) {
			return pattern, true
		}
	}
	return "", false
}

func patternMatches(pattern, fullPath, name string) bool {
	pattern = filepath.Clean(strings.TrimSpace(pattern))
	if pattern == "." {
		return false
	}

	matchTarget := name
	if strings.Contains(pattern, string(filepath.Separator)) {
		matchTarget = strings.TrimPrefix(fullPath, string(filepath.Separator))
		if strings.HasPrefix(pattern, string(filepath.Separator)) {
			matchTarget = fullPath
		}
	}

	if !hasWildcard(pattern) {
		return matchTarget == pattern
	}

	matched, err := filepath.Match(pattern, matchTarget)
	return err == nil && matched
}

func hasWildcard(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

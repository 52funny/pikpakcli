package setup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configTemplate []byte

var force bool

// SetupCmd initializes the default config file.
var SetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Initialize the default config file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := Run(Options{
			Force:   force,
			Prompts: terminalPrompts{},
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Config written to %s\n", path)
		return nil
	},
}

type Options struct {
	Force      bool
	TargetPath string
	Prompts    Prompts
}

type Prompts interface {
	ReadLine(prompt string) (string, error)
	ReadPassword(prompt string) (string, error)
}

type setupValues struct {
	Proxy       string
	Username    string
	Password    string
	DownloadDir string
}

type terminalPrompts struct{}

func init() {
	SetupCmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing config file")
}

func SetConfigTemplate(bs []byte) {
	configTemplate = append(configTemplate[:0], bs...)
}

func Run(opts Options) (string, error) {
	if opts.Prompts == nil {
		opts.Prompts = terminalPrompts{}
	}

	targetPath := opts.TargetPath
	if targetPath == "" {
		var err error
		targetPath, err = conf.DefaultConfigPath()
		if err != nil {
			return "", fmt.Errorf("get default config path: %w", err)
		}
	}

	if !opts.Force {
		if _, err := os.Stat(targetPath); err == nil {
			return "", fmt.Errorf("config already exists: %s (use --force to overwrite)", targetPath)
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("check config path: %w", err)
		}
	}

	values, err := collectValues(opts.Prompts)
	if err != nil {
		return "", err
	}

	bs, err := buildConfig(values)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(targetPath, bs, 0600); err != nil {
		return "", fmt.Errorf("write config file: %w", err)
	}
	return targetPath, nil
}

func collectValues(prompts Prompts) (setupValues, error) {
	username, err := prompts.ReadLine("PikPak username: ")
	if err != nil {
		return setupValues{}, fmt.Errorf("read username: %w", err)
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return setupValues{}, fmt.Errorf("username is required")
	}

	password, err := prompts.ReadPassword("PikPak password: ")
	if err != nil {
		return setupValues{}, fmt.Errorf("read password: %w", err)
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return setupValues{}, fmt.Errorf("password is required")
	}

	proxy, err := prompts.ReadLine("Proxy URL (optional): ")
	if err != nil {
		return setupValues{}, fmt.Errorf("read proxy: %w", err)
	}
	proxy = strings.TrimSpace(proxy)
	if proxy != "" && !strings.Contains(proxy, "://") {
		return setupValues{}, fmt.Errorf("proxy should contains ://")
	}

	downloadDir, err := prompts.ReadLine("Open cache download dir (optional): ")
	if err != nil {
		return setupValues{}, fmt.Errorf("read open download dir: %w", err)
	}

	return setupValues{
		Proxy:       proxy,
		Username:    username,
		Password:    password,
		DownloadDir: strings.TrimSpace(downloadDir),
	}, nil
}

func buildConfig(values setupValues) ([]byte, error) {
	if len(configTemplate) == 0 {
		return nil, fmt.Errorf("embedded config template is empty")
	}

	var node yaml.Node
	if err := yaml.Unmarshal(configTemplate, &node); err != nil {
		return nil, fmt.Errorf("parse embedded config template: %w", err)
	}
	if values.Proxy != "" {
		setMappingValue(&node, []string{"proxy"}, values.Proxy)
	}
	setMappingValue(&node, []string{"username"}, values.Username)
	setMappingValue(&node, []string{"password"}, values.Password)
	if values.DownloadDir != "" {
		setMappingValue(&node, []string{"open", "download_dir"}, values.DownloadDir)
	}

	out, err := yaml.Marshal(&node)
	if err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}
	return out, nil
}

func setMappingValue(node *yaml.Node, path []string, value string) bool {
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return setMappingValue(node.Content[0], path, value)
	}
	if node.Kind != yaml.MappingNode || len(path) == 0 {
		return false
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value != path[0] {
			continue
		}
		if len(path) == 1 {
			node.Content[i+1].Kind = yaml.ScalarNode
			node.Content[i+1].Tag = "!!str"
			node.Content[i+1].Value = value
			node.Content[i+1].Style = 0
			return true
		}
		return setMappingValue(node.Content[i+1], path[1:], value)
	}
	return false
}

func (terminalPrompts) ReadLine(prompt string) (string, error) {
	rl, err := readline.NewEx(&readline.Config{Prompt: prompt})
	if err != nil {
		return "", err
	}
	defer rl.Close()
	return rl.Readline()
}

func (terminalPrompts) ReadPassword(prompt string) (string, error) {
	bs, err := readline.Password(prompt)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

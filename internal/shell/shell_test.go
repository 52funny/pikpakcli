package shell

import (
	"testing"

	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestParseShellArgs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "plain args",
			input: "ls -l -p /Movies",
			want:  []string{"ls", "-l", "-p", "/Movies"},
		},
		{
			name:  "double quoted path",
			input: `cd "/Movies/Kids Cartoons"`,
			want:  []string{"cd", "/Movies/Kids Cartoons"},
		},
		{
			name:  "single quoted path",
			input: "cd '/Movies/Kids Cartoons'",
			want:  []string{"cd", "/Movies/Kids Cartoons"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, parseShellArgs(tt.input))
		})
	}
}

func TestResolveShellPath(t *testing.T) {
	tests := []struct {
		name        string
		currentPath string
		target      string
		want        string
	}{
		{
			name:        "root home shortcut",
			currentPath: "/Movies",
			target:      "~",
			want:        "/",
		},
		{
			name:        "relative child",
			currentPath: "/Movies",
			target:      "Kids",
			want:        "/Movies/Kids",
		},
		{
			name:        "relative parent",
			currentPath: "/Movies/Kids",
			target:      "..",
			want:        "/Movies",
		},
		{
			name:        "absolute path",
			currentPath: "/Movies",
			target:      "/TV Shows/Drama",
			want:        "/TV Shows/Drama",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, resolveShellPath(tt.currentPath, tt.target))
		})
	}
}

func TestSplitCompletionLine(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		tokens []string
		active string
		spaced bool
	}{
		{
			name:   "partial command",
			input:  "sh",
			tokens: []string{},
			active: "sh",
		},
		{
			name:   "command with trailing space",
			input:  "cd ",
			tokens: []string{"cd"},
			active: "",
			spaced: true,
		},
		{
			name:   "quoted path",
			input:  `cd "/Movies/Kids Cartoons`,
			tokens: []string{"cd"},
			active: "/Movies/Kids Cartoons",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, active, spaced := splitCompletionLine(tt.input)
			require.Equal(t, tt.tokens, tokens)
			require.Equal(t, tt.active, active)
			require.Equal(t, tt.spaced, spaced)
		})
	}
}

func TestCompleterCommandsAndFlags(t *testing.T) {
	rootCmd := &cobra.Command{Use: "pikpakcli"}
	listCmd := &cobra.Command{Use: "ls"}
	listCmd.Flags().StringP("path", "p", "/", "")
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(&cobra.Command{Use: "shell"})

	completer := &shellAutoCompleter{
		rootCmd:        rootCmd,
		fileStatSource: fakeFileStatProvider{},
		currentPath: func() string {
			return "/"
		},
	}

	candidates, offset := completer.Do([]rune("sh"), 2)
	require.Equal(t, 2, offset)
	require.Contains(t, candidates, []rune("ell "))

	candidates, offset = completer.Do([]rune("ls -"), 4)
	require.Equal(t, 1, offset)
	require.Contains(t, candidates, []rune("p "))
}

func TestCompleterCDPath(t *testing.T) {
	completer := &shellAutoCompleter{
		rootCmd: &cobra.Command{Use: "pikpakcli"},
		fileStatSource: fakeFileStatProvider{
			folders: map[string][]pikpak.FileStat{
				"": {
					{Name: "Movies", Kind: "drive#folder"},
					{Name: "Music", Kind: "drive#folder"},
				},
				"movies-id": {
					{Name: "Kids Cartoons", Kind: "drive#folder"},
				},
			},
			ids: map[string]string{
				"/Movies": "movies-id",
			},
		},
		currentPath: func() string {
			return "/"
		},
	}

	candidates, offset := completer.Do([]rune("cd /Mov"), len("cd /Mov"))
	require.Equal(t, len([]rune("/Mov")), offset)
	require.Contains(t, candidates, []rune("ies/"))
}

type fakeFileStatProvider struct {
	folders map[string][]pikpak.FileStat
	ids     map[string]string
}

func (f fakeFileStatProvider) GetPathFolderId(dirPath string) (string, error) {
	if id, ok := f.ids[dirPath]; ok {
		return id, nil
	}
	return "", nil
}

func (f fakeFileStatProvider) GetFolderFileStatList(parentId string) ([]pikpak.FileStat, error) {
	return f.folders[parentId], nil
}

package shell

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/52funny/pikpakcli/internal/api"
	"github.com/chzyer/readline"
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
		{
			name:        "clean repeated separators",
			currentPath: "/Movies",
			target:      "Kids//Cartoons",
			want:        "/Movies/Kids/Cartoons",
		},
		{
			name:        "empty target goes root",
			currentPath: "/Movies/Kids",
			target:      "",
			want:        "/",
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

func TestShouldExitOnReadlineError(t *testing.T) {
	require.True(t, shouldExitOnReadlineError(io.EOF))
	require.False(t, shouldExitOnReadlineError(nil))
	require.False(t, shouldExitOnReadlineError(readline.ErrInterrupt))
	require.False(t, shouldExitOnReadlineError(errors.New("other error")))
}

func TestIsReadlineInterrupt(t *testing.T) {
	require.True(t, isReadlineInterrupt(readline.ErrInterrupt))
	require.False(t, isReadlineInterrupt(nil))
	require.False(t, isReadlineInterrupt(io.EOF))
}

func TestSetCommandContextTree(t *testing.T) {
	rootCmd := &cobra.Command{Use: "root"}
	childCmd := &cobra.Command{Use: "child"}
	rootCmd.AddCommand(childCmd)

	ctx1, cancel1 := context.WithCancel(context.Background())
	setCommandContextTree(rootCmd, ctx1)
	cancel1()

	require.ErrorIs(t, rootCmd.Context().Err(), context.Canceled)
	require.ErrorIs(t, childCmd.Context().Err(), context.Canceled)

	ctx2 := context.Background()
	setCommandContextTree(rootCmd, ctx2)

	require.NoError(t, rootCmd.Context().Err())
	require.NoError(t, childCmd.Context().Err())
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
	require.Contains(t, commandCandidates(rootCmd), "clear")

	candidates, offset = completer.Do([]rune("ls -"), 4)
	require.Equal(t, 1, offset)
	require.Contains(t, candidates, []rune("p "))
}

func TestClearScreen(t *testing.T) {
	var out strings.Builder
	clearScreen(&out)
	require.Equal(t, clearScreenSequence, out.String())
}

func TestCompleterCDPath(t *testing.T) {
	completer := &shellAutoCompleter{
		rootCmd: &cobra.Command{Use: "pikpakcli"},
		fileStatSource: fakeFileStatProvider{
			folders: map[string][]api.FileStat{
				"": {
					{Name: "Movies", Kind: api.FileKindFolder},
					{Name: "Music", Kind: api.FileKindFolder},
				},
				"movies-id": {
					{Name: "Kids Cartoons", Kind: api.FileKindFolder},
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

func TestCompleterCDPathFromCurrentDirectory(t *testing.T) {
	completer := &shellAutoCompleter{
		rootCmd: &cobra.Command{Use: "pikpakcli"},
		fileStatSource: fakeFileStatProvider{
			folders: map[string][]api.FileStat{
				"movies-id": {
					{Name: "Kids Cartoons", Kind: api.FileKindFolder},
					{Name: "Drama", Kind: api.FileKindFolder},
				},
			},
			ids: map[string]string{
				"/Movies": "movies-id",
			},
		},
		currentPath: func() string {
			return "/Movies"
		},
	}

	candidates, offset := completer.Do([]rune("cd Ki"), len("cd Ki"))
	require.Equal(t, len([]rune("Ki")), offset)
	require.Contains(t, candidates, []rune("ds Cartoons/"))
}

type fakeFileStatProvider struct {
	folders map[string][]api.FileStat
	ids     map[string]string
}

func (f fakeFileStatProvider) GetPathFolderId(dirPath string) (string, error) {
	if id, ok := f.ids[dirPath]; ok {
		return id, nil
	}
	return "", nil
}

func (f fakeFileStatProvider) GetFolderFileStatList(parentId string) ([]api.FileStat, error) {
	return f.folders[parentId], nil
}

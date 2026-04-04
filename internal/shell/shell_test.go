package shell

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/52funny/pikpakcli/conf"
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
		{
			name:  "escaped spaces",
			input: `cd /My\ Pack/Kids\ Cartoons`,
			want:  []string{"cd", "/My Pack/Kids Cartoons"},
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
		{
			name:   "escaped spaces",
			input:  `cd /My\ Pack/Kids\ Cart`,
			tokens: []string{"cd"},
			active: "/My Pack/Kids Cart",
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
	emptyCmd := &cobra.Command{Use: "empty"}
	emptyCmd.Flags().StringP("path", "p", "/", "")
	rootCmd.AddCommand(emptyCmd)
	downloadCmd := &cobra.Command{Use: "download"}
	downloadCmd.Flags().StringP("path", "p", "/", "")
	rootCmd.AddCommand(downloadCmd)
	shareCmd := &cobra.Command{Use: "share"}
	shareCmd.Flags().StringP("path", "p", "/", "")
	rootCmd.AddCommand(shareCmd)
	deleteCmd := &cobra.Command{Use: "delete"}
	deleteCmd.Flags().StringP("path", "p", "/", "")
	rootCmd.AddCommand(deleteCmd)
	renameCmd := &cobra.Command{Use: "rename"}
	rootCmd.AddCommand(renameCmd)
	rootCmd.AddCommand(&cobra.Command{Use: "shell"})

	completer := &shellAutoCompleter{
		rootCmd: rootCmd,
		fileStatSource: fakeFileStatProvider{
			folders: map[string][]api.FileStat{
				"": {
					{Name: "Movies", Kind: api.FileKindFolder},
					{Name: "Music", Kind: api.FileKindFolder},
					{Name: "movie.mp4", Kind: api.FileKindFile},
				},
			},
		},
		currentPath: func() string {
			return "/"
		},
	}

	candidates, offset := completer.Do([]rune("sh"), 2)
	require.Equal(t, 2, offset)
	require.Contains(t, candidates, []rune("ell "))
	require.Contains(t, commandCandidates(rootCmd), "clear")
	require.Contains(t, commandCandidates(rootCmd), "open")

	candidates, offset = completer.Do([]rune("ls -"), 4)
	require.Equal(t, 1, offset)
	require.Contains(t, candidates, []rune("p "))

	candidates, offset = completer.Do([]rune("ls /Mov"), len("ls /Mov"))
	require.Equal(t, len([]rune("/Mov")), offset)
	require.Contains(t, candidates, []rune("ies/"))

	candidates, offset = completer.Do([]rune("empty -p /Mov"), len("empty -p /Mov"))
	require.Equal(t, len([]rune("/Mov")), offset)
	require.Contains(t, candidates, []rune("ies/"))

	candidates, offset = completer.Do([]rune("download -p /Mov"), len("download -p /Mov"))
	require.Equal(t, len([]rune("/Mov")), offset)
	require.Contains(t, candidates, []rune("ies/"))

	candidates, offset = completer.Do([]rune("download mov"), len("download mov"))
	require.Equal(t, len([]rune("mov")), offset)
	require.Contains(t, candidates, []rune("ie.mp4"))

	candidates, offset = completer.Do([]rune("share mov"), len("share mov"))
	require.Equal(t, len([]rune("mov")), offset)
	require.Contains(t, candidates, []rune("ie.mp4"))

	candidates, offset = completer.Do([]rune("delete mov"), len("delete mov"))
	require.Equal(t, len([]rune("mov")), offset)
	require.Contains(t, candidates, []rune("ie.mp4"))

	candidates, offset = completer.Do([]rune("rename mov"), len("rename mov"))
	require.Equal(t, len([]rune("mov")), offset)
	require.Contains(t, candidates, []rune("ie.mp4"))

	candidates, offset = completer.Do([]rune("open mov"), len("open mov"))
	require.Equal(t, len([]rune("mov")), offset)
	require.Contains(t, candidates, []rune("ie.mp4"))
}

func TestCompleterUploadLocalPath(t *testing.T) {
	tempDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "local.txt"), []byte("x"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(tempDir, "folder"), 0755))

	oldWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	rootCmd := &cobra.Command{Use: "pikpakcli"}
	uploadCmd := &cobra.Command{Use: "upload"}
	uploadCmd.Flags().StringP("path", "p", "/", "")
	rootCmd.AddCommand(uploadCmd)

	completer := &shellAutoCompleter{
		rootCmd:        rootCmd,
		fileStatSource: fakeFileStatProvider{},
		currentPath: func() string {
			return "/"
		},
	}

	candidates, offset := completer.Do([]rune("upload loc"), len("upload loc"))
	require.Equal(t, len([]rune("loc")), offset)
	require.Contains(t, candidates, []rune("al.txt"))

	candidates, offset = completer.Do([]rune("upload fol"), len("upload fol"))
	require.Equal(t, len([]rune("fol")), offset)
	require.Contains(t, candidates, []rune("der/"))
}

func TestCompleterUploadHomePath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tempHomeRoot := filepath.Dir(home)
	homeName := filepath.Base(home)
	testDirName := "codex-upload-home-test"
	testDir := filepath.Join(home, testDirName)
	require.NoError(t, os.MkdirAll(testDir, 0755))
	t.Cleanup(func() {
		_ = os.RemoveAll(testDir)
	})

	rootCmd := &cobra.Command{Use: "pikpakcli"}
	uploadCmd := &cobra.Command{Use: "upload"}
	uploadCmd.Flags().StringP("path", "p", "/", "")
	rootCmd.AddCommand(uploadCmd)

	completer := &shellAutoCompleter{
		rootCmd:        rootCmd,
		fileStatSource: fakeFileStatProvider{},
		currentPath: func() string {
			return "/"
		},
	}

	oldWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempHomeRoot))
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	candidates, offset := completer.Do([]rune("upload ~/"+testDirName[:5]), len("upload ~/"+testDirName[:5]))
	require.Equal(t, len([]rune("~/"+testDirName[:5])), offset)
	require.Contains(t, candidates, []rune(testDirName[5:]+"/"))
	require.NotEmpty(t, homeName)

	candidates, offset = completer.Do([]rune("upload ~/"), len("upload ~/"))
	require.Equal(t, len([]rune("~/")), offset)
	require.Contains(t, candidates, []rune(testDirName+"/"))
}

func TestClearScreen(t *testing.T) {
	var out strings.Builder
	clearScreen(&out)
	require.Equal(t, clearScreenSequence, out.String())
}

func TestAdaptShellArgs(t *testing.T) {
	rootCmd := &cobra.Command{Use: "pikpakcli"}

	listCmd := &cobra.Command{Use: "ls"}
	listCmd.Flags().StringP("path", "p", "/", "")
	rootCmd.AddCommand(listCmd)

	emptyCmd := &cobra.Command{Use: "empty"}
	emptyCmd.Flags().StringP("path", "p", "/", "")
	rootCmd.AddCommand(emptyCmd)

	downloadCmd := &cobra.Command{Use: "download"}
	downloadCmd.Flags().StringP("path", "p", "/", "")
	downloadCmd.Flags().StringP("parent-id", "P", "", "")
	rootCmd.AddCommand(downloadCmd)

	shareCmd := &cobra.Command{Use: "share"}
	shareCmd.Flags().StringP("path", "p", "/", "")
	shareCmd.Flags().StringP("parent-id", "P", "", "")
	rootCmd.AddCommand(shareCmd)

	uploadCmd := &cobra.Command{Use: "upload"}
	uploadCmd.Flags().StringP("path", "p", "/", "")
	uploadCmd.Flags().StringP("parent-id", "P", "", "")
	rootCmd.AddCommand(uploadCmd)

	deleteCmd := &cobra.Command{Use: "delete"}
	deleteCmd.Aliases = []string{"del", "rm"}
	deleteCmd.Flags().StringP("path", "p", "/", "")
	rootCmd.AddCommand(deleteCmd)

	renameCmd := &cobra.Command{Use: "rename"}
	rootCmd.AddCommand(renameCmd)

	newCmd := &cobra.Command{Use: "new"}
	newCmd.Aliases = []string{"n"}
	newFolderCmd := &cobra.Command{Use: "folder"}
	newFolderCmd.Flags().StringP("path", "p", "/", "")
	newFolderCmd.Flags().StringP("parent-id", "P", "", "")
	newCmd.AddCommand(newFolderCmd)
	newURLCmd := &cobra.Command{Use: "url"}
	newURLCmd.Flags().StringP("path", "p", "/", "")
	newURLCmd.Flags().StringP("parent-id", "P", "", "")
	newCmd.AddCommand(newURLCmd)
	newSHACmd := &cobra.Command{Use: "sha"}
	newSHACmd.Flags().StringP("path", "p", "/", "")
	newSHACmd.Flags().StringP("parent-id", "P", "", "")
	newCmd.AddCommand(newSHACmd)
	rootCmd.AddCommand(newCmd)

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "ls injects current path", args: []string{"ls"}, want: []string{"ls", "-p", "/Movies"}},
		{name: "ls rewrites relative arg", args: []string{"ls", "Kids"}, want: []string{"ls", "/Movies/Kids"}},
		{name: "empty rewrites relative arg", args: []string{"empty", "Kids"}, want: []string{"empty", "/Movies/Kids"}},
		{name: "download injects current path", args: []string{"download", "episode.mkv"}, want: []string{"download", "-p", "/Movies", "episode.mkv"}},
		{name: "download rewrites relative path flag", args: []string{"download", "-p", "Kids", "episode.mkv"}, want: []string{"download", "-p", "/Movies/Kids", "episode.mkv"}},
		{name: "download keeps trailing dot as positional target", args: []string{"download", "-g", "episode.mkv", "."}, want: []string{"download", "-p", "/Movies", "-g", "episode.mkv", "."}},
		{name: "share injects current path", args: []string{"share", "episode.mkv"}, want: []string{"share", "-p", "/Movies", "episode.mkv"}},
		{name: "upload injects current path", args: []string{"upload", "local.file"}, want: []string{"upload", "-p", "/Movies", "local.file"}},
		{name: "delete rewrites relative args", args: []string{"delete", "a", "b/c"}, want: []string{"delete", "/Movies/a", "/Movies/b/c"}},
		{name: "rm alias rewrites relative args", args: []string{"rm", "a", "b/c"}, want: []string{"rm", "/Movies/a", "/Movies/b/c"}},
		{name: "rename rewrites first arg only", args: []string{"rename", "old.txt", "new.txt"}, want: []string{"rename", "/Movies/old.txt", "new.txt"}},
		{name: "new folder injects current path", args: []string{"new", "folder", "a/b"}, want: []string{"new", "folder", "-p", "/Movies", "a/b"}},
		{name: "new alias folder injects current path", args: []string{"n", "folder", "a/b"}, want: []string{"n", "folder", "-p", "/Movies", "a/b"}},
		{name: "new url injects current path", args: []string{"new", "url", "https://example.com"}, want: []string{"new", "url", "-p", "/Movies", "https://example.com"}},
		{name: "new sha injects current path", args: []string{"new", "sha", "PikPak://a|1|sha"}, want: []string{"new", "sha", "-p", "/Movies", "PikPak://a|1|sha"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, adaptShellArgs(rootCmd, "/Movies", tt.args))
		})
	}
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
	require.Contains(t, candidates, []rune(`ds\ Cartoons/`))
}

func TestCompleterEscapesSpacesInPath(t *testing.T) {
	completer := &shellAutoCompleter{
		rootCmd: &cobra.Command{Use: "pikpakcli"},
		fileStatSource: fakeFileStatProvider{
			folders: map[string][]api.FileStat{
				"": {
					{Name: "My Pack", Kind: api.FileKindFolder},
				},
			},
		},
		currentPath: func() string {
			return "/"
		},
	}

	candidates, offset := completer.Do([]rune("cd /My"), len("cd /My"))
	require.Equal(t, len([]rune("/My")), offset)
	require.Contains(t, candidates, []rune(`\ Pack/`))
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

func TestClassifyOpenCategory(t *testing.T) {
	require.Equal(t, openCategoryText, classifyOpenCategory("readme.md"))
	require.Equal(t, openCategoryImage, classifyOpenCategory("cover.png"))
	require.Equal(t, openCategoryVideo, classifyOpenCategory("movie.mkv"))
	require.Equal(t, openCategoryAudio, classifyOpenCategory("song.flac"))
	require.Equal(t, openCategoryPDF, classifyOpenCategory("paper.pdf"))
	require.Equal(t, openCategoryDefault, classifyOpenCategory("archive.zip"))
}

func TestBuildOpenCommand(t *testing.T) {
	name, args, err := buildOpenCommand("darwin", conf.OpenConfig{}, "/tmp/demo.txt", openCategoryText)
	require.NoError(t, err)
	require.Equal(t, "open", name)
	require.Equal(t, []string{"-a", "TextEdit", "/tmp/demo.txt"}, args)

	name, args, err = buildOpenCommand("darwin", conf.OpenConfig{}, "/tmp/demo.mp4", openCategoryVideo)
	require.NoError(t, err)
	require.Equal(t, "open", name)
	require.Equal(t, []string{"-a", "IINA", "/tmp/demo.mp4"}, args)

	name, args, err = buildOpenCommand("linux", conf.OpenConfig{
		Video: []string{"vlc", "--fullscreen"},
	}, "/tmp/demo.mp4", openCategoryVideo)
	require.NoError(t, err)
	require.Equal(t, "vlc", name)
	require.Equal(t, []string{"--fullscreen", "/tmp/demo.mp4"}, args)

	name, args, err = buildOpenCommand("linux", conf.OpenConfig{
		Default: []string{"custom-open", "--file", "{path}"},
	}, "/tmp/demo.bin", openCategoryDefault)
	require.NoError(t, err)
	require.Equal(t, "custom-open", name)
	require.Equal(t, []string{"--file", "/tmp/demo.bin"}, args)
}

func TestRemoteVideoOpenURL(t *testing.T) {
	file := &api.File{}
	file.Medias = []struct {
		MediaID   string      `json:"media_id"`
		MediaName string      `json:"media_name"`
		Video     interface{} `json:"video"`
		Link      struct {
			URL    string    `json:"url"`
			Token  string    `json:"token"`
			Expire time.Time `json:"expire"`
		} `json:"link"`
		NeedMoreQuota  bool          `json:"need_more_quota"`
		VipTypes       []interface{} `json:"vip_types"`
		RedirectLink   string        `json:"redirect_link"`
		IconLink       string        `json:"icon_link"`
		IsDefault      bool          `json:"is_default"`
		Priority       int           `json:"priority"`
		IsOrigin       bool          `json:"is_origin"`
		ResolutionName string        `json:"resolution_name"`
		IsVisible      bool          `json:"is_visible"`
		Category       string        `json:"category"`
	}{
		{
			Link: struct {
				URL    string    `json:"url"`
				Token  string    `json:"token"`
				Expire time.Time `json:"expire"`
			}{URL: "https://example.com/visible.m3u8"},
			IsVisible: true,
		},
		{
			Link: struct {
				URL    string    `json:"url"`
				Token  string    `json:"token"`
				Expire time.Time `json:"expire"`
			}{URL: "https://example.com/default.m3u8"},
			IsDefault: true,
			IsVisible: true,
		},
	}

	require.Equal(t, "https://example.com/default.m3u8", remoteVideoOpenURL(file))
}

func TestResolveOpenTargetForVideoPrefersRemoteURL(t *testing.T) {
	file := &api.File{}
	file.Name = "movie.mkv"
	file.Links.ApplicationOctetStream.URL = "https://example.com/download.mp4"

	target, err := resolveOpenTarget(file)
	require.NoError(t, err)
	require.Equal(t, "https://example.com/download.mp4", target)
}

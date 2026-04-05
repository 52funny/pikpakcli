package rubbish

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/52funny/pikpakcli/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRubbishProvider struct {
	pathToID     map[string]string
	folders      map[string][]api.FileStat
	deletedFiles []string
	mu           sync.Mutex
}

func (f *fakeRubbishProvider) GetPathFolderId(dirPath string) (string, error) {
	if id, ok := f.pathToID[filepath.Clean(dirPath)]; ok {
		return id, nil
	}
	return "", errors.New("path not found")
}

func (f *fakeRubbishProvider) GetFolderFileStatList(parentId string) ([]api.FileStat, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	files := f.folders[parentId]
	cloned := make([]api.FileStat, len(files))
	copy(cloned, files)
	return cloned, nil
}

func (f *fakeRubbishProvider) DeleteFile(fileId string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deletedFiles = append(f.deletedFiles, fileId)
	for parentID, files := range f.folders {
		filtered := files[:0]
		for _, file := range files {
			if file.ID != fileId {
				filtered = append(filtered, file)
			}
		}
		f.folders[parentID] = filtered
	}
	delete(f.folders, fileId)
	return nil
}

func TestLoadRules(t *testing.T) {
	dir := t.TempDir()
	rulesFile := filepath.Join(dir, "rules.txt")
	err := os.WriteFile(rulesFile, []byte("# comment\n\n.DS_Store\n*.tmp\n!important.tmp\n"), 0o644)
	require.NoError(t, err)

	rules, err := loadRules(rulesFile)
	require.NoError(t, err)
	assert.Equal(t, []string{".DS_Store", "*.tmp"}, rules.includes)
	assert.Equal(t, []string{"important.tmp"}, rules.excludes)
}

func TestCompiledRulesMatch(t *testing.T) {
	rules := compiledRules{
		includes: []string{".DS_Store", "*.tmp", "cache/*.part", "/System/*"},
		excludes: []string{"keep.tmp", "!ignored", "/System/keep/*"},
	}

	pattern, ok := rules.Match("/Movies/.DS_Store")
	require.True(t, ok)
	assert.Equal(t, ".DS_Store", pattern)

	pattern, ok = rules.Match("/Movies/video.tmp")
	require.True(t, ok)
	assert.Equal(t, "*.tmp", pattern)

	pattern, ok = rules.Match("/cache/file.part")
	require.True(t, ok)
	assert.Equal(t, "cache/*.part", pattern)

	_, ok = rules.Match("/Movies/keep.tmp")
	assert.False(t, ok)

	pattern, ok = rules.Match("/System/logs")
	require.True(t, ok)
	assert.Equal(t, "/System/*", pattern)

	_, ok = rules.Match("/System/keep/file")
	assert.False(t, ok)
}

func TestHandleRubbishListsAndDeletesMatches(t *testing.T) {
	provider := &fakeRubbishProvider{
		pathToID: map[string]string{
			filepath.Clean("/"): "root",
		},
		folders: map[string][]api.FileStat{
			"root": {
				{ID: "movies", Name: "Movies", Kind: api.FileKindFolder},
				{ID: "ds", Name: ".DS_Store", Kind: api.FileKindFile},
				{ID: "keep", Name: "keep.tmp", Kind: api.FileKindFile},
			},
			"movies": {
				{ID: "partial", Name: "video.part", Kind: api.FileKindFile},
				{ID: "poster", Name: "poster.jpg", Kind: api.FileKindFile},
			},
		},
	}

	rules := compiledRules{
		includes: []string{".DS_Store", "*.part", "*.tmp"},
		excludes: []string{"keep.tmp"},
	}

	matches, err := handleRubbish(context.Background(), provider, "/", rules, 4, false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []rubbishMatch{
		{path: filepath.Clean("/.DS_Store"), pattern: ".DS_Store"},
		{path: filepath.Clean("/Movies/video.part"), pattern: "*.part"},
	}, matches)
	assert.Empty(t, provider.deletedFiles)

	matches, err = handleRubbish(context.Background(), provider, "/", rules, 4, true)
	require.NoError(t, err)
	assert.ElementsMatch(t, []rubbishMatch{
		{path: filepath.Clean("/.DS_Store"), pattern: ".DS_Store"},
		{path: filepath.Clean("/Movies/video.part"), pattern: "*.part"},
	}, matches)
	assert.ElementsMatch(t, []string{"ds", "partial"}, provider.deletedFiles)
}

func TestHandleRubbishNormalizesConcurrency(t *testing.T) {
	provider := &fakeRubbishProvider{
		pathToID: map[string]string{
			filepath.Clean("/"): "root",
		},
		folders: map[string][]api.FileStat{
			"root": {
				{ID: "tmp", Name: "a.tmp", Kind: api.FileKindFile},
			},
		},
	}

	matches, err := handleRubbish(context.Background(), provider, "/", compiledRules{includes: []string{"*.tmp"}}, 0, false)
	require.NoError(t, err)
	assert.Equal(t, []rubbishMatch{{path: filepath.Clean("/a.tmp"), pattern: "*.tmp"}}, matches)
}

func TestDefaultRulesPathUsesConfigDir(t *testing.T) {
	configDir, err := os.UserConfigDir()
	require.NoError(t, err)
	path, err := defaultRulesPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(configDir, "pikpakcli", "rules", "rubbish_rules.txt"), path)
}

func TestResolveRulesPathForDirectory(t *testing.T) {
	dir := t.TempDir()

	path, err := resolveRulesPath(dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "rubbish_rules.txt"), path)
}

func TestDownloadDefaultRules(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(".DS_Store\n*.tmp\n"))
	}))
	defer server.Close()

	targetDir := t.TempDir()
	targetPath := filepath.Join(targetDir, "rules.txt")
	err := downloadDefaultRules(targetPath, server.URL)
	require.NoError(t, err)

	bs, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, ".DS_Store\n*.tmp\n", string(bs))
}

func TestBuildLocalOpenCommand(t *testing.T) {
	name, args, err := buildLocalOpenCommand("linux", "/tmp/rules.txt")
	require.NoError(t, err)
	assert.Equal(t, "xdg-open", name)
	assert.Equal(t, []string{"/tmp/rules.txt"}, args)
}

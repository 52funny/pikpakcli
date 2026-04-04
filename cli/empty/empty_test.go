package empty

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/52funny/pikpakcli/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeEmptyFolderProvider struct {
	rootID       string
	pathToID     map[string]string
	folders      map[string][]api.FileStat
	deletedFiles []string
	mu           sync.Mutex
}

func (f *fakeEmptyFolderProvider) GetPathFolderId(dirPath string) (string, error) {
	if id, ok := f.pathToID[filepath.Clean(dirPath)]; ok {
		return id, nil
	}
	return "", errors.New("path not found")
}

func (f *fakeEmptyFolderProvider) GetFolderFileStatList(parentId string) ([]api.FileStat, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	files := f.folders[parentId]
	cloned := make([]api.FileStat, len(files))
	copy(cloned, files)
	return cloned, nil
}

func (f *fakeEmptyFolderProvider) DeleteFile(fileId string) error {
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

func TestHandleEmptyFoldersDeletesNestedEmptyFolders(t *testing.T) {
	provider := &fakeEmptyFolderProvider{
		pathToID: map[string]string{
			filepath.Clean("/"): "root",
		},
		folders: map[string][]api.FileStat{
			"root": {
				{ID: "movies", Name: "Movies", Kind: api.FileKindFolder},
				{ID: "music", Name: "Music", Kind: api.FileKindFolder},
				{ID: "video", Name: "video.mp4", Kind: api.FileKindFile},
			},
			"movies": {
				{ID: "kids", Name: "Kids", Kind: api.FileKindFolder},
			},
			"kids":  {},
			"music": {},
		},
	}

	deleted, err := handleEmptyFolders(context.Background(), provider, "/", 4, true)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{filepath.Clean("/Movies/Kids"), filepath.Clean("/Movies"), filepath.Clean("/Music")}, deleted)
	assert.ElementsMatch(t, []string{"kids", "movies", "music"}, provider.deletedFiles)
}

func TestHandleEmptyFoldersSkipsNonEmptyRootTarget(t *testing.T) {
	provider := &fakeEmptyFolderProvider{
		pathToID: map[string]string{
			filepath.Clean("/Movies"): "movies",
		},
		folders: map[string][]api.FileStat{
			"movies": {
				{ID: "episode", Name: "episode.mkv", Kind: api.FileKindFile},
			},
		},
	}

	deleted, err := handleEmptyFolders(context.Background(), provider, "/Movies", 4, true)
	require.NoError(t, err)
	assert.Empty(t, deleted)
	assert.Empty(t, provider.deletedFiles)
}

func TestHandleEmptyFoldersDeletesTargetWhenItBecomesEmpty(t *testing.T) {
	provider := &fakeEmptyFolderProvider{
		pathToID: map[string]string{
			filepath.Clean("/Movies"): "movies",
		},
		folders: map[string][]api.FileStat{
			"movies": {
				{ID: "kids", Name: "Kids", Kind: api.FileKindFolder},
			},
			"kids": {},
		},
	}

	deleted, err := handleEmptyFolders(context.Background(), provider, "/Movies", 4, true)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Clean("/Movies/Kids"), filepath.Clean("/Movies")}, deleted)
	assert.Equal(t, []string{"kids", "movies"}, provider.deletedFiles)
}

func TestHandleEmptyFoldersNormalizesInvalidConcurrency(t *testing.T) {
	provider := &fakeEmptyFolderProvider{
		pathToID: map[string]string{
			filepath.Clean("/Movies"): "movies",
		},
		folders: map[string][]api.FileStat{
			"movies": {},
		},
	}

	deleted, err := handleEmptyFolders(context.Background(), provider, "/Movies", 0, true)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Clean("/Movies")}, deleted)
	assert.Equal(t, []string{"movies"}, provider.deletedFiles)
}

func TestHandleEmptyFoldersListsWithoutDeleting(t *testing.T) {
	provider := &fakeEmptyFolderProvider{
		pathToID: map[string]string{
			filepath.Clean("/"): "root",
		},
		folders: map[string][]api.FileStat{
			"root": {
				{ID: "movies", Name: "Movies", Kind: api.FileKindFolder},
			},
			"movies": {},
		},
	}

	emptyFolders, err := handleEmptyFolders(context.Background(), provider, "/", 4, false)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Clean("/Movies")}, emptyFolders)
	assert.Empty(t, provider.deletedFiles)
}

type blockingEmptyFolderProvider struct {
	fakeEmptyFolderProvider
	block chan struct{}
}

func (f *blockingEmptyFolderProvider) GetFolderFileStatList(parentId string) ([]api.FileStat, error) {
	if parentId == "slow" {
		<-f.block
	}
	return f.fakeEmptyFolderProvider.GetFolderFileStatList(parentId)
}

func TestHandleEmptyFoldersHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	provider := &fakeEmptyFolderProvider{
		pathToID: map[string]string{
			filepath.Clean("/"): "root",
		},
		folders: map[string][]api.FileStat{
			"root": {},
		},
	}

	deleted, err := handleEmptyFolders(ctx, provider, "/", 4, false)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, deleted)
}

func TestHandleEmptyFoldersStopsWaitingAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	provider := &blockingEmptyFolderProvider{
		fakeEmptyFolderProvider: fakeEmptyFolderProvider{
			pathToID: map[string]string{
				filepath.Clean("/"): "root",
			},
			folders: map[string][]api.FileStat{
				"root": {
					{ID: "slow", Name: "slow", Kind: api.FileKindFolder},
				},
				"slow": {},
			},
		},
		block: make(chan struct{}),
	}

	done := make(chan error, 1)
	go func() {
		_, err := handleEmptyFolders(ctx, provider, "/", 4, false)
		done <- err
	}()

	cancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("handleEmptyFolders did not stop promptly after cancellation")
	}

	close(provider.block)
}

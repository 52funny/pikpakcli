package download

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/52funny/pikpakcli/internal/api"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

type fakeTargetResolver struct {
	getFileByPath func(path string) (api.FileStat, error)
	getFileStat   func(parentId string, name string) (api.FileStat, error)
	getPathFolder func(dirPath string) (string, error)
}

func (f fakeTargetResolver) GetFileByPath(path string) (api.FileStat, error) {
	return f.getFileByPath(path)
}

func (f fakeTargetResolver) GetFileStat(parentId string, name string) (api.FileStat, error) {
	return f.getFileStat(parentId, name)
}

func (f fakeTargetResolver) GetPathFolderId(dirPath string) (string, error) {
	return f.getPathFolder(dirPath)
}

func TestRemoteTargetPathJoinsBasePath(t *testing.T) {
	originalFolder := folder
	t.Cleanup(func() {
		folder = originalFolder
	})

	folder = "/Movies"
	require.Equal(t, filepath.Clean("/Movies/Kids/Peppa.mp4"), remoteTargetPath("Kids/Peppa.mp4"))
	require.Equal(t, filepath.Clean("/TV"), remoteTargetPath("/TV"))
}

func TestResolveDownloadTargetUsesParentIDForDirectChild(t *testing.T) {
	originalFolder := folder
	originalParentID := parentId
	t.Cleanup(func() {
		folder = originalFolder
		parentId = originalParentID
	})

	folder = "/Movies"
	parentId = "parent-123"

	resolver := fakeTargetResolver{
		getFileStat: func(gotParentID string, gotName string) (api.FileStat, error) {
			require.Equal(t, "parent-123", gotParentID)
			require.Equal(t, "Peppa.mp4", gotName)
			return api.FileStat{ID: "file-1", Name: "Peppa.mp4"}, nil
		},
		getFileByPath: func(path string) (api.FileStat, error) {
			return api.FileStat{}, errors.New("should not resolve by path")
		},
		getPathFolder: func(dirPath string) (string, error) {
			return "", errors.New("should not resolve folder id")
		},
	}

	stat, err := resolveDownloadTarget(resolver, "Peppa.mp4")
	require.NoError(t, err)
	require.Equal(t, "file-1", stat.ID)
}

func TestResolveDownloadTargetJoinsBasePathForNestedArg(t *testing.T) {
	originalFolder := folder
	originalParentID := parentId
	t.Cleanup(func() {
		folder = originalFolder
		parentId = originalParentID
	})

	folder = "/Movies"
	parentId = "parent-123"

	resolver := fakeTargetResolver{
		getFileStat: func(parentId string, name string) (api.FileStat, error) {
			return api.FileStat{}, errors.New("should not resolve direct child")
		},
		getFileByPath: func(path string) (api.FileStat, error) {
			require.Equal(t, filepath.Clean("/Movies/Kids/Peppa.mp4"), path)
			return api.FileStat{ID: "file-2", Name: "Peppa.mp4"}, nil
		},
		getPathFolder: func(dirPath string) (string, error) {
			return "", errors.New("should not resolve folder id")
		},
	}

	stat, err := resolveDownloadTarget(resolver, "Kids/Peppa.mp4")
	require.NoError(t, err)
	require.Equal(t, "file-2", stat.ID)
}

func TestResolveDownloadTargetWithoutArgsUsesBaseFolder(t *testing.T) {
	originalFolder := folder
	originalParentID := parentId
	t.Cleanup(func() {
		folder = originalFolder
		parentId = originalParentID
	})

	folder = "/Movies"
	parentId = ""

	resolver := fakeTargetResolver{
		getFileByPath: func(path string) (api.FileStat, error) {
			require.Equal(t, filepath.Clean("/Movies"), path)
			return api.FileStat{Kind: api.FileKindFolder, ID: "folder-1", Name: "Movies"}, nil
		},
		getFileStat: func(parentId string, name string) (api.FileStat, error) {
			return api.FileStat{}, errors.New("should not resolve by parent id")
		},
		getPathFolder: func(dirPath string) (string, error) {
			return "", errors.New("should not resolve folder id")
		},
	}

	stat, err := resolveDownloadTarget(resolver, "")
	require.NoError(t, err)
	require.Equal(t, "folder-1", stat.ID)
	require.Equal(t, api.FileKindFolder, stat.Kind)
}

func TestRequiresExplicitOutputFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringP("output", "o", ".", "")

	require.False(t, requiresExplicitOutputFlag(cmd, []string{"."}))
	require.True(t, requiresExplicitOutputFlag(cmd, []string{"file.txt", "."}))
	require.True(t, requiresExplicitOutputFlag(cmd, []string{"file.txt", ".."}))
	require.False(t, requiresExplicitOutputFlag(cmd, []string{"file.txt"}))

	require.NoError(t, cmd.Flags().Set("output", "."))
	require.False(t, requiresExplicitOutputFlag(cmd, []string{"file.txt", "."}))
}

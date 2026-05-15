package download

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

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

func TestParseTimeRangeSpec(t *testing.T) {
	closed, err := parseTimeRangeSpec("10-20")
	require.NoError(t, err)
	require.Equal(t, "10", closed.Start)
	require.Equal(t, "20", closed.End)

	openEnded, err := parseTimeRangeSpec("01:02-")
	require.NoError(t, err)
	require.Equal(t, "01:02", openEnded.Start)
	require.Empty(t, openEnded.End)
}

func TestParseTimeRangeSpecRejectsInvalidValues(t *testing.T) {
	for _, spec := range []string{"", "-10", "10", "a-10", "10-b", "10-20-30", "01::02-03:04"} {
		_, err := parseTimeRangeSpec(spec)
		require.Error(t, err, spec)
	}
}

func TestTimeRangeOutputName(t *testing.T) {
	require.Equal(t, "movie.10-20.mp4", timeRangeOutputName("movie.mp4", TimeRange{Start: "10", End: "20"}))
	require.Equal(t, "movie.01-02-end.mkv", timeRangeOutputName("movie.mkv", TimeRange{Start: "01:02"}))
}

func TestMediaClipSourceURLPrefersDefaultVisibleMedia(t *testing.T) {
	file := &api.File{}
	file.Links.ApplicationOctetStream.URL = "https://example.com/original"
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
			}{URL: "https://example.com/visible"},
			IsVisible: true,
		},
		{
			Link: struct {
				URL    string    `json:"url"`
				Token  string    `json:"token"`
				Expire time.Time `json:"expire"`
			}{URL: "https://example.com/default"},
			IsDefault: true,
			IsVisible: true,
		},
	}

	require.Equal(t, "https://example.com/default", mediaClipSourceURL(file))
}

func TestFFmpegClipperReportsMissingFFmpeg(t *testing.T) {
	t.Setenv("PATH", "")

	err := ffmpegClipper{}.Clip("https://example.com/video.mp4", TimeRange{Start: "0", End: "10"}, filepath.Join(t.TempDir(), "clip.mp4"))

	require.Error(t, err)
	require.Contains(t, err.Error(), "ffmpeg is required for --time-range")
	require.Contains(t, err.Error(), "install ffmpeg")
}

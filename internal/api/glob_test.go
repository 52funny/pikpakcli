package api

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeRemotePatternProvider struct {
	getPathFolderID       func(dirPath string) (string, error)
	getFolderFileStatList func(parentId string) ([]FileStat, error)
}

func (f fakeRemotePatternProvider) GetPathFolderId(dirPath string) (string, error) {
	return f.getPathFolderID(dirPath)
}

func (f fakeRemotePatternProvider) GetFolderFileStatList(parentId string) ([]FileStat, error) {
	return f.getFolderFileStatList(parentId)
}

func TestExpandRemotePatternsReturnsAbsoluteMatches(t *testing.T) {
	provider := fakeRemotePatternProvider{
		getPathFolderID: func(dirPath string) (string, error) {
			require.Equal(t, "/Movies", dirPath)
			return "movies-id", nil
		},
		getFolderFileStatList: func(parentId string) ([]FileStat, error) {
			require.Equal(t, "movies-id", parentId)
			return []FileStat{
				{Name: "a.mp4"},
				{Name: "b.mp4"},
				{Name: "note.txt"},
			}, nil
		},
	}

	matches, err := ExpandRemotePatterns(provider, "/Movies", []string{"*.mp4"}, false)
	require.NoError(t, err)
	require.Equal(t, []string{"/Movies/a.mp4", "/Movies/b.mp4"}, matches)
}

func TestExpandRemotePatternsCanKeepRelativeMatches(t *testing.T) {
	provider := fakeRemotePatternProvider{
		getPathFolderID: func(dirPath string) (string, error) {
			require.Equal(t, "/Movies/Kids", dirPath)
			return "kids-id", nil
		},
		getFolderFileStatList: func(parentId string) ([]FileStat, error) {
			require.Equal(t, "kids-id", parentId)
			return []FileStat{
				{Name: "a.srt"},
				{Name: "b.srt"},
			}, nil
		},
	}

	matches, err := ExpandRemotePatterns(provider, "/Movies", []string{"Kids/*.srt"}, true)
	require.NoError(t, err)
	require.Equal(t, []string{"Kids/a.srt", "Kids/b.srt"}, matches)
}

func TestExpandRemotePatternsReturnsNoMatchError(t *testing.T) {
	provider := fakeRemotePatternProvider{
		getPathFolderID: func(dirPath string) (string, error) {
			return "movies-id", nil
		},
		getFolderFileStatList: func(parentId string) ([]FileStat, error) {
			return []FileStat{{Name: "note.txt"}}, nil
		},
	}

	_, err := ExpandRemotePatterns(provider, "/Movies", []string{"*.mp4"}, false)
	require.EqualError(t, err, "no matches found for *.mp4")
}

func TestExpandRemotePatternsPropagatesLookupErrors(t *testing.T) {
	provider := fakeRemotePatternProvider{
		getPathFolderID: func(dirPath string) (string, error) {
			return "", errors.New("lookup failed")
		},
		getFolderFileStatList: func(parentId string) ([]FileStat, error) {
			return nil, errors.New("should not list")
		},
	}

	_, err := ExpandRemotePatterns(provider, "/Movies", []string{"Kids/*.mp4"}, false)
	require.EqualError(t, err, "lookup failed")
}

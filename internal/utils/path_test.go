package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitRemotePath(t *testing.T) {
	separator := string(filepath.Separator)

	tests := []struct {
		name     string
		input    string
		wantDir  string
		wantName string
	}{
		{
			name:     "full path",
			input:    separator + filepath.Join("Movies", "Peppa_Pig.mp4"),
			wantDir:  "Movies",
			wantName: "Peppa_Pig.mp4",
		},
		{
			name:     "relative nested path",
			input:    filepath.Join("Movies", "Kids", "Peppa_Pig.mp4"),
			wantDir:  filepath.Join("Movies", "Kids"),
			wantName: "Peppa_Pig.mp4",
		},
		{
			name:     "file name only",
			input:    "Peppa_Pig.mp4",
			wantDir:  "",
			wantName: "Peppa_Pig.mp4",
		},
		{
			name:     "root path",
			input:    separator,
			wantDir:  "",
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, name := SplitRemotePath(tt.input)
			require.Equal(t, tt.wantDir, dir)
			require.Equal(t, tt.wantName, name)
		})
	}
}

func TestExpandLocalPath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	require.Equal(t, home, ExpandLocalPath("~"))
	require.Equal(t, filepath.Join(home, "Downloads"), ExpandLocalPath("~/Downloads"))
	require.Equal(t, filepath.Join(home, "Downloads"), ExpandLocalPath("$HOME/Downloads"))
	require.Equal(t, "relative/path", ExpandLocalPath("relative/path"))
}

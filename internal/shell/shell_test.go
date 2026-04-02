package shell

import (
	"testing"

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

package download

import (
	"path/filepath"
	"testing"

	"github.com/52funny/pikpakcli/internal/api"
	"github.com/stretchr/testify/require"
)

func TestTrimRunes(t *testing.T) {
	require.Equal(t, "abcdef", trimRunes("abcdef", 6))
	require.Equal(t, "你好世...", trimRunes("你好世界欢迎你", 6))
}

func TestProgressDisplayNameIncludesParentDir(t *testing.T) {
	warp := warpFile{
		f:      &api.File{FileStat: api.FileStat{Name: "Peppa.mp4"}},
		output: filepath.Join("Film", "Kids"),
	}

	require.Equal(t, "Kids/Peppa.mp4", progressDisplayName(warp))
}

package save

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseShareID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"full https URL", "https://mypikpak.com/s/VNd8GjsfMiZjnBU575_1zmwIo1", "VNd8GjsfMiZjnBU575_1zmwIo1"},
		{"pan subdomain", "https://pan.mypikpak.com/s/VNd8GjsfMiZjnBU575_1zmwIo1", "VNd8GjsfMiZjnBU575_1zmwIo1"},
		{"no scheme", "mypikpak.com/s/abc123", "abc123"},
		{"trailing slash", "https://mypikpak.com/s/VNd8GjsfMiZjnBU575_1zmwIo1/", "VNd8GjsfMiZjnBU575_1zmwIo1"},
		{"url with subfolder", "https://mypikpak.com/s/VNd8GjsfMiZjnBU575_1zmwIo1/dd3e", "VNd8GjsfMiZjnBU575_1zmwIo1"},
		{"bare share id", "VNd8GjsfMiZjnBU575_1zmwIo1", "VNd8GjsfMiZjnBU575_1zmwIo1"},
		{"bare share id trailing slash", "VNd8GjsfMiZjnBU575_1zmwIo1/", "VNd8GjsfMiZjnBU575_1zmwIo1"},
		{"empty", "", ""},
		{"unrelated url", "https://example.com/foo/bar", ""},
		{"mypikpak root", "https://mypikpak.com", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, parseShareID(tt.in))
		})
	}
}

// A rejected share link makes RunE return an error, which cobra's Execute()
// turns into a non-zero exit code (see cli/root.go Execute).
func TestSaveCommandInvalidLinkReturnsError(t *testing.T) {
	err := SaveCommand.RunE(SaveCommand, []string{"https://example.com/foo/bar"})
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid share link")
}

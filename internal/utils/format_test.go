package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatStorage(t *testing.T) {
	assert.Equal(t, "2048", FormatStorage("2048", false))
	assert.Equal(t, "2KB", FormatStorage("2048", true))
	assert.Equal(t, "1.50KB", FormatStorage("1536", true))
	assert.Equal(t, "bad", FormatStorage("bad", true))
	assert.Equal(t, "1KB", FormatStorage("1024", true))
}

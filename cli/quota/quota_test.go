package quota

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatTransferValue(t *testing.T) {
	human = false
	assert.Equal(t, "2048", formatTransferValue(2048))

	human = true
	assert.Equal(t, "2KB", formatTransferValue(2048))
}

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuotaRemaining(t *testing.T) {
	remaining, err := (Quota{Limit: "10", Usage: "3"}).Remaining()
	require.NoError(t, err)
	assert.Equal(t, int64(7), remaining)
}

func TestQuotaRemainingInvalid(t *testing.T) {
	_, err := (Quota{Limit: "bad", Usage: "3"}).Remaining()
	require.Error(t, err)
}

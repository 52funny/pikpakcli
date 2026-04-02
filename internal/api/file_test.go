package api

import (
	"errors"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeNetError struct{}

func (fakeNetError) Error() string   { return "i/o timeout" }
func (fakeNetError) Timeout() bool   { return true }
func (fakeNetError) Temporary() bool { return true }

func TestIsRetryableListError(t *testing.T) {
	assert.True(t, isRetryableListError(io.ErrUnexpectedEOF))
	assert.True(t, isRetryableListError(errors.New("unexpected EOF")))
	assert.True(t, isRetryableListError(fakeNetError{}))
	assert.True(t, isRetryableListError(errors.New("read: connection reset by peer")))
	assert.False(t, isRetryableListError(errors.New("permission denied")))
	assert.False(t, isRetryableListError(nil))
}

func TestFakeNetErrorImplementsNetError(t *testing.T) {
	var err net.Error = fakeNetError{}
	assert.True(t, err.Timeout())
	assert.True(t, err.Temporary())
}

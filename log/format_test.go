package log

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTextFormat(t *testing.T) {
	notty, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
	require.NoError(t, err)
	stdin := os.Stdin
	os.Stdin = notty
	defer func() {
		os.Stdin = stdin
	}()
	assert.Equal(t, textFormat, GetTextFormat())
}

func TestGetTextFormatTTY(t *testing.T) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	require.NoError(t, err)
	stdin := os.Stdin
	os.Stdin = tty
	defer func() {
		os.Stdin = stdin
	}()
	assert.Equal(t, textFormatTTY, GetTextFormat())
}

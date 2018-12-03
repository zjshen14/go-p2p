package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureIPv4(t *testing.T) {
	ip, err := EnsureIPv4("localhost")
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", ip)

	ip, err = EnsureIPv4("127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", ip)

	_, err = EnsureIPv4("unknown")
	require.Error(t, err)
}

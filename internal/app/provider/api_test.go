package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIRuntime_InvalidDatabaseURL(t *testing.T) {
	t.Parallel()

	_, err := NewAPIRuntime(context.Background(), Config{DatabaseURL: "://invalid"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connect postgres")
}

func TestNewAPIRuntime_RedisUnreachable(t *testing.T) {
	t.Parallel()

	// pgxpool.New is lazy — a valid-format URL succeeds even with no DB running.
	// Port 1 on loopback is always closed, so Redis Ping returns "connection refused" immediately.
	_, err := NewAPIRuntime(context.Background(), Config{
		DatabaseURL:  "postgres://user:pass@127.0.0.1:5432/db",
		RedisAddress: "127.0.0.1:1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connect redis")
}

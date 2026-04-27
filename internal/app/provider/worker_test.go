package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorkerRuntime_InvalidDatabaseURL(t *testing.T) {
	t.Parallel()

	_, err := NewWorkerRuntime(context.Background(), Config{DatabaseURL: "://invalid"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connect postgres")
}

func TestNewWorkerRuntime_ValidURLBuildsRuntime(t *testing.T) {
	t.Parallel()

	// pgxpool.New and kafka clients are lazy — a valid-format URL succeeds even
	// without a running DB or broker. The runtime should be constructed without error.
	runtime, err := NewWorkerRuntime(context.Background(), Config{
		DatabaseURL:         "postgres://user:pass@127.0.0.1:5432/db",
		KafkaBrokers:        []string{"localhost:9092"},
		WorkerPoolSize:      1,
		WorkerPoolQueueSize: 10,
	})
	require.NoError(t, err)
	assert.NotNil(t, runtime.Run)
	assert.NotNil(t, runtime.Close)
	runtime.Close()
}

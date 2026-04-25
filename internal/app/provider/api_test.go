package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloserStack_CloseRunsInLIFOOrder(t *testing.T) {
	t.Parallel()

	stack := &closerStack{}
	order := make([]string, 0, 2)

	stack.Add(func() { order = append(order, "first") })
	stack.Add(func() { order = append(order, "second") })

	err := stack.Close()
	require.NoError(t, err)
	assert.Equal(t, []string{"second", "first"}, order)
}

func TestCloserStack_CloseJoinsErrors(t *testing.T) {
	t.Parallel()

	errFirst := errors.New("first")
	errSecond := errors.New("second")
	stack := &closerStack{}

	stack.AddWithError(func() error { return errFirst })
	stack.AddWithError(func() error { return errSecond })

	err := stack.Close()
	require.Error(t, err)
	assert.ErrorIs(t, err, errFirst)
	assert.ErrorIs(t, err, errSecond)
}

func TestNewRuntime_InvalidDatabaseURL(t *testing.T) {
	t.Parallel()

	_, err := NewRuntime(context.Background(), Config{DatabaseURL: "://invalid"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connect postgres")
}

package provider

import (
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

func TestCloserStack_CloseEmptyStackReturnsNil(t *testing.T) {
	t.Parallel()

	stack := &closerStack{}

	err := stack.Close()
	require.NoError(t, err)
}

func TestCloserStack_CloseSkipsNilErrorFromAddWithError(t *testing.T) {
	t.Parallel()

	stack := &closerStack{}
	stack.AddWithError(func() error { return nil })

	err := stack.Close()
	require.NoError(t, err)
}

func TestCloserStack_AddMixedRunsAllInLIFOOrder(t *testing.T) {
	t.Parallel()

	errC := errors.New("c failed")
	stack := &closerStack{}
	order := make([]string, 0, 3)

	stack.Add(func() { order = append(order, "a") })
	stack.AddWithError(func() error { order = append(order, "b"); return nil })
	stack.AddWithError(func() error { order = append(order, "c"); return errC })

	err := stack.Close()
	require.Error(t, err)
	assert.ErrorIs(t, err, errC)
	assert.Equal(t, []string{"c", "b", "a"}, order)
}

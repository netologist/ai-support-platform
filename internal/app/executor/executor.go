package executor

import "context"

// Executor defines a generic command executor for operations.
type Executor[T any, R any] interface {
	Execute(ctx context.Context, cmd T) (R, error)
}

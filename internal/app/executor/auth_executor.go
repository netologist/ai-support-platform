package executor

import (
	"context"

	"github.com/netologist/ai-support-platform/internal/app/command"
)

type LoginExecutor interface {
	Execute(ctx context.Context, cmd command.LoginCommand) (command.LoginResult, error)
}

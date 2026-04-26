package executor

import (
	"github.com/netologist/ai-support-platform/internal/app/command"
)

type LoginExecutor Executor[command.LoginCommand, command.LoginResult]

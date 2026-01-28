package agents

import (
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
)

type Agent interface {
	SelectAction(go_ *environment.Go_) environment.Action
}

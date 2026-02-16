package agents

import (
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
)

type Agent interface {
	SelectAction(go_ *environment.Game) environment.Action
}

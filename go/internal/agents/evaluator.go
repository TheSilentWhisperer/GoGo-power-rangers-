package agents

import (
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/utils"
)

type Evaluator interface {
	GetEvaluationQueue() *utils.LockedQueue[utils.Triple[MctsNode, int, *environment.Game]]
}

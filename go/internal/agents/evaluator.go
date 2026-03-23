package agents

import (
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/utils"
)

type Evaluator interface {
	Evaluate(utils.Triple[MctsNode, int, *environment.Game]) utils.Triple[MctsNode, float64, *environment.Game]
	GetEvaluationQueue() *utils.LockedQueue[utils.Triple[MctsNode, int, *environment.Game]]
}

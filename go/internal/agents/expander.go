package agents

import (
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/utils"
)

type Expander interface {
	ExpandAndEvaluate(utils.Triple[MctsNode, int, *environment.Game]) int
	GetToExpand() chan utils.Triple[MctsNode, int, *environment.Game]
}

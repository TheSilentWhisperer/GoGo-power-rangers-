package agents

import (
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/utils"
)

type Expander interface {
	ExpandAndEvaluate(utils.Triple[MctsNode, int, *environment.Game]) int
	GetToExpand() chan utils.Triple[MctsNode, int, *environment.Game]
}

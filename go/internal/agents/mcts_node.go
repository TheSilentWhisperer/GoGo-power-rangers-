package agents

import "github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"

type MctsNode interface {
	SelectBestChildIndex() int
	UpdateStats(value float64, action_idx int)
	ExpandChild(child_idx int, game *environment.Game)
	GetParent() MctsNode
	GetIdx() int
	GetN() []int
	GetTotalN() int
	GetQ() []float64
	GetChildren() []MctsNode
	GetIsEvaluating(action_idx int) bool
	SetIsEvaluating(action_idx int, value bool)
	GetIsTerminal(action_idx int) bool
	SetIsTerminal(action_idx int, value bool)
	GetHasPriors() bool
}

package agents

import "github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"

type MctsNode interface {
	Reset(game *environment.Game)
	SelectBestChildIndex() int
	UpdateStats(value int, action_idx int)
	GetParent() MctsNode
	GetIdx() int
	GetN() []int
	GetQ() []float64
	GetChildren() []MctsNode
	GetIsExpanded() []int32
}

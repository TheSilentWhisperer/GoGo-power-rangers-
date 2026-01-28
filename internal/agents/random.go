package agents

import (
	"math/rand"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
)

type randomAgent struct{}

func NewRandomAgent() *randomAgent {
	return &randomAgent{}
}

func (ra *randomAgent) SelectAction(go_ *environment.Go_) environment.Action {
	var legalActions []environment.Action = go_.Legal_actions
	var n int = len(legalActions)
	var r int = rand.Intn(n)
	return legalActions[r]
}

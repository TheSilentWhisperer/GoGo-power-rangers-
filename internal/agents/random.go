package agents

import (
	"math/rand"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
)

type randomAgent struct{}

func NewRandomAgent() *randomAgent {
	return &randomAgent{}
}

func (ra *randomAgent) SelectAction(game *environment.Game) environment.Action {
	var legalActions []environment.Action = game.LegalActions
	var n int = len(legalActions)
	var r int = rand.Intn(n)
	return legalActions[r]
}

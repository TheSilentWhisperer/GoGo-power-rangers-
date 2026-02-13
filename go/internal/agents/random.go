package agents

import (
	"math/rand"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
)

type RandomAgent struct{}

func NewRandomAgent() *RandomAgent {
	return &RandomAgent{}
}

func (ra *RandomAgent) SelectAction(game *environment.Game) environment.Action {
	var legal_actions []environment.Action = game.LegalActions
	var n int = len(legal_actions)
	var r int = rand.Intn(n-1) + 1 // Exclude the last action (resign) to make the agent more competitive
	return legal_actions[r]
}

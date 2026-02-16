package agents

import "github.com/TheSilentWhisperer/GoGo-power-rangers-/go/gen/proto/remote_trainer"

func NewPuctAgent(simulations_per_move int, nb_routines int, resign_threshold float64, client remote_trainer.PositionEvaluatorClient) *MctsAgent {
	var expander *PuctExpander = NewPuctExpander(nb_routines, client)
	return NewMctsAgent(simulations_per_move, nb_routines, resign_threshold, expander)
}

package agents

import (
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/gen/proto/remote_trainer"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/utils"
)

func NewPuctAgent(simulations_per_move int, nb_routines int, max_parallel_searches int, resign_threshold float64, client remote_trainer.PositionEvaluatorClient, epsilon float64, alpha float64) *MctsAgent {
	var agent *MctsAgent = &MctsAgent{
		SimulationsPerMove:   simulations_per_move,
		Epsilon:              epsilon,
		Alpha:                alpha,
		SimulationsDone:      utils.NewLockedValue(0),
		NbRoutines:           nb_routines,
		MaxParallelSearch:    max_parallel_searches,
		CurrentNbSearches:    utils.NewLockedValue(0),
		ExpansionQueue:       utils.NewLockedQueue[utils.Triple[MctsNode, int, *environment.Game]](max_parallel_searches),
		BackpropagationQueue: utils.NewLockedQueue[utils.Pair[MctsNode, float64]](max_parallel_searches),
		ResignThreshold:      resign_threshold,
	}
	agent.Evaluator = NewPuctEvaluator(max_parallel_searches, client, agent.SimulationsDone, agent.SimulationsPerMove)
	return agent
}

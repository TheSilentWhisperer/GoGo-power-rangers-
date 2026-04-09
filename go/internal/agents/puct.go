package agents

import (
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/gen/proto/remote_trainer"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/utils"
)

func NewPuctAgent(simulations_per_move int, nb_routines int, max_parallel_searches int, resign_threshold float64, client remote_trainer.PositionEvaluatorClient, epsilon float64, alpha float64) *MctsAgent {
	return NewPuctAgentWithMode(simulations_per_move, nb_routines, max_parallel_searches, resign_threshold, client, epsilon, alpha, false)
}

func NewPuctAgentEval(simulations_per_move int, nb_routines int, max_parallel_searches int, resign_threshold float64, client remote_trainer.PositionEvaluatorClient) *MctsAgent {
	return NewPuctAgentWithMode(simulations_per_move, nb_routines, max_parallel_searches, resign_threshold, client, 0, 0, true)
}

func NewPuctAgentWithMode(simulations_per_move int, nb_routines int, max_parallel_searches int, resign_threshold float64, client remote_trainer.PositionEvaluatorClient, epsilon float64, alpha float64, evaluation_mode bool) *MctsAgent {
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
		EvaluationMode:       evaluation_mode,
	}
	agent.Evaluator = NewPuctEvaluator(max_parallel_searches, client, agent.SimulationsDone, agent.SimulationsPerMove)
	return agent
}

package agents

import (
	"context"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/gen/proto/remote_trainer"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
)

type PUCTAgent struct {
	*MCTSAgent
	Client remote_trainer.PositionEvaluatorClient
}

// Constructor
func NewPUCTAgent(simulations_per_move int, nb_routines int, resign_threshold float64, client remote_trainer.PositionEvaluatorClient) *PUCTAgent {
	return &PUCTAgent{
		MCTSAgent: NewMCTSAgent(simulations_per_move, nb_routines, resign_threshold),
		Client:    client,
	}
}

func (agent *PUCTAgent) Expand(node *MctsNode, action_idx int, game *environment.Game) {
	// just use uniform priors for now, we will replace this with a neural network evaluation later
	var priors []float64 = make([]float64, node.K)
	for i := 0; i < node.K; i++ {
		priors[i] = 1.0 / float64(node.K)
	}
	var expanded_child *PuctNode = NewPuctNode(game, node, action_idx, priors)
	node.Children[action_idx] = &expanded_child.MctsNode
}

func (agent *PUCTAgent) Evaluate(simulations_per_move int, nb_routines int) int {
	// just use a dummy evaluation for now, we will replace this with a neural network evaluation later
	var request remote_trainer.EvaluatePositionRequest = remote_trainer.EvaluatePositionRequest{
		X: 31,
		Y: 12,
	}
	response, err := agent.Client.EvaluatePosition(context.Background(), &request)
	if err != nil {
		println("Error evaluating position:", err.Error())
		return 0
	}
	var value int64 = response.Z

	println("Evaluated position with value:", value)
	return 0
}

func (agent *PUCTAgent) ExpandAndEvaluate(to_expand ExpandTuple, game *environment.Game) int {
	agent.Expand(to_expand.First, to_expand.Second, game)
	// Playing the action to reach the expanded child made so the opponent (expanded child) had this final value
	var value int = agent.Evaluate(2, 4)
	// So the value for the parent node is the negation of this value
	return -value
}

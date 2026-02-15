package agents

import (
	"context"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/gen/proto/remote_trainer"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/utils"
)

// LockedValue moved to internal/utils

type PuctExpander struct {
	*UctExpander
	Client remote_trainer.PositionEvaluatorClient
}

func NewPuctExpander(nb_routines int, client remote_trainer.PositionEvaluatorClient) *PuctExpander {
	var UctExpander *UctExpander = NewUctExpander(nb_routines)
	return &PuctExpander{
		UctExpander: UctExpander,
		Client:      client,
	}
}

func (expander *PuctExpander) GetToExpand() chan utils.Triple[MctsNode, int, *environment.Game] {
	return expander.ToExpand
}

func (expander *PuctExpander) Expand(to_expand utils.Triple[MctsNode, int, *environment.Game]) {
	var node MctsNode = to_expand.First
	var child_idx int = to_expand.Second
	var game *environment.Game = to_expand.Third
	game.PlayAction(game.LegalActions[child_idx])
	var child_node MctsNode = NewPuctNode(game, node, child_idx, expander.Client) // We will set the priors later when we have the neural network evaluation
	node.GetChildren()[child_idx] = child_node
}

func (agent *PuctExpander) Evaluate(game *environment.Game) utils.Pair[int, []float64] {
	// just use a dummy evaluation for now, we will replace this with a neural network evaluation later
	var request remote_trainer.EvaluatePositionRequest = remote_trainer.EvaluatePositionRequest{
		X: 31,
		Y: 12,
	}
	response, err := agent.Client.EvaluatePosition(context.Background(), &request)
	if err != nil {
		println("Error evaluating position:", err.Error())
		return utils.NewPair(0, make([]float64, 0))
	}
	var value int64 = response.Z

	println("Evaluated position with value:", value)
	return utils.NewPair(0, make([]float64, 0)) // We will set the priors later when we have the neural network evaluation
}

func (agent *PuctExpander) ExpandAndEvaluate(to_expand utils.Triple[MctsNode, int, *environment.Game]) int {
	println("Expanding and evaluating a node...")

	agent.Expand(to_expand)
	// Playing the action to reach the expanded child made so the opponent (expanded child) had this final value
	var evaluation utils.Pair[int, []float64] = agent.Evaluate(to_expand.Third)
	var value int = evaluation.First
	var priors []float64 = evaluation.Second

	// Set the priors for the expanded child node
	var child_node *PuctNode = to_expand.First.GetChildren()[to_expand.Second].(*PuctNode)
	child_node.P = priors
	// So the value for the parent node is the negation of this value
	return -value
}

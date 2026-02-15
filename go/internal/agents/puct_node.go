package agents

import (
	"context"
	"math"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/gen/proto/remote_trainer"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
)

type PuctNode struct {
	*UctNode
	P      []float64 // Prior probabilities for each action
	Client remote_trainer.PositionEvaluatorClient
}

// Constructor
func NewPuctNode(game *environment.Game, parent MctsNode, idx int, client remote_trainer.PositionEvaluatorClient) *PuctNode {
	return &PuctNode{
		UctNode: &UctNode{
			Parent:     parent,
			Idx:        idx,
			K:          len(game.LegalActions),
			TotalN:     0,
			N:          make([]int, len(game.LegalActions)),
			Q:          make([]float64, len(game.LegalActions)),
			Children:   make([]MctsNode, len(game.LegalActions)),
			IsExpanded: make([]int32, len(game.LegalActions)),
		},
		P:      make([]float64, len(game.LegalActions)),
		Client: client,
	}
}

// Getters
func (node *PuctNode) GetParent() MctsNode {
	return node.Parent
}

func (node *PuctNode) GetIdx() int {
	return node.Idx
}

func (node *PuctNode) GetN() []int {
	return node.N
}

func (node *PuctNode) GetQ() []float64 {
	return node.Q
}

func (node *PuctNode) GetChildren() []MctsNode {
	return node.Children
}

func (node *PuctNode) GetIsExpanded() []int32 {
	return node.IsExpanded
}

// Methods
func (node *PuctNode) Reset(game *environment.Game) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	node.TotalN = 0
	for i := 0; i < node.K; i++ {
		node.N[i] = 0
		node.Q[i] = 0
		node.Children[i] = nil
		node.IsExpanded[i] = 0
	}

	var request remote_trainer.EvaluatePositionRequest = remote_trainer.EvaluatePositionRequest{
		X: 31,
		Y: 12,
	}
	response, err := node.Client.EvaluatePosition(context.Background(), &request)
	if err != nil {
		println("Error evaluating position:", err.Error())
		return
	}
	var _ int64 = response.Z

	var priors []float64 = make([]float64, 0)
	node.P = priors
}

func (node *PuctNode) SelectBestChildIndex() int {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	//By default, we use UCT (Upper Confidence Bound for Trees) with exploration constant sqrt(2)

	var c float64 = 1.0
	var best_action_idx int
	var best_value float64 = math.Inf(-1)
	for action_idx := 0; action_idx < node.K; action_idx++ {
		var exploration_term float64 = node.P[action_idx] * math.Sqrt(float64(node.TotalN)) / (1 + float64(node.N[action_idx]))
		var puct_value float64 = node.Q[action_idx] + c*exploration_term
		if puct_value > best_value {
			best_value = puct_value
			best_action_idx = action_idx
		}
	}

	// Add virtual loss
	node.TotalN += 1
	node.N[best_action_idx] += 1
	node.Q[best_action_idx] += (-1 - node.Q[best_action_idx]) / float64(node.N[best_action_idx]) // Pessimisticly suppose the value is -1

	return best_action_idx
}

func (node *PuctNode) UpdateStats(value int, action_idx int) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	// We artificially added a visit which resulted in a value of -1, replace it with the actual value
	node.Q[action_idx] += float64(value+1) / float64(node.N[action_idx])
}

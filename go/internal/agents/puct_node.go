package agents

import (
	"math"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
)

type PuctNode struct {
	MctsNode
	P []float64 // Prior probabilities for each action
}

func NewPuctNode(game *environment.Game, parent *MctsNode, idx int, priors []float64) *PuctNode {
	return &PuctNode{
		MctsNode: MctsNode{
			Parent:     parent,
			Idx:        idx,
			K:          len(game.LegalActions),
			TotalN:     0,
			N:          make([]int, len(game.LegalActions)),
			Q:          make([]float64, len(game.LegalActions)),
			Children:   make([]*MctsNode, len(game.LegalActions)),
			IsExpanded: make([]int32, len(game.LegalActions)),
		},
		P: priors,
	}
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

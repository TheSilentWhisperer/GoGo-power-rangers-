package agents

import (
	"math"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
)

type PuctNode struct {
	*UctNode
	P []float64 // Prior probabilities for each action
}

// Constructor
func NewPuctNode(game *environment.Game, parent MctsNode, idx int) *PuctNode {
	return &PuctNode{
		UctNode: &UctNode{
			Parent:       parent,
			Idx:          idx,
			K:            len(game.LegalActions),
			TotalN:       0,
			N:            make([]int, len(game.LegalActions)),
			Q:            make([]float64, len(game.LegalActions)),
			Children:     make([]MctsNode, len(game.LegalActions)),
			IsEvaluating: make([]bool, len(game.LegalActions)),
			IsTerminal:   make([]bool, len(game.LegalActions)),
		},
		P: make([]float64, len(game.LegalActions)),
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

func (node *PuctNode) GetIsEvaluating(action_idx int) bool {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	return node.IsEvaluating[action_idx]
}

func (node *PuctNode) SetIsEvaluating(action_idx int, value bool) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	node.IsEvaluating[action_idx] = value
}

func (node *PuctNode) GetIsTerminal(action_idx int) bool {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	return node.IsTerminal[action_idx]
}

func (node *PuctNode) SetIsTerminal(action_idx int, value bool) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	node.IsTerminal[action_idx] = value
}

func (node *PuctNode) SetPriors(priors []float64) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	copy(node.P, priors)
}

func (node *PuctNode) ExpandChild(child_idx int, game *environment.Game) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	if game.IsTerminal() {
		node.IsTerminal[child_idx] = true
	}

	if node.Children[child_idx] == nil {
		node.Children[child_idx] = NewPuctNode(game, node, child_idx)
		node.IsEvaluating[child_idx] = true
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

func (node *PuctNode) UpdateStats(value float64, action_idx int) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	// We artificially added a visit which resulted in a value of -1, replace it with the actual value
	node.Q[action_idx] += float64(value+1) / float64(node.N[action_idx])
}

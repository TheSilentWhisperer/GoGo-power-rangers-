package agents

import (
	"math"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
)

type PuctNode struct {
	*UctNode
	P          []float64 // Prior probabilities for each action
	HasPriors  bool      // Track if priors have been set (prevents selection before priors arrive)
}

// Constructor
func NewPuctNode(game *environment.Game, parent MctsNode, idx int, P []float64) *PuctNode {

	var Q []float64 = make([]float64, len(game.LegalActions))
	for i := range Q {
		Q[i] = 0.0
	}

	return &PuctNode{
		UctNode: &UctNode{
			Parent:       parent,
			Idx:          idx,
			K:            len(game.LegalActions),
			TotalN:       0,
			N:            make([]int, len(game.LegalActions)),
			W:            make([]float64, len(game.LegalActions)),
			Q:            Q,
			Children:     make([]MctsNode, len(game.LegalActions)),
			IsEvaluating: make([]bool, len(game.LegalActions)),
			IsTerminal:   make([]bool, len(game.LegalActions)),
		},
		P:         P,
		HasPriors: len(P) > 0 && P[0] != 0, // True if priors are non-empty/non-zero
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
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	// Return a copy to avoid external modifications
	n_copy := make([]int, len(node.N))
	copy(n_copy, node.N)
	return n_copy
}

func (node *PuctNode) GetTotalN() int {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	return node.TotalN
}

func (node *PuctNode) GetQ() []float64 {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	// Return a copy to avoid external modifications
	q_copy := make([]float64, len(node.Q))
	copy(q_copy, node.Q)
	return q_copy
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
	node.HasPriors = true
}

func (node *PuctNode) GetHasPriors() bool {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	return node.HasPriors
}

func (node *PuctNode) ExpandChild(child_idx int, game *environment.Game) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	if game.IsTerminal() {
		node.IsTerminal[child_idx] = true
	}

	if node.Children[child_idx] == nil {
		node.Children[child_idx] = NewPuctNode(game, node, child_idx, make([]float64, len(game.LegalActions)))
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
	for action_idx := 1; action_idx < node.K; action_idx++ {
		var exploration_term float64 = node.P[action_idx] * math.Sqrt(float64(node.TotalN)) / (1 + float64(node.N[action_idx]))
		var puct_value float64 = node.Q[action_idx] + c*exploration_term
		if puct_value > best_value {
			best_value = puct_value
			best_action_idx = action_idx
		}
	}

	// Add virtual loss: increment visit counts so other selectors prefer other actions.
	// Do NOT overwrite the stored Q average here — use counts to reduce selection probability.

	node.TotalN += 1
	node.N[best_action_idx] += 1
	return best_action_idx
}

func (node *PuctNode) UpdateStats(value float64, action_idx int) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	node.W[action_idx] += value
	node.Q[action_idx] = node.W[action_idx] / float64(node.N[action_idx])
}

package agents

import (
	"math"
	"sync"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
)

type UctNode struct {
	Mutex        sync.Mutex
	Parent       MctsNode
	Idx          int       // Index of the action taken to reach this node from its parent
	K            int       // Number of legal actions
	TotalN       int       // Total visit count
	N            []int     // Visit counts for each action
	W            []float64 // Total reward for each action
	Q            []float64 // Total reward for each action
	Children     []MctsNode
	IsEvaluating []bool
	IsTerminal   []bool
}

// Constructor
func NewUctNode(game *environment.Game, parent MctsNode, idx int) *UctNode {
	return &UctNode{
		Parent:       parent,
		Idx:          idx,
		K:            len(game.LegalActions),
		TotalN:       0,
		N:            make([]int, len(game.LegalActions)),
		W:            make([]float64, len(game.LegalActions)),
		Q:            make([]float64, len(game.LegalActions)),
		Children:     make([]MctsNode, len(game.LegalActions)),
		IsEvaluating: make([]bool, len(game.LegalActions)),
		IsTerminal:   make([]bool, len(game.LegalActions)),
	}
}

// Getters
func (node *UctNode) GetParent() MctsNode {
	return node.Parent
}

func (node *UctNode) GetIdx() int {
	return node.Idx
}

func (node *UctNode) GetN() []int {
	return node.N
}

func (node *UctNode) GetTotalN() int {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	return node.TotalN
}

func (node *UctNode) GetQ() []float64 {
	return node.Q
}

func (node *UctNode) GetChildren() []MctsNode {
	return node.Children
}

func (node *UctNode) GetIsEvaluating(action_idx int) bool {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	return node.IsEvaluating[action_idx]
}

func (node *UctNode) SetIsEvaluating(action_idx int, value bool) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	node.IsEvaluating[action_idx] = value
}

func (node *UctNode) GetIsTerminal(action_idx int) bool {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	return node.IsTerminal[action_idx]
}

func (node *UctNode) SetIsTerminal(action_idx int, value bool) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	node.IsTerminal[action_idx] = value
}

func (node *UctNode) GetHasPriors() bool {
	// UCT nodes don't use priors, so always return true
	return true
}

// Methods
func (node *UctNode) ExpandChild(child_idx int, game *environment.Game) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	if game.IsTerminal() {
		node.IsTerminal[child_idx] = true
	}

	if node.Children[child_idx] == nil {
		node.Children[child_idx] = NewUctNode(game, node, child_idx)
		node.IsEvaluating[child_idx] = true
	}
}

func (node *UctNode) SelectBestChildIndex() int {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	//By default, we use UCT (Upper Confidence Bound for Trees) with exploration constant sqrt(2)

	var c float64 = math.Sqrt(2)
	var best_action_idx int = -1
	var best_value float64 = math.Inf(-1)
	for action_idx := 0; action_idx < node.K; action_idx++ {
		var exploration_term float64
		if node.N[action_idx] == 0 {
			exploration_term = math.Inf(1)
		} else {
			exploration_term = c * math.Sqrt(math.Log(float64(node.TotalN))/float64(node.N[action_idx]))
		}
		var uct_value float64 = node.Q[action_idx] + exploration_term
		if uct_value > best_value {
			best_value = uct_value
			best_action_idx = action_idx
		}
	}

	// Add virtual loss: increment visit counts so other selectors prefer other actions.
	// Do NOT overwrite the stored Q average here — use counts to reduce selection probability.
	node.TotalN += 1
	node.N[best_action_idx] += 1

	return best_action_idx
}

func (node *UctNode) UpdateStats(value float64, action_idx int) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	// Update running average Q for the action using online update formula
	if node.N[action_idx] <= 0 {
		node.Q[action_idx] = value
	} else {
		node.Q[action_idx] += (value - node.Q[action_idx]) / float64(node.N[action_idx])
	}
}

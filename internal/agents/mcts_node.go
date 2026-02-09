package agents

import (
	"math"
	"sync"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
)

type MctsNode struct {
	Mutex      sync.Mutex
	Parent     *MctsNode
	Idx        int       // Index of the action taken to reach this node from its parent
	K          int       // Number of legal actions
	TotalN     int       // Total visit count
	N          []int     // Visit counts for each action
	Q          []float64 // Total reward for each action
	Children   []*MctsNode
	IsExpanded []int32 // Atomic boolean flags to indicate if child nodes are expanded
}

// Constructor
func NewMctsNode(game *environment.Game, parent *MctsNode, idx int) *MctsNode {
	return &MctsNode{
		Parent:     parent,
		Idx:        idx,
		K:          len(game.LegalActions),
		TotalN:     0,
		N:          make([]int, len(game.LegalActions)),
		Q:          make([]float64, len(game.LegalActions)),
		Children:   make([]*MctsNode, len(game.LegalActions)),
		IsExpanded: make([]int32, len(game.LegalActions)),
	}
}

// Methods
func (node *MctsNode) SelectBestChildIndex() int {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	//By default, we use UCT (Upper Confidence Bound for Trees) with exploration constant sqrt(2)

	var c float64 = math.Sqrt(2)
	var best_action_idx int
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

	// Add virtual loss
	node.TotalN += 1
	node.N[best_action_idx] += 1
	node.Q[best_action_idx] += (-1 - node.Q[best_action_idx]) / float64(node.N[best_action_idx]) // Pessimisticly suppose the value is -1

	return best_action_idx
}

func (node *MctsNode) UpdateStats(value int, action_idx int) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	// We artificially added a visit which resulted in a value of -1, replace it with the actual value
	node.Q[action_idx] += float64(value+1) / float64(node.N[action_idx])
}

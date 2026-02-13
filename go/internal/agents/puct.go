package agents

import (
	"sync"
	"sync/atomic"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/gen/go/position_evaluation"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/utils"
)

type PUCTAgent struct {
	SimulationsPerMove  int
	SimulationsDone     *utils.LockedValue
	NbRoutines          int
	Root                *MctsNode
	ToBackpropagate     chan BackpropagateTuple
	ToExpandAndEvaluate chan ExpandTuple
	ResignThreshold     float64
	Client              position_evaluation.PositionEvaluatorClient
}

// Constructor
func NewPUCTAgent(simulations_per_move int, nb_routines int, resign_threshold float64) *PUCTAgent {
	return &PUCTAgent{
		SimulationsPerMove:  simulations_per_move,
		NbRoutines:          nb_routines,
		ToBackpropagate:     make(chan BackpropagateTuple, nb_routines),
		ToExpandAndEvaluate: make(chan ExpandTuple, nb_routines),
		ResignThreshold:     resign_threshold,
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

}

func (agent *PUCTAgent) ExpandAndEvaluate(to_expand ExpandTuple, game *environment.Game) int {
	agent.Expand(to_expand.First, to_expand.Second, game)
	// Playing the action to reach the expanded child made so the opponent (expanded child) had this final value
	var value int = agent.Evaluate(2, 4)
	// So the value for the parent node is the negation of this value
	return -value
}

func (agent *MCTSAgent) Backpropagate(to_backpropagate BackpropagateTuple) {
	var node *MctsNode = to_backpropagate.First
	var value int = to_backpropagate.Second
	if node.Parent != nil {
		node.Parent.UpdateStats(value, node.Idx)
		agent.Backpropagate(utils.NewTriple(node.Parent, -value, (*environment.Game)(nil)))
	}
}

func (agent *MCTSAgent) ExploreTree(wg *sync.WaitGroup, game *environment.Game) {

	defer wg.Done()
	for agent.SimulationsDone.Get() < agent.SimulationsPerMove {
		select {
		case to_backpropagate := <-agent.ToBackpropagate:
			agent.Backpropagate(to_backpropagate)
		case to_expand := <-agent.ToExpandAndEvaluate:
			if to_expand.Second == -1 {
				// Terminal node reached, no expansion
				agent.SimulationsDone.Incr() // We are sure to expand a new node
				var value int                // Value of the game for the parent of the backpropagated node (terminal node)
				var winner environment.Stone = to_expand.Third.GetWinner()
				switch winner {
				case environment.Empty:
					value = 0 // Draw
				case to_expand.Third.Board.CurrentPlayer:
					value = -1 // Loss for the parent
				default:
					value = 1 // Win for the parent
				}

				agent.ToBackpropagate <- utils.NewTriple(to_expand.First, value, (*environment.Game)(nil))
				continue
			}
			// atomic check to avoid expanding the same node multiple times
			if atomic.CompareAndSwapInt32((&to_expand.First.IsExpanded[to_expand.Second]), 0, 1) {
				agent.SimulationsDone.Incr()                                        // We are sure to expand a new node
				var value int = agent.ExpandAndEvaluate(to_expand, to_expand.Third) // Value of the game for the parent of the backpropagated node (expanded node)
				var expanded_child *MctsNode = to_expand.First.Children[to_expand.Second]
				agent.ToBackpropagate <- utils.NewTriple(expanded_child, value, (*environment.Game)(nil))
			}

		default:
			var game_copy *environment.Game = game.DeepCopy()
			var to_expand ExpandTuple = agent.SelectLeaf(agent.Root, game_copy)
			agent.ToExpandAndEvaluate <- to_expand
		}
	}
}

func (agent *MCTSAgent) SelectAction(game *environment.Game) environment.Action {

	// reset MCTS tree
	agent.Root = NewMctsNode(game, nil, -1)
	agent.SimulationsDone = utils.NewLockedValue(0)

	var wg sync.WaitGroup
	wg.Add(agent.NbRoutines)

	for nb_routines := 0; nb_routines < agent.NbRoutines; nb_routines++ {

		go agent.ExploreTree(&wg, game)
	}

	wg.Wait()

	var final_action environment.Action = agent.GetFinalAction(game.LegalActions)
	return final_action
}

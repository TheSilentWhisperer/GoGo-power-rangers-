package agents

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/utils"
)

// LockedValue moved to internal/utils

type ExpandTuple = utils.Triple[*MctsNode, int, *environment.Game]

type BackpropagateTuple = utils.Triple[*MctsNode, int, *environment.Game]

type MCTSAgent struct {
	SimulationsPerMove  int
	SimulationsDone     *utils.LockedValue
	NbRoutines          int
	Root                *MctsNode
	ToBackpropagate     chan BackpropagateTuple
	ToExpandAndEvaluate chan ExpandTuple
	ResignThreshold     float64
}

// Constructor
func NewMCTSAgent(simulations_per_move int, nb_routines int, resign_threshold float64) *MCTSAgent {
	return &MCTSAgent{
		SimulationsPerMove:  simulations_per_move,
		NbRoutines:          nb_routines,
		ToBackpropagate:     make(chan BackpropagateTuple, nb_routines),
		ToExpandAndEvaluate: make(chan ExpandTuple, nb_routines),
		ResignThreshold:     resign_threshold,
	}
}

// Methods
func (agent *MCTSAgent) GetFinalAction(legal_actions []environment.Action) environment.Action {
	// get the argmax of self.root.N
	var best_action_index int = 0
	var max_visits int = agent.Root.N[0]
	var max_value float64 = agent.Root.Q[0]
	for action_index, visits := range agent.Root.N {
		if agent.Root.Q[action_index] > max_value {
			max_value = agent.Root.Q[action_index]
		}
		if visits > max_visits {
			max_visits = visits
			best_action_index = action_index
		}
	}
	// print in decimal format the max value for debugging purposes
	fmt.Printf("Best action index: %d, max value: %.4f, visits: %d\n", best_action_index, max_value, max_visits)

	if max_value <= agent.ResignThreshold {
		return environment.Resign{}
	}

	return legal_actions[best_action_index]
}

func (agent *MCTSAgent) SelectLeaf(node *MctsNode, game *environment.Game) ExpandTuple {
	var best_action_idx int = node.SelectBestChildIndex()
	if game.IsTerminal() {
		return utils.NewTriple(node, -1, game.DeepCopy())
	}
	if node.Children[best_action_idx] == nil {
		return utils.NewTriple(node, best_action_idx, game.DeepCopy())
	}
	game.PlayAction(game.LegalActions[best_action_idx])
	return agent.SelectLeaf(node.Children[best_action_idx], game)
}

func (agent *MCTSAgent) Expand(node *MctsNode, child_idx int, game *environment.Game) {
	game.PlayAction(game.LegalActions[child_idx])
	var child_node *MctsNode = NewMctsNode(game, node, child_idx)
	node.Children[child_idx] = child_node
}

func (agent *MCTSAgent) Evaluate(game *environment.Game) int {
	var current_player environment.Stone = game.Board.CurrentPlayer
	var both_players Agent = NewRandomAgent()
	for !game.IsTerminal() {
		game.PlayAction(both_players.SelectAction(game))
	}
	if game.GetWinner() == environment.Empty {
		return 0 // Draw
	} else if game.GetWinner() == current_player {
		return 1 // Win
	} else {
		return -1 // Loss
	}
}

func (agent *MCTSAgent) ExpandAndEvaluate(to_expand ExpandTuple, game *environment.Game) int {
	agent.Expand(to_expand.First, to_expand.Second, game)
	// Playing the action to reach the expanded child made so the opponent (expanded child) had this final value
	var value int = agent.Evaluate(game)
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

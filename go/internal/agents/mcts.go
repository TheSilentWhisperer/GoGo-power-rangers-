package agents

import (
	"sync"
	"sync/atomic"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/utils"
)

type MctsAgent struct {
	SimulationsPerMove int
	SimulationsDone    *utils.LockedValue
	NbRoutines         int
	Root               MctsNode
	ToBackpropagate    chan utils.Triple[MctsNode, int, *environment.Game]
	ResignThreshold    float64
	Expander           Expander
}

// Constructor
func NewMctsAgent(simulations_per_move int, nb_routines int, resign_threshold float64, expander Expander) *MctsAgent {
	return &MctsAgent{
		SimulationsPerMove: simulations_per_move,
		NbRoutines:         nb_routines,
		ToBackpropagate:    make(chan utils.Triple[MctsNode, int, *environment.Game], nb_routines),
		ResignThreshold:    resign_threshold,
		Expander:           expander,
	}
}

// Methods
func (agent *MctsAgent) GetFinalAction(legal_actions []environment.Action) environment.Action {
	// get the argmax of self.root.N
	var best_action_index int = 0
	var max_visits int = agent.Root.GetN()[0]
	var max_value float64 = agent.Root.GetQ()[0]
	for action_index, visits := range agent.Root.GetN() {
		if agent.Root.GetQ()[action_index] > max_value {
			max_value = agent.Root.GetQ()[action_index]
		}
		if visits > max_visits {
			max_visits = visits
			best_action_index = action_index
		}
	}

	if max_value <= agent.ResignThreshold {
		return environment.Resign{}
	}

	return legal_actions[best_action_index]
}

func (agent *MctsAgent) SelectLeaf(node MctsNode, game *environment.Game) utils.Triple[MctsNode, int, *environment.Game] {
	var best_action_idx int = node.SelectBestChildIndex()
	if game.IsTerminal() {
		return utils.NewTriple(node, -1, game.DeepCopy())
	}
	if node.GetChildren()[best_action_idx] == nil {
		return utils.NewTriple(node, best_action_idx, game.DeepCopy())
	}
	game.PlayAction(game.LegalActions[best_action_idx])
	return agent.SelectLeaf(node.GetChildren()[best_action_idx], game)
}

func (agent *MctsAgent) Backpropagate(to_backpropagate utils.Triple[MctsNode, int, *environment.Game]) {
	var node MctsNode = to_backpropagate.First
	var value int = to_backpropagate.Second
	if node.GetParent() != nil {
		node.GetParent().UpdateStats(value, node.GetIdx())
		agent.Backpropagate(utils.NewTriple(node.GetParent(), -value, (*environment.Game)(nil)))
	}
}

func (agent *MctsAgent) ExploreTree(wg *sync.WaitGroup, game *environment.Game) {

	defer wg.Done()
	for agent.SimulationsDone.Get() < agent.SimulationsPerMove {
		select {
		case to_backpropagate := <-agent.ToBackpropagate:
			agent.Backpropagate(to_backpropagate)
		case to_expand := <-agent.Expander.GetToExpand():
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
			if atomic.CompareAndSwapInt32((&to_expand.First.GetIsExpanded()[to_expand.Second]), 0, 1) {
				agent.SimulationsDone.Incr()                                // We are sure to expand a new node
				var value int = agent.Expander.ExpandAndEvaluate(to_expand) // Value of the game for the parent of the backpropagated node (expanded node)
				var expanded_child MctsNode = to_expand.First.GetChildren()[to_expand.Second]
				agent.ToBackpropagate <- utils.NewTriple(expanded_child, value, (*environment.Game)(nil))
			}

		default:
			var game_copy *environment.Game = game.DeepCopy()
			var to_expand utils.Triple[MctsNode, int, *environment.Game] = agent.SelectLeaf(agent.Root, game_copy)
			agent.Expander.GetToExpand() <- to_expand
		}
	}
}

func (agent *MctsAgent) SelectAction(game *environment.Game) environment.Action {

	// reset MCTS tree
	switch expander := agent.Expander.(type) {
	case *UctExpander:
		agent.Root = NewUctNode(game, nil, -1)
	case *PuctExpander:
		agent.Root = NewPuctNode(game, nil, -1, expander.Client)
	default:
		panic("Unknown expander type")
	}

	var wg sync.WaitGroup
	wg.Add(agent.NbRoutines)

	for nb_routines := 0; nb_routines < agent.NbRoutines; nb_routines++ {

		go agent.ExploreTree(&wg, game)
	}

	wg.Wait()

	var final_action environment.Action = agent.GetFinalAction(game.LegalActions)
	return final_action
}

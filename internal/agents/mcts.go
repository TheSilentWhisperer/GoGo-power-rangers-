package agents

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
)

type lockedValue struct {
	mutex sync.Mutex
	value int
}

func newLockedValue(value int) *lockedValue {
	return &lockedValue{
		value: value,
	}
}

func (lv *lockedValue) get() int {
	lv.mutex.Lock()
	defer lv.mutex.Unlock()
	return lv.value
}

func (lv *lockedValue) incr() {
	lv.mutex.Lock()
	defer lv.mutex.Unlock()
	lv.value += 1
}

type expandTuple struct {
	node      *mctsNode
	child_idx int
	game      *environment.Game
}

type backpropagateTuple struct {
	node  *mctsNode
	value int
	game  *environment.Game
}

type mctsAgent struct {
	simulationsPerMove  int
	simulationsDone     *lockedValue
	nbRoutines          int
	root                *mctsNode
	toBackpropagate     chan backpropagateTuple
	toExpandAndEvaluate chan expandTuple
	resignThreshold     float64
}

// Constructor
func NewMCTSAgent(simulationsPerMove int, nbRoutines int, resignThreshold float64) *mctsAgent {
	return &mctsAgent{
		simulationsPerMove:  simulationsPerMove,
		nbRoutines:          nbRoutines,
		toBackpropagate:     make(chan backpropagateTuple, nbRoutines),
		toExpandAndEvaluate: make(chan expandTuple, nbRoutines),
		resignThreshold:     resignThreshold,
	}
}

// Methods
func (agent *mctsAgent) getFinalAction(legalActions []environment.Action) environment.Action {
	// get the argmax of self.root.N
	var bestActionIndex int = 0
	var maxVisits int = agent.root.N[0]
	var maxValue float64 = agent.root.Q[0]
	for actionIndex, visits := range agent.root.N {
		if agent.root.Q[actionIndex] > maxValue {
			maxValue = agent.root.Q[actionIndex]
		}
		if visits > maxVisits {
			maxVisits = visits
			bestActionIndex = actionIndex
		}
	}
	// print in decimal format the max value for debugging purposes
	fmt.Printf("Best action index: %d, max value: %.4f, visits: %d\n", bestActionIndex, maxValue, maxVisits)

	if maxValue <= agent.resignThreshold {
		return environment.Resign{}
	}

	return legalActions[bestActionIndex]
}

func (agent *mctsAgent) selectLeaf(node *mctsNode, game *environment.Game) expandTuple {
	var best_action_idx int = node.selectBestChildIndex()
	if game.IsTerminal() {
		return expandTuple{node: node, child_idx: -1, game: game.DeepCopy()}
	}
	if node.children[best_action_idx] == nil {
		return expandTuple{node: node, child_idx: best_action_idx, game: game.DeepCopy()}
	}
	game.PlayAction(game.LegalActions[best_action_idx])
	return agent.selectLeaf(node.children[best_action_idx], game)
}

func (agent *mctsAgent) expand(node *mctsNode, child_idx int, game *environment.Game) {
	game.PlayAction(game.LegalActions[child_idx])
	var child_node *mctsNode = newMctsNode(game, node, child_idx)
	node.children[child_idx] = child_node
}

func (agent *mctsAgent) evaluate(game *environment.Game) int {
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

func (agent *mctsAgent) expandAndEvaluate(toExpand expandTuple, game *environment.Game) int {
	agent.expand(toExpand.node, toExpand.child_idx, game)
	// Playing the action to reach the expanded child made so the opponent (expanded child) had this final value
	var value int = agent.evaluate(game)
	// So the value for the parent node is the negation of this value
	return -value
}

func (agent *mctsAgent) backpropagate(toBackpropagate backpropagateTuple) {
	var node *mctsNode = toBackpropagate.node
	var value int = toBackpropagate.value
	if node.parent != nil {
		node.parent.updateStats(value, node.idx)
		agent.backpropagate(backpropagateTuple{node: node.parent, value: -value})
	}
}

func (agent *mctsAgent) exploreTree(wg *sync.WaitGroup, game *environment.Game) {

	defer wg.Done()
	for agent.simulationsDone.get() < agent.simulationsPerMove {
		select {
		case toBackpropagate := <-agent.toBackpropagate:
			agent.backpropagate(toBackpropagate)
		case toExpand := <-agent.toExpandAndEvaluate:
			if toExpand.child_idx == -1 {
				// Terminal node reached, no expansion
				agent.simulationsDone.incr() // We are sure to expand a new node
				var value int                // Value of the game for the parent of the backpropagated node (terminal node)
				var winner environment.Stone = toExpand.game.GetWinner()
				switch winner {
				case environment.Empty:
					value = 0 // Draw
				case toExpand.game.Board.CurrentPlayer:
					value = -1 // Loss for the parent
				default:
					value = 1 // Win for the parent
				}

				agent.toBackpropagate <- backpropagateTuple{node: toExpand.node, value: value}
				continue
			}
			// atomic check to avoid expanding the same node multiple times
			if atomic.CompareAndSwapInt32((&toExpand.node.is_expanded[toExpand.child_idx]), 0, 1) {
				agent.simulationsDone.incr()                                     // We are sure to expand a new node
				var value int = agent.expandAndEvaluate(toExpand, toExpand.game) // Value of the game for the parent of the backpropagated node (expanded node)
				var expanded_child *mctsNode = toExpand.node.children[toExpand.child_idx]
				agent.toBackpropagate <- backpropagateTuple{node: expanded_child, value: value}
			}

		default:
			var game_copy *environment.Game = game.DeepCopy()
			var toExpand expandTuple = agent.selectLeaf(agent.root, game_copy)
			agent.toExpandAndEvaluate <- toExpand
		}
	}
}

func (agent *mctsAgent) SelectAction(game *environment.Game) environment.Action {

	// reset MCTS tree
	agent.root = newMctsNode(game, nil, -1)
	agent.simulationsDone = newLockedValue(0)

	var wg sync.WaitGroup
	wg.Add(agent.nbRoutines)

	for nb_routines := 0; nb_routines < agent.nbRoutines; nb_routines++ {

		go agent.exploreTree(&wg, game)
	}

	wg.Wait()

	var finalAction environment.Action = agent.getFinalAction(game.LegalActions)
	return finalAction
}

package agents

import (
	"context"
	"fmt"
	"sync"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/utils"
	"google.golang.org/protobuf/types/known/emptypb"
)

type MctsAgent struct {
	SimulationsPerMove   int
	SimulationsDone      *utils.LockedValue
	NbRoutines           int
	MaxParallelSearch    int
	CurrentNbSearches    *utils.LockedValue
	Root                 MctsNode
	ExpansionQueue       *utils.LockedQueue[utils.Triple[MctsNode, int, *environment.Game]]     // (Node, ActionIdx, GameState)
	BackpropagationQueue *utils.LockedQueue[utils.Triple[MctsNode, float64, *environment.Game]] // (Node, Value, GameState)
	ResignThreshold      float64
	Evaluator            Evaluator
}

// Constructor
func NewMctsAgent(simulations_per_move int, nb_routines int, max_parallel_search int, resign_threshold float64, evaluator Evaluator) *MctsAgent {
	return &MctsAgent{
		SimulationsPerMove:   simulations_per_move,
		SimulationsDone:      utils.NewLockedValue(0),
		NbRoutines:           nb_routines,
		MaxParallelSearch:    max_parallel_search,
		CurrentNbSearches:    utils.NewLockedValue(0),
		ExpansionQueue:       utils.NewLockedQueue[utils.Triple[MctsNode, int, *environment.Game]](max_parallel_search),
		BackpropagationQueue: utils.NewLockedQueue[utils.Triple[MctsNode, float64, *environment.Game]](max_parallel_search),
		ResignThreshold:      resign_threshold,
		Evaluator:            evaluator,
	}
}

// Methods
func (agent *MctsAgent) GetFinalAction(legal_actions []environment.Action) environment.Action {
	// get the argmax of self.root.N
	var best_action_index int = 0
	var max_visits int = agent.Root.GetN()[0]
	for action_index, visits := range agent.Root.GetN() {
		if visits > max_visits {
			max_visits = visits
			best_action_index = action_index
		}
		fmt.Printf("Action index: %d, Visits: %d, Value: %f\n", action_index, visits, agent.Root.GetQ()[action_index])
	}

	if agent.Root.GetQ()[best_action_index] <= agent.ResignThreshold {
		return environment.Resign{}
	}

	return legal_actions[best_action_index]
}

func (agent *MctsAgent) SelectLeaf(node MctsNode, game *environment.Game) utils.Triple[MctsNode, int, *environment.Game] {
	if game.IsTerminal() {
		panic("Do not call SelectLeaf on a terminal node")
	}

	var best_action_idx int = node.SelectBestChildIndex()

	if node.GetChildren()[best_action_idx] == nil || node.GetIsTerminal(best_action_idx) || node.GetIsEvaluating(best_action_idx) {
		return utils.NewTriple(node, best_action_idx, game.DeepCopy())
	}
	game.PlayAction(game.LegalActions[best_action_idx])
	return agent.SelectLeaf(node.GetChildren()[best_action_idx], game)
}

func (agent *MctsAgent) Backpropagate(to_backpropagate utils.Triple[MctsNode, float64, *environment.Game]) {
	var node MctsNode = to_backpropagate.First
	var value float64 = to_backpropagate.Second
	if node.GetParent() != nil {
		node.GetParent().UpdateStats(-value, node.GetIdx()) // The value is negated because the value for the parent is the opposite of the value for the child
		agent.Backpropagate(utils.NewTriple(node.GetParent(), -value, (*environment.Game)(nil)))
	}
}

func (agent *MctsAgent) ExploreTree(wg *sync.WaitGroup, game *environment.Game) {

	defer wg.Done()
	for agent.SimulationsDone.Get() < agent.SimulationsPerMove {

		defer agent.CurrentNbSearches.Decr()

		// Backpropagate
		to_backpropagate, ok := agent.BackpropagationQueue.Dequeue()
		if ok {
			agent.Backpropagate(to_backpropagate)
			agent.SimulationsDone.Incr()
			agent.CurrentNbSearches.Decr()
			continue
		}

		// Evaluate
		to_evaluate, ok := agent.Evaluator.GetEvaluationQueue().Dequeue()
		if ok {
			var evaluation utils.Triple[MctsNode, float64, *environment.Game] = agent.Evaluator.Evaluate(to_evaluate)
			agent.BackpropagationQueue.Enqueue(evaluation)
			continue
		}

		// Expand
		to_expand, ok := agent.ExpansionQueue.Dequeue()
		if ok {
			var node MctsNode = to_expand.First
			var child_idx int = to_expand.Second
			var game_state *environment.Game = to_expand.Third

			node.ExpandChild(child_idx, game_state)
			agent.Evaluator.GetEvaluationQueue().Enqueue(to_expand)
			continue
		}

		// Select
		if !agent.CurrentNbSearches.CompareAndIncrement(agent.MaxParallelSearch) {
			continue
		}

		var selection utils.Triple[MctsNode, int, *environment.Game] = agent.SelectLeaf(agent.Root, game.DeepCopy())
		var selected_node MctsNode = selection.First
		var selected_action_idx int = selection.Second
		var game *environment.Game = selection.Third
		var action environment.Action = game.LegalActions[selected_action_idx]
		game.PlayAction(action)
		if selected_node.GetChildren()[selected_action_idx] == nil {
			// The node is not expanded yet, we can expand it
			agent.ExpansionQueue.Enqueue(selection)
		} else {
			// The node is either terminal or currently being evaluated, anyway we add it for evaluation
			agent.Evaluator.GetEvaluationQueue().Enqueue(selection)
		}
	}
	println("Finished exploring the tree")
}

func (agent *MctsAgent) SelectAction(game *environment.Game) environment.Action {

	// reset MCTS tree and initialize priors if needed
	switch evaluator := agent.Evaluator.(type) {
	case *UctEvaluator:
		agent.Root = NewUctNode(game, nil, -1)
	case *PuctEvaluator:
		agent.Root = NewPuctNode(game, nil, -1)
		evaluator.Reset()
		evaluator.Client.ResetServer(context.Background(), &emptypb.Empty{})
	default:
		panic("Unknown evaluator type")
	}
	switch root := agent.Root.(type) {
	case *PuctNode:
		root.SetPriors(make([]float64, len(game.LegalActions))) // We will set the priors later when we have the neural network evaluation
	}

	// reset simulations done (preserve the same LockedValue instance so Evaluator shares it)
	agent.SimulationsDone.Set(0)

	var wg sync.WaitGroup
	wg.Add(agent.NbRoutines)

	for nb_routines := 0; nb_routines < agent.NbRoutines; nb_routines++ {
		go agent.ExploreTree(&wg, game)
	}

	wg.Wait()

	var final_action environment.Action = agent.GetFinalAction(game.LegalActions)
	return final_action
}

package agents

import (
	"context"
	"fmt"
	"math"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/gen/proto/remote_trainer"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/utils"
	"github.com/jmcvetta/randutil"
	"gonum.org/v1/gonum/stat/distuv"
)

type MctsAgent struct {
	SimulationsPerMove   int
	Epsilon              float64
	Alpha                float64
	SimulationsDone      *utils.LockedValue
	NbRoutines           int
	MaxParallelSearch    int
	CurrentNbSearches    *utils.LockedValue
	Root                 MctsNode
	ExpansionQueue       *utils.LockedQueue[utils.Triple[MctsNode, int, *environment.Game]] // (Node, ActionIdx, GameState)
	BackpropagationQueue *utils.LockedQueue[utils.Pair[MctsNode, float64]]                  // (Node, Value, GameState)
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
		BackpropagationQueue: utils.NewLockedQueue[utils.Pair[MctsNode, float64]](max_parallel_search),
		ResignThreshold:      resign_threshold,
		Evaluator:            evaluator,
	}
}

// Methods
func (agent *MctsAgent) GetFinalAction(game *environment.Game, legal_actions []environment.Action) environment.Action {
	var tau float64 = (float64(len(legal_actions)-1) / 81) // Scale temperature by board size to keep it consistent across different board sizes
	tau = math.Pow(tau, 2)
	var max_N int = 0
	for i, n := range agent.Root.GetN() {
		if i > 0 && n > max_N {
			max_N = n
		}
	}

	// Randomly sample with a temperature of 1, which means we sample proportionally to the visit count of each action
	choices := make([]randutil.Choice, len(legal_actions))
	for i := 0; i < len(legal_actions); i++ {
		var weight = math.Log(float64(agent.Root.GetN()[i])) - math.Log(float64(max_N)) // Subtract max_N to prevent overflow when exponentiating
		weight = math.Exp(weight / tau)
		choices[i] = randutil.Choice{
			Item:   i,
			Weight: int(weight), // Temperature of 3 means we sample proportionally to visit count cubed
		}
	}

	choice, err := randutil.WeightedChoice(choices)
	if err != nil {
		panic(err)
	}
	var best_action_index int = choice.Item.(int)

	// Only resign if ALL possible moves are valued below threshold
	var maxQ float64 = math.Inf(-1)
	for i := 1; i < len(legal_actions); i++ {
		if agent.Root.GetQ()[i] > maxQ {
			maxQ = agent.Root.GetQ()[i]
		}
	}

	if maxQ <= agent.ResignThreshold {
		fmt.Printf("Player: %v, Resigning (best Q across all moves: %f, threshold: %f)\n", game.Board.CurrentPlayer, maxQ, agent.ResignThreshold)
		return environment.Resign{}
	}

	var best_action environment.Action = legal_actions[best_action_index]

	fmt.Printf("Player: %v, Selected action: %v, N: %d, Q: %f, P: %f, MaxQ: %f\n", game.Board.CurrentPlayer, best_action, agent.Root.GetN()[best_action_index], agent.Root.GetQ()[best_action_index], agent.Root.(*PuctNode).P[best_action_index], maxQ)

	return best_action
}

func (agent *MctsAgent) SelectLeaf(node MctsNode, game *environment.Game) utils.Triple[MctsNode, int, *environment.Game] {
	if game.IsTerminal() {
		panic("Do not call SelectLeaf on a terminal node")
	}

	var best_action_idx int = node.SelectBestChildIndex()

	if node.GetChildren()[best_action_idx] == nil || node.GetIsTerminal(best_action_idx) || node.GetIsEvaluating(best_action_idx) {
		return utils.NewTriple(node, best_action_idx, game.DeepCopy())
	}

	// Don't traverse deeper into nodes that don't have priors yet (async evaluation in progress)
	child := node.GetChildren()[best_action_idx]
	if !child.GetHasPriors() {
		return utils.NewTriple(node, best_action_idx, game.DeepCopy())
	}

	game.PlayAction(game.LegalActions[best_action_idx])
	return agent.SelectLeaf(child, game)
}

func (agent *MctsAgent) Backpropagate(to_backpropagate utils.Pair[MctsNode, float64]) {
	var node MctsNode = to_backpropagate.First
	var value float64 = to_backpropagate.Second
	if node.GetParent() != nil {
		node.GetParent().UpdateStats(-value, node.GetIdx()) // The value is negated because the value for the parent is the opposite of the value for the child
		agent.Backpropagate(utils.NewPair(node.GetParent(), -value))
	}
}

func (agent *MctsAgent) ExploreTree(wg *sync.WaitGroup, game *environment.Game) {

	defer wg.Done()
	for agent.SimulationsDone.Get() < agent.SimulationsPerMove {
		// Check if game is terminal before continuing
		if game.IsTerminal() {
			break
		}

		// Backpropagate

		to_backpropagate, ok := agent.BackpropagationQueue.Dequeue()
		if ok {
			agent.Backpropagate(to_backpropagate)
			agent.CurrentNbSearches.Decr()
			agent.SimulationsDone.Incr()
			continue
		}

		switch agent.Evaluator.(type) {
		case *PuctEvaluator:
			var evaluation_pair utils.Pair[utils.Triple[MctsNode, float64, *environment.Game], bool] = agent.Evaluator.(*PuctEvaluator).RetrieveEvaluation()
			var evaluation utils.Triple[MctsNode, float64, *environment.Game] = evaluation_pair.First
			var ok bool = evaluation_pair.Second
			if ok {
				var game *environment.Game = evaluation.Third
				if game.IsTerminal() {
					var evaluated_node MctsNode = evaluation.First
					if evaluated_node == nil {
						panic("Evaluated node should be terminal but it is nil")
					}

					var value float64
					switch game.GetWinner() {
					case game.Board.CurrentPlayer:
						value = 1.0
					case game.Board.CurrentPlayer.Opponent():
						value = -1.0
					default:
						panic("Terminal game should have a winner")
					}

					agent.BackpropagationQueue.Enqueue(utils.NewPair(evaluated_node, value))
					continue
				}

				to_backpropagate := utils.NewPair(evaluation.First, evaluation.Second)
				agent.BackpropagationQueue.Enqueue(to_backpropagate)
				continue
			}
		}
		// Submit evaluation
		to_evaluate, ok := agent.Evaluator.GetEvaluationQueue().Dequeue()
		if ok {
			switch agent.Evaluator.(type) {
			case *PuctEvaluator:
				// Batch all evaluations - no early flushing needed with synchronous training
				// Only flush at the very end to process remaining batch
				remaining := agent.SimulationsPerMove - agent.SimulationsDone.Get()
				should_flush := remaining == 0
				agent.Evaluator.(*PuctEvaluator).SubmitEvaluation(to_evaluate, should_flush)
				continue
			case *UctEvaluator:
				// For a UCT evaluator, we can directly evaluate the node
				var evaluation utils.Triple[MctsNode, float64, *environment.Game] = agent.Evaluator.(*UctEvaluator).Evaluate(to_evaluate)
				to_backpropagate := utils.NewPair(evaluation.First, evaluation.Second)
				agent.BackpropagationQueue.Enqueue(to_backpropagate)
				continue
			default:
				panic("Unknown evaluator type")
			}
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

		// Select: acquire a search slot which will be released when the corresponding backpropagation completes
		// if !agent.CurrentNbSearches.CompareAndIncrement(agent.MaxParallelSearch) {
		// 	continue
		// }

		if agent.Root.GetTotalN() == agent.SimulationsPerMove {
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

}

func (agent *MctsAgent) InitRootNode(game *environment.Game) {

	switch evaluator := agent.Evaluator.(type) {
	case *UctEvaluator:
		agent.Root = NewUctNode(game, nil, -1)
	case *PuctEvaluator:
		//Evaluate game to get priors for the root node
		var request *remote_trainer.EvaluationRequest = game.EvaluationRequest(0, true) // We can use any request_id here since we know the root evaluation will be processed before any other evaluation, and we set flush to true to bypass the batch queue and get the evaluation as soon as possible
		response, err := agent.Evaluator.(*PuctEvaluator).Client.EvaluatePosition(context.Background(), request)
		if err != nil {
			panic(err)
		}
		// Map full priors vector (Resign, Pass, board...) to the root node's local action ordering
		fullPriors := response.GetPriors()
		mapped := make([]float64, len(game.LegalActions))
		expected := 2 + game.Board.Height*game.Board.Width
		if len(fullPriors) < expected {
			panic("Received priors of unexpected length for root")
		}
		for i, action := range game.LegalActions {
			switch a := action.(type) {
			case environment.Resign:
				mapped[i] = fullPriors[0]
			case environment.Pass:
				mapped[i] = fullPriors[1]
			case environment.PutStone:
				put := a
				idx := 2 + put.I*game.Board.Width + put.J
				mapped[i] = fullPriors[idx]
			default:
				panic("Unknown action type when mapping root priors")
			}
		}

		// Add Dirichlet noise to the root priors to encourage exploration.
		alpha := agent.Alpha / float64(len(mapped)) // Scale alpha by the number of actions to keep the noise level consistent across different board sizes
		eps := agent.Epsilon

		k := len(mapped)
		src := mrand.New(mrand.NewSource(time.Now().UnixNano()))
		g := distuv.Gamma{Alpha: alpha, Beta: 1, Src: src}
		dir := make([]float64, k)
		var sum float64
		for i := 0; i < k; i++ {
			v := g.Rand()
			dir[i] = v
			sum += v
		}
		if sum == 0 {
			for i := range dir {
				dir[i] = 1.0 / float64(k)
			}
		} else {
			for i := range dir {
				dir[i] /= sum
			}
		}

		noisy := make([]float64, k)
		for i := 0; i < k; i++ {
			noisy[i] = max((1-eps)*mapped[i]+eps*dir[i], 0)
		}

		agent.Root = NewPuctNode(game, nil, -1, noisy)
		evaluator.Reset()
	default:
		panic("Unknown evaluator type")
	}
}

func (agent *MctsAgent) SelectAction(game *environment.Game) environment.Action {

	agent.InitRootNode(game)

	// reset simulations done (preserve the same LockedValue instance so Evaluator shares it)
	agent.SimulationsDone.Set(0)

	// Clear evaluation maps to prevent stale requests from previous move
	if puctEval, ok := agent.Evaluator.(*PuctEvaluator); ok {
		puctEval.Reset()
	}

	var wg sync.WaitGroup
	wg.Add(agent.NbRoutines)

	for nb_routines := 0; nb_routines < agent.NbRoutines; nb_routines++ {
		go agent.ExploreTree(&wg, game)
	}

	// fmt.Printf("[MCTS] Waiting for all %d workers to complete...\n", agent.NbRoutines)
	wg.Wait()
	// fmt.Printf("[MCTS] All workers completed\n")

	agent.CurrentNbSearches.Set(0)

	var final_action environment.Action = agent.GetFinalAction(game, game.LegalActions)
	return final_action
}

package agents

import (
	"context"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/gen/proto/remote_trainer"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/utils"
	"google.golang.org/protobuf/types/known/emptypb"
)

// LockedValue moved to internal/utils

type PuctEvaluator struct {
	*UctEvaluator
	request_id_counter  *utils.LockedValue
	SimulationsDone     *utils.LockedValue
	SimulationsPerMove  int
	RemoteEvaluationMap *utils.LockedMap[int, MctsNode] // request_id -> evaluated_node
	Client              remote_trainer.PositionEvaluatorClient
}

func NewPuctEvaluator(max_parallel_searches int, client remote_trainer.PositionEvaluatorClient, simulations_done *utils.LockedValue, simulations_per_move int) *PuctEvaluator {
	return &PuctEvaluator{
		UctEvaluator:        NewUctEvaluator(max_parallel_searches),
		request_id_counter:  utils.NewLockedValue(0),
		SimulationsDone:     simulations_done,
		SimulationsPerMove:  simulations_per_move,
		RemoteEvaluationMap: utils.NewLockedMap[int, MctsNode](),
		Client:              client,
	}
}

func (evaluator *PuctEvaluator) Reset() {
	evaluator.request_id_counter.Set(0)
	evaluator.RemoteEvaluationMap = utils.NewLockedMap[int, MctsNode]()
}

func (evaluator *PuctEvaluator) GetEvaluationQueue() *utils.LockedQueue[utils.Triple[MctsNode, int, *environment.Game]] {
	return evaluator.EvaluationQueue
}

func (evaluator *PuctEvaluator) Evaluate(to_evaluate utils.Triple[MctsNode, int, *environment.Game]) utils.Triple[MctsNode, float64, *environment.Game] {
	//Does not necessarily retrieve the evaluation from the same position, but it can also retrieve the evaluation from a different call
	var node MctsNode = to_evaluate.First
	var action_idx int = to_evaluate.Second
	var game *environment.Game = to_evaluate.Third

	if node.GetIsTerminal(action_idx) {
		var evaluated_node MctsNode = node.GetChildren()[action_idx]
		var value float64
		var current_player environment.Stone = game.Board.CurrentPlayer
		if game.GetWinner() == environment.Empty {
			value = 0
		} else if game.GetWinner() == current_player {
			value = 1
		} else {
			value = -1
		}
		node.SetIsEvaluating(action_idx, false)
		return utils.NewTriple(evaluated_node, value, game)
	}

	var evaluated_node MctsNode = node.GetChildren()[action_idx]

	// Send the evaluation request to the remote trainer
	var request_id int = evaluator.request_id_counter.GetAndIncrement()
	evaluator.RemoteEvaluationMap.Set(request_id, evaluated_node)
	var message *remote_trainer.EvaluationRequest = game.Message(request_id)
	evaluator.Client.EvaluatePosition(context.Background(), message)

	//Wait for any response to be available in the RemoteEvaluationMap, and return the corresponding node
	for evaluator.SimulationsDone.Get() < evaluator.SimulationsPerMove {
		// println(evaluator.SimulationsDone.Get(), "/", evaluator.SimulationsPerMove, " simulations done. Waiting for evaluation response...")
		//call with empty proto argument
		response, err := evaluator.Client.RetrieveEvaluation(context.Background(), &emptypb.Empty{})
		if err != nil {
			panic(err)
		}
		// Check if reponse is not empty
		if response.GetData() != nil {
			var response_id int = int(response.Data.GetRequestId())
			response_node, ok := evaluator.RemoteEvaluationMap.Get(response_id)
			if !ok {
				panic("Received a response for an unknown request_id")
			}
			evaluator.RemoteEvaluationMap.Delete(response_id)
			var priors []float64 = response.Data.GetPriors()
			var value float64 = response.Data.GetValue()
			switch response_node := response_node.(type) {
			case *PuctNode:
				response_node.SetPriors(priors)
			default:
				panic("Expected a PuctNode")
			}
			response_node.GetParent().SetIsEvaluating(response_node.GetIdx(), false)
			return utils.NewTriple(response_node, value, game)
		}
	}
	return utils.NewTriple(evaluated_node, 0.0, game) // Return a dummy evaluation if we reached the simulations per move limit (should not happen often)
}

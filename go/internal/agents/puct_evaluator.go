package agents

import (
	"context"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/gen/proto/remote_trainer"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/utils"
)

// LockedValue moved to internal/utils

type PuctEvaluator struct {
	*UctEvaluator
	request_id_counter  *utils.LockedValue
	SimulationsDone     *utils.LockedValue
	SimulationsPerMove  int
	RemoteEvaluationMap *utils.LockedMap[int64, utils.Pair[MctsNode, *environment.Game]] // request_id -> (evaluated_node, game)
	Client              remote_trainer.PositionEvaluatorClient
}

func NewPuctEvaluator(max_parallel_searches int, client remote_trainer.PositionEvaluatorClient, simulations_done *utils.LockedValue, simulations_per_move int) *PuctEvaluator {
	return &PuctEvaluator{
		UctEvaluator:        NewUctEvaluator(max_parallel_searches),
		request_id_counter:  utils.NewLockedValue(0),
		SimulationsDone:     simulations_done,
		SimulationsPerMove:  simulations_per_move,
		RemoteEvaluationMap: utils.NewLockedMap[int64, utils.Pair[MctsNode, *environment.Game]](),
		Client:              client,
	}
}

func (evaluator *PuctEvaluator) Reset() {
	evaluator.request_id_counter.Set(0)
	evaluator.RemoteEvaluationMap = utils.NewLockedMap[int64, utils.Pair[MctsNode, *environment.Game]]()
}

func (evaluator *PuctEvaluator) GetEvaluationQueue() *utils.LockedQueue[utils.Triple[MctsNode, int, *environment.Game]] {
	return evaluator.EvaluationQueue
}

func (evaluator *PuctEvaluator) RetrieveEvaluation() utils.Pair[utils.Triple[MctsNode, float64, *environment.Game], bool] {
	// Request responses without timeout to allow indefinite wait
	request := &remote_trainer.RetrieveRequest{}
	response, err := evaluator.Client.RetrieveEvaluation(context.Background(), request)
	if err != nil {
		// Timeout is normal - just continue, will try again next iteration
		return utils.NewPair(utils.NewTriple[MctsNode, float64, *environment.Game](nil, 0, nil), false)
	}
	// Check if response is not empty
	if response.GetData() == nil {
		return utils.NewPair(utils.NewTriple[MctsNode, float64, *environment.Game](nil, 0, nil), false)
	}

	// Map the response back to the corresponding node and game state
	var response_id int64 = response.Data.GetRequestId()
	response_pair, ok := evaluator.RemoteEvaluationMap.Get(response_id)
	if !ok {
		// Response for unknown request_id - skip
		return utils.NewPair(utils.NewTriple[MctsNode, float64, *environment.Game](nil, 0, nil), false)
	}

	var response_node MctsNode = response_pair.First
	var game *environment.Game = response_pair.Second
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
	return utils.NewPair(utils.NewTriple(response_node, value, game), true)
}

func (evaluator *PuctEvaluator) SubmitEvaluation(to_evaluate utils.Triple[MctsNode, int, *environment.Game], flush bool) {
	var node MctsNode = to_evaluate.First
	var action_idx int = to_evaluate.Second
	var game *environment.Game = to_evaluate.Third

	var evaluated_node MctsNode = node.GetChildren()[action_idx]

	// Send the evaluation request to the remote trainer
	local_counter := evaluator.request_id_counter.GetAndIncrement()
	// Simple 64-bit counter: 18.4 quintillion requests before wrap
	var request_id int64 = int64(local_counter)
	evaluator.RemoteEvaluationMap.Set(request_id, utils.NewPair(evaluated_node, game))
	var request *remote_trainer.EvaluationRequest = game.EvaluationRequest(request_id, flush)

	// Submit asynchronously without timeout to allow indefinite wait
	go func() {
		_, err := evaluator.Client.SubmitEvaluation(context.Background(), request)
		if err != nil {
			// Submission errors are handled by orphan cleanup - timeout will clean up stale requests
			_ = err
		}
	}()
}

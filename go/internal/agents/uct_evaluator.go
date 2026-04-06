package agents

import (
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/utils"
)

// LockedValue moved to internal/utils

type UctEvaluator struct {
	EvaluationQueue *utils.LockedQueue[utils.Triple[MctsNode, int, *environment.Game]] // (Node, ActionIdx, GameState)
}

func NewUctEvaluator(max_parallel_searches int) *UctEvaluator {
	return &UctEvaluator{
		EvaluationQueue: utils.NewLockedQueue[utils.Triple[MctsNode, int, *environment.Game]](max_parallel_searches),
	}
}

func (evaluator *UctEvaluator) GetEvaluationQueue() *utils.LockedQueue[utils.Triple[MctsNode, int, *environment.Game]] {
	return evaluator.EvaluationQueue
}

func (evaluator *UctEvaluator) Evaluate(to_evaluate utils.Triple[MctsNode, int, *environment.Game]) utils.Triple[MctsNode, float64, *environment.Game] {

	var node MctsNode = to_evaluate.First
	var action_idx int = to_evaluate.Second
	var game *environment.Game = to_evaluate.Third

	defer node.SetIsEvaluating(action_idx, false)

	var evaluated_node MctsNode = node.GetChildren()[action_idx]

	var current_player environment.Stone = game.Board.CurrentPlayer

	var both_players Agent = NewRandomAgent()
	for !game.IsTerminal() {
		game.PlayAction(both_players.SelectAction(game))
	}
	// Use winner only (±1) — do not care about score difference
	var winner environment.Stone = game.GetWinner()
	var value float64
	switch winner {
	case current_player:
		value = 1.0
	case current_player.Opponent():
		value = -1.0
	default:
		value = 0.0
	}
	return utils.NewTriple(evaluated_node, value, game)
}

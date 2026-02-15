package agents

import (
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/utils"
)

// LockedValue moved to internal/utils

type UctExpander struct {
	ToExpand chan utils.Triple[MctsNode, int, *environment.Game]
}

func NewUctExpander(nb_routines int) *UctExpander {
	return &UctExpander{
		ToExpand: make(chan utils.Triple[MctsNode, int, *environment.Game], nb_routines),
	}
}

func (expander *UctExpander) GetToExpand() chan utils.Triple[MctsNode, int, *environment.Game] {
	return expander.ToExpand
}

func (expander *UctExpander) Expand(to_expand utils.Triple[MctsNode, int, *environment.Game]) {
	var node MctsNode = to_expand.First
	var child_idx int = to_expand.Second
	var game *environment.Game = to_expand.Third
	game.PlayAction(game.LegalActions[child_idx])
	var child_node MctsNode = NewUctNode(game, node, child_idx)
	node.GetChildren()[child_idx] = child_node
}

func (expander *UctExpander) Evaluate(game *environment.Game) int {
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

func (expander *UctExpander) ExpandAndEvaluate(to_expand utils.Triple[MctsNode, int, *environment.Game]) int {
	expander.Expand(to_expand)
	// Playing the action to reach the expanded child made so the opponent (expanded child) had this final value
	var value int = expander.Evaluate(to_expand.Third)
	// So the value for the parent node is the negation of this value
	return -value
}

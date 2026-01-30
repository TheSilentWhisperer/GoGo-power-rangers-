package main

import (
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/agents"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
)

func emulate_game(game *environment.Game, black_player agents.Agent, white_player agents.Agent) {
	for !game.IsTerminal() {
		var current_agent agents.Agent
		if game.Board.CurrentPlayer == environment.Black {
			current_agent = black_player
		} else {
			current_agent = white_player
		}
		var action environment.Action = current_agent.SelectAction(game)
		game.PlayAction(action)
		// game.DisplayBoard()
		// _, _ = fmt.Scanln() // Wait for user input to proceed to the next turn
	}
}

func main() {
	for i := 0; i < 10000; i++ {
		game := environment.NewGame(
			9, // height
			9, // width
		)
		black_agent, white_agent := agents.NewRandomAgent(), agents.NewRandomAgent()
		emulate_game(game, black_agent, white_agent)
	}
}

package main

import (
	"fmt"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/agents"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
)

func emulate_game(go_ *environment.Go_, black_player agents.Agent, white_player agents.Agent) {
	for !go_.IsTerminal() {
		var current_agent agents.Agent
		if go_.Board.CurrentPlayer == environment.Black {
			current_agent = black_player
		} else {
			current_agent = white_player
		}
		var action environment.Action = current_agent.SelectAction(go_)
		go_.PlayAction(action)
		go_.DisplayBoard()
		_, _ = fmt.Scanln() // Wait for user input to proceed to the next turn
	}
}

func main() {
	go_ := environment.NewGoGame(
		9, // height
		9, // width
	)
	black_agent, white_agent := agents.NewRandomAgent(), agents.NewRandomAgent()
	emulate_game(go_, black_agent, white_agent)
}

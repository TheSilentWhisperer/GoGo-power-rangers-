package ui

import (
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/agents"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
	"github.com/hajimehoshi/ebiten/v2"
)

func (app *App) Update() error {

	// Start by updating key states
	for _, key_state := range app.KeyStates {
		key_state.Get().Update()
	}

	if app.Game.Get().IsTerminal() {
		return nil // Game over, no more updates needed
	}

	if app.KeyStates[ebiten.KeySpace].Get().JustPressed() {
		app.IsPaused.Set(!app.IsPaused.Get())
	}

	var current_agent agents.Agent
	if app.Game.Get().Board.CurrentPlayer == environment.Black {
		current_agent = app.BlackAgent
	} else {
		current_agent = app.WhiteAgent
	}

	select {
	case app.MoveSearchInitiated <- true:
		// We are the first to initiate the move search, start the goroutine
		// to compute the move in the background
		go func() {
			defer func() { <-app.MoveSearchInitiated }() // Ensure that we reset the channel when done

			app.IsThinking.Set(true)
			var game_copy *environment.Game = app.Game.Get().DeepCopy()
			var action environment.Action = current_agent.SelectAction(game_copy)
			var current_player string
			switch game_copy.Board.CurrentPlayer {
			case environment.Black:
				current_player = "Black"
			case environment.White:
				current_player = "White"
			default:
				current_player = "Unknown"
			}
			println("Current player:", current_player, "Selected action:", action.String())
			app.IsThinking.Set(false)

			for app.IsPaused.Get() {
				// Wait for the space key to be pressed to play the move, this allows the user to see the move before it is played
			}
			game_copy.PlayAction(action)
			app.Game.Set(game_copy)
		}()
	default:
		// Another goroutine has already initiated the move search, do nothing
	}

	return nil
}

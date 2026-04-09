package ui

import (
	"context"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/agents"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

func (app *App) Update() error {

	// Start by updating key states
	for _, key_state := range app.KeyStates {
		key_state.Get().Update()
	}

	gameState := app.GameState

	if gameState.Game.Get().IsTerminal() {
		// Send training data synchronously in training mode, skip in evaluation mode
		if !app.EvaluationMode {
			training_data := gameState.Game.Get().TrainingData(gameState.PositionHistory)
			_, err := app.Client.AppendDataset(context.Background(), training_data)
			if err != nil {
				// Log but don't crash on training data submission errors
				println("[WARN] Failed to submit training data:", err.Error())
			}
		}

		// Reset game state for next game
		new_game := environment.NewGame(9, 9, 4, 6.5)
		gameState.Game.Set(new_game)
		gameState.PositionHistory = make([]*utils.LockedPair[*environment.Game, []int], 0)

		return nil // Stop updates during training submission
	}

	if app.KeyStates[ebiten.KeySpace].Get().JustPressed() {
		app.IsPaused.Set(!app.IsPaused.Get())
	}

	var current_agent agents.Agent
	if gameState.Game.Get().Board.CurrentPlayer == environment.Black {
		current_agent = gameState.BlackAgent
	} else {
		current_agent = gameState.WhiteAgent
	}

	select {
	case app.MoveSearchInitiated <- true:
		// We are the first to initiate the move search, start the goroutine
		// to compute the move in the background
		go func() {
			defer func() { <-app.MoveSearchInitiated }() // Ensure that we reset the channel when done
			gameState.IsThinking.Set(true)
			var game_copy *environment.Game = gameState.Game.Get().DeepCopy()

			var action environment.Action = current_agent.SelectAction(game_copy)
			gameState.IsThinking.Set(false)
			for app.IsPaused.Get() {
				// Wait for the space key to be pressed to play the move, this allows the user to see the move before it is played
			}

			// Store game state and root visit counts for training data BEFORE playing the move
			// Get a fresh copy of the current game state (gameState.Game hasn't been modified by MCTS)
			// This ensures the stored game state matches the MCTS Root's legal actions
			mcts_agent := current_agent.(*agents.MctsAgent)
			var position_state *environment.Game = gameState.Game.Get().DeepCopy()
			gameState.PositionHistory = append(gameState.PositionHistory, utils.NewLockedPair(position_state, mcts_agent.Root.GetN()))

			game_copy.PlayAction(action)
			gameState.Game.Set(game_copy)
		}()
	default:
		// Another goroutine has already initiated the move search, do nothing
	}

	return nil
}

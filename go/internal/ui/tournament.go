package ui

import (
	"fmt"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/gen/proto/remote_trainer"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/agents"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
)

type TournamentResult struct {
	SimsA           int
	SimsB           int
	WinrateAAsBlack float64 // A plays black, B plays white
	WinrateAAsWhite float64 // A plays white, B plays black
}

type TournamentManager struct {
	InferenceClient remote_trainer.PositionEvaluatorClient
	Results         map[string]TournamentResult
}

func NewTournamentManager(inference_client remote_trainer.PositionEvaluatorClient) *TournamentManager {
	return &TournamentManager{
		InferenceClient: inference_client,
		Results:         make(map[string]TournamentResult),
	}
}

func (tm *TournamentManager) RunTournament() {
	simulations := []int{200, 800, 3200}
	numGamesPerPairing := 10 // 10 as black, 10 as white (20 total for cross play, 10 total for self-play)

	fmt.Println("\n========== TOURNAMENT START ==========")
	fmt.Printf("Testing %d agents: 200, 800, 3200 simulations per move\n", len(simulations))
	fmt.Printf("Each pairing: %d games (self-play and cross-play included)\n\n", numGamesPerPairing)

	// Round-robin tournament: play all pairings (full matrix)
	for _, simsA := range simulations {
		for _, simsB := range simulations {

			fmt.Printf("Match: %d sims vs %d sims ... ", simsA, simsB)

			winsAAsBlack := 0
			winsAAsWhite := 0

			if simsA == simsB {
				// Self-play: only play 10 games (both will get black and white)
				for game := 0; game < numGamesPerPairing; game++ {
					agentA := agents.NewPuctAgentEval(simsA, 8, 16, -0.9, tm.InferenceClient)
					agentB := agents.NewPuctAgentEval(simsB, 8, 16, -0.9, tm.InferenceClient)

					winner := tm.PlayGame(agentA, agentB)
					if winner == environment.Black {
						winsAAsBlack++ // AgentA (black) wins
					} else if winner == environment.White {
						winsAAsWhite++ // AgentB (white) wins, but since they're same strength, this is white winrate
					}
				}
			} else {
				// Cross-play: 10 as black, 10 as white
				// Round 1: A plays black, B plays white
				for game := 0; game < numGamesPerPairing; game++ {
					agentA := agents.NewPuctAgentEval(simsA, 8, 16, -0.9, tm.InferenceClient)
					agentB := agents.NewPuctAgentEval(simsB, 8, 16, -0.9, tm.InferenceClient)

					winner := tm.PlayGame(agentA, agentB)
					if winner == environment.Black {
						winsAAsBlack++
					}
				}

				// Round 2: B plays black, A plays white (swap agents)
				for game := 0; game < numGamesPerPairing; game++ {
					agentA := agents.NewPuctAgentEval(simsA, 8, 16, -0.9, tm.InferenceClient)
					agentB := agents.NewPuctAgentEval(simsB, 8, 16, -0.9, tm.InferenceClient)

					winner := tm.PlayGame(agentB, agentA) // Note: B is black now
					if winner == environment.White {      // A is white
						winsAAsWhite++
					}
				}
			}

			winrateAAsBlack := float64(winsAAsBlack) / float64(numGamesPerPairing)
			winrateAAsWhite := float64(winsAAsWhite) / float64(numGamesPerPairing)

			key := fmt.Sprintf("%d_vs_%d", simsA, simsB)
			tm.Results[key] = TournamentResult{
				SimsA:           simsA,
				SimsB:           simsB,
				WinrateAAsBlack: winrateAAsBlack,
				WinrateAAsWhite: winrateAAsWhite,
			}

			fmt.Printf("Done! Black: %.1f%%, White: %.1f%%\n", winrateAAsBlack*100, winrateAAsWhite*100)
		}
	}

	tm.PrintResults()
}

func (tm *TournamentManager) PlayGame(blackAgent agents.Agent, whiteAgent agents.Agent) environment.Stone {
	game := environment.NewGame(9, 9, 4, 6.5)
	totalMoves := 0
	maxMoves := 162 // 9x9 board

	for !game.IsTerminal() && totalMoves < maxMoves {
		var agent agents.Agent

		game_copy := game.DeepCopy()

		if game.Board.CurrentPlayer == environment.Black {
			agent = blackAgent
		} else {
			agent = whiteAgent
		}

		action := agent.SelectAction(game_copy)
		game.PlayAction(action)
		totalMoves++
	}

	// Determine winner
	return game.GetWinner()
}

func (tm *TournamentManager) PrintResults() {
	simulations := []int{200, 800, 3200}

	fmt.Println("\n========== TOURNAMENT RESULTS ==========")
	fmt.Println("\nWinrate table (rows vs columns):")
	fmt.Println("Format: Row as Black / Row as White")
	fmt.Print("        ")
	for _, sims := range simulations {
		fmt.Printf("%14d ", sims)
	}
	fmt.Println()

	for _, simsA := range simulations {
		fmt.Printf("%4d: ", simsA)

		for _, simsB := range simulations {
			key := fmt.Sprintf("%d_vs_%d", simsA, simsB)

			// Display result directly from full matrix (no flipping)
			if result, exists := tm.Results[key]; exists {
				fmt.Printf("  %5.1f%%/%5.1f%%  ", result.WinrateAAsBlack*100, result.WinrateAAsWhite*100)
			} else {
				fmt.Print("   ?  /  ?   ")
			}
		}
		fmt.Println()
	}

	fmt.Println("\n========== TOURNAMENT END ==========\n")
}

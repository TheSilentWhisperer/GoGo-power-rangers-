package environment

import (
	"fmt"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/gen/proto/remote_trainer"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/utils"
)

type Score struct {
	Black float64
	White float64
}

func NewScore(black, white float64) Score {
	return Score{
		Black: black,
		White: white,
	}
}

// Struct
type Game struct {
	Komi         float64
	Board        *Board
	LegalActions []Action
	BoardHasher  *BoardHasher
}

// Constructor
func NewGame(height, width, history_length int, komi float64) *Game {
	var game *Game = &Game{
		Komi:         komi,
		Board:        NewBoard(height, width, history_length),
		LegalActions: make([]Action, 0),
		BoardHasher:  NewBoardHasher(height, width),
	}
	game.ComputeLegalActions()
	game.BoardHasher.UpdateHashHistory()
	return game
}

func (game *Game) DeepCopy() *Game {
	var game_copy *Game = &Game{
		Komi:         game.Komi,
		Board:        game.Board.DeepCopy(),
		LegalActions: make([]Action, len(game.LegalActions)),
		BoardHasher:  game.BoardHasher.DeepCopy(),
	}
	copy(game_copy.LegalActions, game.LegalActions)
	return game_copy
}

func (game *Game) GetLegalMask() []bool {
	var board *Board = game.Board
	var legal_mask []bool = make([]bool, 2+board.Height*board.Width)
	legal_mask[0] = true // Resign action is always legal
	legal_mask[1] = true // Pass action is always legal
	for _, action := range game.LegalActions {
		switch action := action.(type) {
		case PutStone:
			var put_stone PutStone = action
			legal_mask[2+put_stone.I*board.Width+put_stone.J] = true
		}
	}
	return legal_mask
}

func (game *Game) EvaluationRequest(request_id int64, flush bool) *remote_trainer.EvaluationRequest {
	var board *Board = game.Board
	var history_length int = game.Board.HistoryLength

	var flattened_board_history []int64 = make([]int64, history_length*board.Height*board.Width)

	for h := 0; h < history_length; h++ {
		for i := 0; i < board.Height; i++ {
			for j := 0; j < board.Width; j++ {
				if h >= board.Matrix.Len() {
					flattened_board_history[h*board.Height*board.Width+i*board.Width+j] = 0
					continue
				}
				switch board.Matrix.At(h)[i][j] {
				case Empty:
					flattened_board_history[h*board.Height*board.Width+i*board.Width+j] = 0
				case board.CurrentPlayer:
					flattened_board_history[h*board.Height*board.Width+i*board.Width+j] = 1
				case board.CurrentPlayer.Opponent():
					flattened_board_history[h*board.Height*board.Width+i*board.Width+j] = -1
				}
			}
		}
	}

	var player_color, pass_count int64
	switch board.CurrentPlayer {
	case Black:
		player_color = 1
	case White:
		player_color = -1
	default:
		panic("Invalid current player color")
	}

	switch board.Passes {
	case NewPasses(true, false):
		pass_count = 1
	case NewPasses(false, true):
		pass_count = 1
	case NewPasses(true, true):
		pass_count = 2
	default:
		pass_count = 0
	}

	var enemy_passed int64
	switch board.CurrentPlayer {
	case Black:
		enemy_passed = pass_count
	case White:
		enemy_passed = -pass_count
	default:
		panic("Invalid current player color")
	}

	var message *remote_trainer.EvaluationRequest = &remote_trainer.EvaluationRequest{
		RequestId:             int64(request_id),
		HistoryLength:         int64(history_length),
		Height:                int64(board.Height),
		Width:                 int64(board.Width),
		FlattenedBoardHistory: flattened_board_history,
		BlackToPlay:           player_color,
		EnemyPassed:           enemy_passed,
		LegalActionsMask:      game.GetLegalMask(),
		Flush:                 flush,
	}
	return message
}

func (game *Game) TrainingData(position_history []*utils.LockedPair[*Game, []int]) *remote_trainer.TrainingData {

	var data []*remote_trainer.TrainingSample = make([]*remote_trainer.TrainingSample, len(position_history))

	// Determine outcome from Black's perspective (always consistent reference frame)
	var final_winner Stone = game.GetWinner()
	var value float64 = map[bool]float64{true: 1.0, false: -1.0}[final_winner == Black]

	for i, locked_pair := range position_history {
		var game *Game = locked_pair.Pair.First
		var N []int = locked_pair.Pair.Second

		// Sanity check: game.LegalActions and N must have the same length
		if len(game.LegalActions) != len(N) {
			panic(fmt.Sprintf("Position %d: LegalActions length (%d) != N length (%d)", i, len(game.LegalActions), len(N)))
		}

		var visit_counts []int64 = make([]int64, 83)
		for i, action := range game.LegalActions {
			switch action := action.(type) {
			case PutStone:
				var put_stone PutStone = action
				visit_counts[2+put_stone.I*game.Board.Width+put_stone.J] = int64(N[i])
			case Pass:
				visit_counts[1] = int64(N[i])
			case Resign:
				visit_counts[0] = int64(N[i])
			default:
				panic("Unknown action type")
			}
		}

		var training_sample *remote_trainer.TrainingSample = &remote_trainer.TrainingSample{
			Inputs:       game.EvaluationRequest(0, false),
			PolicyTarget: visit_counts,
		}
		data[i] = training_sample
	}

	var training_data *remote_trainer.TrainingData = &remote_trainer.TrainingData{
		Data:  data,
		Value: value,
	}
	return training_data
}

// Methods
func (game *Game) ComputeScore() Score {
	// Count stones first
	var black_stones, white_stones, empty_cells int
	for i := 0; i < game.Board.Height; i++ {
		for j := 0; j < game.Board.Width; j++ {
			switch game.Board.Matrix.Front()[i][j] {
			case Black:
				black_stones++
			case White:
				white_stones++
			default:
				empty_cells++
			}
		}
	}

	// Prepare visited map for empty-region flood-fill
	var visited [][]bool = make([][]bool, game.Board.Height)
	for i := range visited {
		visited[i] = make([]bool, game.Board.Width)
	}

	var black_territory, white_territory, neutral_points int

	// For each empty cell not yet visited, BFS its connected empty region
	for i := 0; i < game.Board.Height; i++ {
		for j := 0; j < game.Board.Width; j++ {
			if visited[i][j] || game.Board.Matrix.Front()[i][j] != Empty {
				continue
			}

			// BFS queue
			var queue []Position
			queue = append(queue, NewPosition(i, j))
			visited[i][j] = true
			var component []Position
			surroundsBlack := false
			surroundsWhite := false

			for len(queue) > 0 {
				pos := queue[0]
				queue = queue[1:]
				component = append(component, pos)

				neighbors := game.Board.GetNeighbors(pos.First, pos.Second)
				for npos, nstone := range neighbors {
					switch nstone {
					case Empty:
						if !visited[npos.First][npos.Second] {
							visited[npos.First][npos.Second] = true
							queue = append(queue, npos)
						}
					case Black:
						surroundsBlack = true
					case White:
						surroundsWhite = true
					}
				}
			}

			// Assign territory: only if surrounded exclusively by one color
			if surroundsBlack && !surroundsWhite {
				black_territory += len(component)
			} else if surroundsWhite && !surroundsBlack {
				white_territory += len(component)
			} else {
				neutral_points += len(component)
			}
		}
	}

	// Final scores: stones + territory (komi applied to white)
	var black_score float64 = float64(black_stones + black_territory)
	var white_score float64 = float64(white_stones + white_territory)
	white_score += game.Komi

	return NewScore(black_score, white_score)
}

func (game *Game) GetWinner() Stone {
	if game.Board.Winner != Empty {
		return game.Board.Winner
	}
	if game.Board.Resigned != Empty {
		return game.Board.Resigned.Opponent()
	}
	var score Score = game.ComputeScore()
	if score.Black > score.White {
		game.Board.Winner = Black
		return Black
	} else {
		game.Board.Winner = White
		return White
	}
}

func (game *Game) IsTerminal() bool {
	return (game.Board.Passes.Black && game.Board.Passes.White) || game.Board.Resigned != Empty
}

func (game *Game) GetNeighboringLiberties(i, j int) (int, map[Position]int, map[Position]int) {
	var liberties int = 0
	var friendly_shared_liberties map[Position]int = make(map[Position]int)
	var enemy_shared_liberties map[Position]int = make(map[Position]int)

	var neighbors map[Position]Stone = game.Board.GetNeighbors(i, j)

	for neighbor, neighbor_stone := range neighbors {
		if neighbor_stone == Empty {
			liberties++
			continue
		}
		var neighbor_root Position = game.Board.UnionFind.Find(neighbor)
		switch neighbor_stone {
		case game.Board.CurrentPlayer:
			friendly_shared_liberties[neighbor_root] += 1
			liberties++
		default:
			enemy_shared_liberties[neighbor_root] += 1
		}
	}
	return liberties, friendly_shared_liberties, enemy_shared_liberties
}

func (game *Game) IsLegalAction(i, j int) bool {

	if game.Board.Matrix.Front()[i][j] != Empty {
		return false
	}

	var liberties int
	var friendly_shared_liberties, enemy_shared_liberties map[Position]int
	liberties, friendly_shared_liberties, enemy_shared_liberties = game.GetNeighboringLiberties(i, j)

	//Any capturing move is legal iff it does not violate superko
	for enemy_root, shared_liberties := range enemy_shared_liberties {
		var enemy_group *Group = game.Board.UnionFind.Groups[enemy_root]
		if enemy_group.Liberties-shared_liberties == 0 {
			var captured_stones map[Position]Stone = game.Board.GetCapturedStones(enemy_group)
			var placed_pos Position = NewPosition(i, j)
			var placed_stone Stone = game.Board.CurrentPlayer
			var resulting_hash uint64 = game.BoardHasher.ComputeResultingHash(captured_stones, placed_pos, placed_stone)
			// Check if resulting hash is in history (Superko rule)
			for _, past_hash := range game.BoardHasher.HashHistory {
				if resulting_hash == past_hash {
					return false
				}
			}
			return true
		}
	}

	//Any suicidal move that does not capture is illegal
	var sum_friendly_liberties int = liberties
	for friendly_root, shared_liberties := range friendly_shared_liberties {
		var friendly_group *Group = game.Board.UnionFind.Groups[friendly_root]
		sum_friendly_liberties += friendly_group.Liberties - 2*shared_liberties
	}
	if sum_friendly_liberties == 0 {
		return false
	}

	//Still need to check for superko
	var captured_stones map[Position]Stone = make(map[Position]Stone) // No captures
	var placed_pos Position = NewPosition(i, j)
	var placed_stone Stone = game.Board.CurrentPlayer
	var resulting_hash uint64 = game.BoardHasher.ComputeResultingHash(captured_stones, placed_pos, placed_stone)
	// Check if resulting hash is in history (Superko rule)
	for _, past_hash := range game.BoardHasher.HashHistory {
		if resulting_hash == past_hash {
			return false
		}
	}
	return true
}

func (game *Game) ComputeLegalActions() {
	var legal_actions []Action = make([]Action, 0, game.Board.Height*game.Board.Width+1)
	// Add resign action
	legal_actions = append(legal_actions, Resign{})
	// Add pass action
	legal_actions = append(legal_actions, Pass{})
	// Add put stone actions
	for i := 0; i < game.Board.Height; i++ {
		for j := 0; j < game.Board.Width; j++ {
			if game.IsLegalAction(i, j) {
				legal_actions = append(legal_actions, PutStone{I: i, J: j})
			}
		}
	}
	game.LegalActions = legal_actions
}

func (game *Game) CaptureGroup(captured_group *Group) {
	var captured_stones map[Position]Stone = game.Board.GetCapturedStones(captured_group)
	for pos, stone := range captured_stones {
		var i, j int = pos.First, pos.Second
		// Remove stone from board and update board hash
		game.Board.Matrix.Front()[i][j] = Empty
		game.BoardHasher.UpdateHash(i, j, stone, Empty, false)
		// Remove stone from union-find
		game.Board.UnionFind.RemoveStone(pos)
		// Update neighboring friendly groups' liberties
		var neighbors map[Position]Stone = game.Board.GetNeighbors(i, j)
		for neighbor, neighbor_stone := range neighbors {
			if neighbor_stone == game.Board.CurrentPlayer {
				var neighbor_root Position = game.Board.UnionFind.Find(neighbor)
				var neighbor_group *Group = game.Board.UnionFind.Groups[neighbor_root]
				neighbor_group.Liberties++
			}
		}
	}

	// Remove the captured group from union-find
	game.Board.UnionFind.RemoveGroup(captured_group)
}

func (game *Game) PutStone(i, j int) {

	liberties, friendly_shared_liberties, enemy_shared_liberties := game.GetNeighboringLiberties(i, j)

	// Place the Stone and update board hash
	game.Board.Matrix.Front()[i][j] = game.Board.CurrentPlayer
	game.BoardHasher.UpdateHash(i, j, Empty, game.Board.CurrentPlayer, false)

	// Add new stone to union-find
	game.Board.UnionFind.AddStone(NewPosition(i, j), liberties)

	var new_stone_pos Position = NewPosition(i, j)
	var new_stone_root Position = game.Board.UnionFind.Find(new_stone_pos)
	var new_stone_group *Group = game.Board.UnionFind.Groups[new_stone_root]

	// Merge with friendly groups
	for friendly_root, shared_liberties := range friendly_shared_liberties {
		var friendly_group *Group = game.Board.UnionFind.Groups[friendly_root]
		new_stone_group = game.Board.UnionFind.Union(friendly_group, new_stone_group, shared_liberties)
	}

	// Check for captures
	for enemy_root, shared_liberties := range enemy_shared_liberties {
		var enemy_group *Group = game.Board.UnionFind.Groups[enemy_root]
		if enemy_group.Liberties-shared_liberties == 0 {
			// Capture the group (hash is updated inside CaptureGroup)
			game.CaptureGroup(enemy_group)
		} else {
			// Update liberties of the enemy group
			enemy_group.Liberties -= shared_liberties
		}
	}
}

func (game *Game) PlayAction(action Action) {

	//push a copy of the front of the queue to the front of the history to be modified by the action
	var board_copy [][]Stone = make([][]Stone, game.Board.Height)
	for i := range board_copy {
		board_copy[i] = make([]Stone, game.Board.Width)
		copy(board_copy[i], game.Board.Matrix.Front()[i])
	}
	// Remove the oldest board state from the back if we exceed the history length (this will happen when we have a full history and push a new board state to the front)
	if game.Board.Matrix.Len() == game.Board.HistoryLength {
		game.Board.Matrix.PopBack()
	}
	game.Board.Matrix.PushFront(board_copy)

	switch a := action.(type) {
	case PutStone:
		game.PutStone(a.I, a.J)
		game.Board.Passes = NewPasses(false, false) // Reset passes after a move
	case Pass:
		switch game.Board.CurrentPlayer {
		case Black:
			game.Board.Passes.Black = true
		case White:
			game.Board.Passes.White = true
		}
	case Resign:
		switch game.Board.CurrentPlayer {
		case Black:
			game.Board.Resigned = Black
		case White:
			game.Board.Resigned = White
		}
	}

	// Switch current player
	if game.Board.CurrentPlayer == Black {
		game.Board.CurrentPlayer = White
	} else {
		game.Board.CurrentPlayer = Black
	}
	// Update board hash for player switch
	game.BoardHasher.UpdateHash(0, 0, Empty, Empty, true)

	// Recompute legal actions and update hash history
	game.BoardHasher.UpdateHashHistory()
	game.ComputeLegalActions()
}

// Debugging and Display
func (game *Game) DebugLiberties() {
	for _, _ = range game.Board.UnionFind.Groups {
		// println("Group at (", root_pos.First, ",", root_pos.Second, ") has", group.Liberties, "liberties")
	}
}

func (game *Game) DebugHasher() {
	if len(game.BoardHasher.HashHistory) > 1000 {
		println(len(game.BoardHasher.HashHistory), " hashes recorded. Latest hash:")
		game.DisplayBoard()
		println("Current board hash:", game.BoardHasher.BoardHash)
	}
}

func (game *Game) DisplayBoard() {
	var stone_to_char = map[Stone]string{
		Empty: ".",
		Black: "○",
		White: "●",
	}

	// Print board
	for i := 0; i < game.Board.Height; i++ {
		for j := 0; j < game.Board.Width; j++ {
			print(stone_to_char[game.Board.Matrix.Front()[i][j]], " ")
		}
		println()
	}

	// Print current player
	println("Current Player:", stone_to_char[game.Board.CurrentPlayer])
}

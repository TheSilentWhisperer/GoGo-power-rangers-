package environment

import "github.com/TheSilentWhisperer/GoGo-power-rangers-/go/gen/proto/remote_trainer"

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

func (game *Game) Message(request_id int) *remote_trainer.EvaluationRequest {
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
				case Black:
					flattened_board_history[h*board.Height*board.Width+i*board.Width+j] = 1
				case White:
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
		panic("We should not ask for a position evaluation of a terminal position")
	default:
		pass_count = 0
	}

	var message *remote_trainer.EvaluationRequest = &remote_trainer.EvaluationRequest{
		RequestId:             int64(request_id),
		HistoryLength:         int64(history_length),
		Height:                int64(board.Height),
		Width:                 int64(board.Width),
		FlattenedBoardHistory: flattened_board_history,
		BlackToPlay:           player_color,
		EnemyPassed:           pass_count,
		LegalActionsMask:      game.GetLegalMask(),
	}
	return message
}

// Methods
func (game *Game) ComputeScore() Score {
	var black_score float64 = 0.0
	var white_score float64 = game.Komi
	// dfs from each empty point to determine territory
	var is_black [][]bool = make([][]bool, game.Board.Height)
	var is_white [][]bool = make([][]bool, game.Board.Height)
	for i := range is_white {
		is_white[i] = make([]bool, game.Board.Width)
	}
	for i := range is_black {
		is_black[i] = make([]bool, game.Board.Width)
	}
	var visited [][]bool = make([][]bool, game.Board.Height)
	for i := range visited {
		visited[i] = make([]bool, game.Board.Width)
	}

	var dfs func(i, j int)

	dfs = func(i, j int) {
		visited[i][j] = true
		var neighbors map[Position]Stone = game.Board.GetNeighbors(i, j)
		for neighbor, neighbor_stone := range neighbors {
			switch neighbor_stone {
			case Empty:
				if !visited[neighbor.First][neighbor.Second] {
					dfs(neighbor.First, neighbor.Second)
				}
				is_black[i][j] = is_black[neighbor.First][neighbor.Second] || is_black[i][j]
				is_white[i][j] = is_white[neighbor.First][neighbor.Second] || is_white[i][j]
			case Black:
				is_black[i][j] = true
			case White:
				is_white[i][j] = true
			}
		}
	}

	for i := 0; i < game.Board.Height; i++ {
		for j := 0; j < game.Board.Width; j++ {
			if !visited[i][j] && game.Board.Matrix.Front()[i][j] == Empty {
				dfs(i, j)
			}
		}
	}

	for i := 0; i < game.Board.Height; i++ {
		for j := 0; j < game.Board.Width; j++ {
			switch game.Board.Matrix.Front()[i][j] {
			case Black:
				black_score += 1.0
			case White:
				white_score += 1.0
			case Empty:
				if is_black[i][j] && !is_white[i][j] {
					black_score += 1.0
				} else if is_white[i][j] && !is_black[i][j] {
					white_score += 1.0
				} else {
					black_score += 0.5
					white_score += 0.5
				}
			}
		}
	}

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
	for root_pos, group := range game.Board.UnionFind.Groups {
		println("Group at (", root_pos.First, ",", root_pos.Second, ") has", group.Liberties, "liberties")
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

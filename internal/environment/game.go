package environment

type Score struct {
	black float64
	white float64
}

func newScore(black, white float64) Score {
	return Score{
		black: black,
		white: white,
	}
}

// Struct
type Game struct {
	komi         float64
	Board        *board
	LegalActions []Action
	boardHasher  *boardHasher
}

// Constructor
func NewGame(height, width int, komi float64) *Game {
	var game *Game = &Game{
		komi:         komi,
		Board:        newBoard(height, width),
		LegalActions: make([]Action, 0),
		boardHasher:  newBoardHasher(height, width),
	}
	game.computeLegalActions()
	game.boardHasher.updateHashHistory()
	return game
}

func (game *Game) DeepCopy() *Game {
	var game_copy *Game = &Game{
		komi:         game.komi,
		Board:        game.Board.deepCopy(),
		LegalActions: make([]Action, len(game.LegalActions)),
		boardHasher:  game.boardHasher.deepCopy(),
	}
	copy(game_copy.LegalActions, game.LegalActions)
	return game_copy
}

// Methods
func (game *Game) ComputeScore() Score {
	var black_score float64 = 0.0
	var white_score float64 = game.komi
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
		var neighbors map[Position]Stone = game.Board.getNeighbors(i, j)
		for neighbor, neighbor_stone := range neighbors {
			switch neighbor_stone {
			case Empty:
				if !visited[neighbor.I][neighbor.J] {
					dfs(neighbor.I, neighbor.J)
				}
				is_black[i][j] = is_black[neighbor.I][neighbor.J] || is_black[i][j]
				is_white[i][j] = is_white[neighbor.I][neighbor.J] || is_white[i][j]
			case Black:
				is_black[i][j] = true
			case White:
				is_white[i][j] = true
			}
		}
	}

	for i := 0; i < game.Board.Height; i++ {
		for j := 0; j < game.Board.Width; j++ {
			if !visited[i][j] && game.Board.Matrix[i][j] == Empty {
				dfs(i, j)
			}
		}
	}

	for i := 0; i < game.Board.Height; i++ {
		for j := 0; j < game.Board.Width; j++ {
			switch game.Board.Matrix[i][j] {
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

	return newScore(black_score, white_score)
}

func (game *Game) GetWinner() Stone {
	if game.Board.resigned != Empty {
		return game.Board.resigned.Opponent()
	}
	var score Score = game.ComputeScore()
	if score.black > score.white {
		return Black
	} else {
		return White
	}
}

func (game *Game) IsTerminal() bool {
	return (game.Board.Passes.Black && game.Board.Passes.White) || game.Board.resigned != Empty
}

func (game *Game) getNeighboringLiberties(i, j int) (int, map[Position]int, map[Position]int) {
	var liberties int = 0
	var friendly_shared_liberties map[Position]int = make(map[Position]int)
	var enemy_shared_liberties map[Position]int = make(map[Position]int)

	var neighbors map[Position]Stone = game.Board.getNeighbors(i, j)

	for neighbor, neighbor_stone := range neighbors {
		if neighbor_stone == Empty {
			liberties++
			continue
		}
		var neighbor_root Position = game.Board.unionFind.find(neighbor)
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

func (game *Game) isLegalAction(i, j int) bool {

	if game.Board.Matrix[i][j] != Empty {
		return false
	}

	var liberties int
	var friendly_shared_liberties, enemy_shared_liberties map[Position]int
	liberties, friendly_shared_liberties, enemy_shared_liberties = game.getNeighboringLiberties(i, j)

	//Any capturing move is legal iff it does not violate superko
	for enemy_root, shared_liberties := range enemy_shared_liberties {
		var enemy_group *group = game.Board.unionFind.groups[enemy_root]
		if enemy_group.liberties-shared_liberties == 0 {
			var captured_stones map[Position]Stone = game.Board.getCapturedStones(enemy_group)
			var placed_pos Position = NewPosition(i, j)
			var placed_stone Stone = game.Board.CurrentPlayer
			var resulting_hash uint64 = game.boardHasher.computeResultingHash(captured_stones, placed_pos, placed_stone)
			// Check if resulting hash is in history (Superko rule)
			for _, past_hash := range game.boardHasher.hashHistory {
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
		var friendly_group *group = game.Board.unionFind.groups[friendly_root]
		sum_friendly_liberties += friendly_group.liberties - 2*shared_liberties
	}
	if sum_friendly_liberties == 0 {
		return false
	}

	//Still need to check for superko
	var captured_stones map[Position]Stone = make(map[Position]Stone) // No captures
	var placed_pos Position = NewPosition(i, j)
	var placed_stone Stone = game.Board.CurrentPlayer
	var resulting_hash uint64 = game.boardHasher.computeResultingHash(captured_stones, placed_pos, placed_stone)
	// Check if resulting hash is in history (Superko rule)
	for _, past_hash := range game.boardHasher.hashHistory {
		if resulting_hash == past_hash {
			return false
		}
	}
	return true
}

func (game *Game) computeLegalActions() {
	var legal_actions []Action = make([]Action, 0, game.Board.Height*game.Board.Width+1)
	// Add resign action
	legal_actions = append(legal_actions, Resign{})
	// Add pass action
	legal_actions = append(legal_actions, pass{})
	// Add put stone actions
	for i := 0; i < game.Board.Height; i++ {
		for j := 0; j < game.Board.Width; j++ {
			if game.isLegalAction(i, j) {
				legal_actions = append(legal_actions, putStone{i: i, j: j})
			}
		}
	}
	game.LegalActions = legal_actions
}

func (game *Game) captureGroup(captured_group *group) {

	var capturedStones map[Position]Stone = game.Board.getCapturedStones(captured_group)
	for pos, stone := range capturedStones {
		var i, j int = pos.I, pos.J
		// Remove stone from board and update board hash
		game.Board.Matrix[i][j] = Empty
		game.boardHasher.updateHash(i, j, stone, Empty, false)
		// Remove stone from union-find
		game.Board.unionFind.removeStone(pos)
		// Update neighboring friendly groups' liberties
		var neighbors map[Position]Stone = game.Board.getNeighbors(i, j)
		for neighbor, neighbor_stone := range neighbors {
			if neighbor_stone == game.Board.CurrentPlayer {
				var neighbor_root Position = game.Board.unionFind.find(neighbor)
				var neighbor_group *group = game.Board.unionFind.groups[neighbor_root]
				neighbor_group.liberties++
			}
		}
	}

	// Remove the captured group from union-find
	game.Board.unionFind.removeGroup(captured_group)
}

func (game *Game) putStone(i, j int) {

	liberties, friendly_shared_liberties, enemy_shared_liberties := game.getNeighboringLiberties(i, j)

	// Place the Stone and update board hash
	game.Board.Matrix[i][j] = game.Board.CurrentPlayer
	game.boardHasher.updateHash(i, j, Empty, game.Board.CurrentPlayer, false)

	// Add new stone to union-find
	game.Board.unionFind.addStone(NewPosition(i, j), liberties)

	var new_stone_pos Position = NewPosition(i, j)
	var new_stone_root Position = game.Board.unionFind.find(new_stone_pos)
	var new_stone_group *group = game.Board.unionFind.groups[new_stone_root]

	// Merge with friendly groups
	for friendly_root, shared_liberties := range friendly_shared_liberties {
		var friendly_group *group = game.Board.unionFind.groups[friendly_root]
		new_stone_group = game.Board.unionFind.union(friendly_group, new_stone_group, shared_liberties)
	}

	// Check for captures
	for enemy_root, shared_liberties := range enemy_shared_liberties {
		var enemy_group *group = game.Board.unionFind.groups[enemy_root]
		if enemy_group.liberties-shared_liberties == 0 {
			// Capture the group (hash is updated inside captureGroup)
			game.captureGroup(enemy_group)
		} else {
			// Update liberties of the enemy group
			enemy_group.liberties -= shared_liberties
		}
	}

}

func (game *Game) PlayAction(action Action) {
	switch a := action.(type) {
	case putStone:
		game.putStone(a.i, a.j)
		game.Board.Passes = NewPasses(false, false) // Reset passes after a move
	case pass:
		switch game.Board.CurrentPlayer {
		case Black:
			game.Board.Passes.Black = true
		case White:
			game.Board.Passes.White = true
		}
	case Resign:
		switch game.Board.CurrentPlayer {
		case Black:
			game.Board.resigned = Black
		case White:
			game.Board.resigned = White
		}
	}

	// Switch current player
	if game.Board.CurrentPlayer == Black {
		game.Board.CurrentPlayer = White
	} else {
		game.Board.CurrentPlayer = Black
	}
	// Update board hash for player switch
	game.boardHasher.updateHash(0, 0, Empty, Empty, true)

	// Recompute legal actions and update hash history
	game.boardHasher.updateHashHistory()
	game.computeLegalActions()
}

// Debugging and Display
func (game *Game) DebugLiberties() {
	for root_pos, group := range game.Board.unionFind.groups {
		println("Group at (", root_pos.I, ",", root_pos.J, ") has", group.liberties, "liberties")
	}
}

func (game *Game) DebugHasher() {
	if len(game.boardHasher.hashHistory) > 1000 {
		println(len(game.boardHasher.hashHistory), " hashes recorded. Latest hash:")
		game.DisplayBoard()
		println("Current board hash:", game.boardHasher.boardHash)
	}
}

func (game *Game) DisplayBoard() {
	var stoneToChar = map[Stone]string{
		Empty: ".",
		Black: "○",
		White: "●",
	}

	// Print board
	for i := 0; i < game.Board.Height; i++ {
		for j := 0; j < game.Board.Width; j++ {
			print(stoneToChar[game.Board.Matrix[i][j]], " ")
		}
		println()
	}

	// Print current player
	println("Current Player:", stoneToChar[game.Board.CurrentPlayer])
}

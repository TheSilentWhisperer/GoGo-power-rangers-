package environment

type Game struct {
	Board        *board
	LegalActions []Action
	boardHasher  *boardHasher
}

// Constructor
func NewGame(height, width int) *Game {
	var game *Game = &Game{
		Board:        newBoard(height, width),
		LegalActions: make([]Action, 0),
		boardHasher:  newBoardHasher(height, width),
	}
	game.computeLegalActions()
	game.boardHasher.updateHashHistory()
	return game
}

// Methods
func (game *Game) IsTerminal() bool {
	return game.Board.passes >= 2
}

func (game *Game) getNeighboringLiberties(i, j int) (int, map[position]int, map[position]int) {
	var liberties int = 0
	var friendly_shared_liberties map[position]int = make(map[position]int)
	var enemy_shared_liberties map[position]int = make(map[position]int)

	var neighbors map[position]Stone = game.Board.getNeighbors(i, j)

	for neighbor, neighbor_stone := range neighbors {
		if neighbor_stone == Empty {
			liberties++
			continue
		}
		var neighbor_root position = game.Board.unionFind.find(neighbor)
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

	if game.Board.matrix[i][j] != Empty {
		return false
	}

	var liberties int
	var friendly_shared_liberties, enemy_shared_liberties map[position]int
	liberties, friendly_shared_liberties, enemy_shared_liberties = game.getNeighboringLiberties(i, j)

	//Any capturing move is legal iff it does not violate superko
	for enemy_root, shared_liberties := range enemy_shared_liberties {
		var enemy_group *group = game.Board.unionFind.groups[enemy_root]
		if enemy_group.liberties-shared_liberties == 0 {
			var captured_stones map[position]Stone = game.Board.getCapturedStones(enemy_group)
			var placed_pos position = newPosition(i, j)
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
	var captured_stones map[position]Stone = make(map[position]Stone) // No captures
	var placed_pos position = newPosition(i, j)
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
	var legal_actions []Action = make([]Action, 0, game.Board.height*game.Board.width+1)
	// Add pass action
	legal_actions = append(legal_actions, pass{})
	// Add put stone actions
	for i := 0; i < game.Board.height; i++ {
		for j := 0; j < game.Board.width; j++ {
			if game.isLegalAction(i, j) {
				legal_actions = append(legal_actions, putStone{i: i, j: j})
			}
		}
	}
	game.LegalActions = legal_actions
}

func (game *Game) captureGroup(captured_group *group) {

	var capturedStones map[position]Stone = game.Board.getCapturedStones(captured_group)
	for pos, stone := range capturedStones {
		var i, j int = pos.i, pos.j
		// Remove stone from board and update board hash
		game.Board.matrix[i][j] = Empty
		game.boardHasher.updateHash(i, j, stone, Empty, false)
		// Remove stone from union-find
		game.Board.unionFind.removeStone(pos)
		// Update neighboring friendly groups' liberties
		var neighbors map[position]Stone = game.Board.getNeighbors(i, j)
		for neighbor, neighbor_stone := range neighbors {
			if neighbor_stone == game.Board.CurrentPlayer {
				var neighbor_root position = game.Board.unionFind.find(neighbor)
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
	game.Board.matrix[i][j] = game.Board.CurrentPlayer
	game.boardHasher.updateHash(i, j, Empty, game.Board.CurrentPlayer, false)

	// Add new stone to union-find
	game.Board.unionFind.addStone(newPosition(i, j), liberties)

	var new_stone_pos position = newPosition(i, j)
	var new_stone_root position = game.Board.unionFind.find(new_stone_pos)
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
		game.Board.passes = 0
	case pass:
		game.Board.passes++
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
		println("Group at (", root_pos.i, ",", root_pos.j, ") has", group.liberties, "liberties")
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
	for i := 0; i < game.Board.height; i++ {
		for j := 0; j < game.Board.width; j++ {
			print(stoneToChar[game.Board.matrix[i][j]], " ")
		}
		println()
	}

	// Print current player
	println("Current Player:", stoneToChar[game.Board.CurrentPlayer])
}

package environment

type Go_ struct {
	Board         *board
	Legal_actions []Action
}

// Constructor
func NewGoGame(height, width int) *Go_ {
	var go_ *Go_ = &Go_{
		Board:         newBoard(height, width),
		Legal_actions: make([]Action, 0),
	}
	go_.computeLegalActions()
	return go_
}

// Methods
func (go_ *Go_) IsTerminal() bool {
	return go_.Board.passes >= 2
}

func (go_ *Go_) getNeighbors(i, j int) []position {
	var neighbors []position
	var directions []position = []position{
		{-1, 0}, // Up
		{1, 0},  // Down
		{0, -1}, // Left
		{0, 1},  // Right
	}
	for _, dir := range directions {
		ni, nj := i+dir.i, j+dir.j
		if ni < 0 || ni >= go_.Board.height || nj < 0 || nj >= go_.Board.width {
			continue
		}
		neighbors = append(neighbors, position{i: ni, j: nj})
	}
	return neighbors
}

func (go_ *Go_) getNeighboringLiberties(i, j int) (int, map[position]int, map[position]int) {
	var liberties int = 0
	var friendly_shared_liberties map[position]int = make(map[position]int)
	var enemy_shared_liberties map[position]int = make(map[position]int)

	var neighbors []position = go_.getNeighbors(i, j)

	for _, neighbor := range neighbors {
		ni, nj := neighbor.i, neighbor.j
		neighbor_stone := go_.Board.matrix[ni][nj]
		if neighbor_stone == Empty {
			liberties++
			continue
		}
		var neighbor_root position = go_.Board.unionFind.find(neighbor)
		switch neighbor_stone {
		case go_.Board.CurrentPlayer:
			friendly_shared_liberties[neighbor_root] += 1
			liberties++
		default:
			enemy_shared_liberties[neighbor_root] += 1
		}
	}
	return liberties, friendly_shared_liberties, enemy_shared_liberties
}

func (go_ *Go_) isLegalAction(i, j int) bool {

	if go_.Board.matrix[i][j] != Empty {
		return false
	}

	var liberties int
	var friendly_shared_liberties, enemy_shared_liberties map[position]int
	liberties, friendly_shared_liberties, enemy_shared_liberties = go_.getNeighboringLiberties(i, j)

	//Any capturing move is legal
	for enemy_root, shared_liberties := range enemy_shared_liberties {
		var enemy_group *group = go_.Board.unionFind.groups[enemy_root]
		if enemy_group.liberties-shared_liberties == 0 {
			return true
		}
	}

	//Any suicidal move that does not capture is illegal
	var sum_friendly_liberties int = liberties
	for friendly_root, shared_liberties := range friendly_shared_liberties {
		var friendly_group *group = go_.Board.unionFind.groups[friendly_root]
		sum_friendly_liberties += friendly_group.liberties - 2*shared_liberties
	}
	if sum_friendly_liberties == 0 {
		return false
	}
	return true
}

func (go_ *Go_) computeLegalActions() {
	var legal_actions []Action = make([]Action, 0, go_.Board.height*go_.Board.width+1)
	// Add pass action
	legal_actions = append(legal_actions, pass{})
	// Add put stone actions
	for i := 0; i < go_.Board.height; i++ {
		for j := 0; j < go_.Board.width; j++ {
			if go_.isLegalAction(i, j) {
				legal_actions = append(legal_actions, putStone{i: i, j: j})
			}
		}
	}
	go_.Legal_actions = legal_actions
}

func (go_ *Go_) captureGroup(captured_group *group) {

	var dfs func(pos position)

	dfs = func(pos position) {

		// Remove stone from union-find to mark as visited
		go_.Board.unionFind.removeStone(pos)

		// Remove the stone from the board
		var i, j int = pos.i, pos.j
		go_.Board.matrix[i][j] = Empty

		// Update friendly neighboring groups' liberties and recursively call on enemy neighbors
		var neighbors []position = go_.getNeighbors(i, j)
		for _, neighbor := range neighbors {
			var neighbor_stone Stone = go_.Board.matrix[neighbor.i][neighbor.j]
			if neighbor_stone == Empty {
				continue
			}
			// friendly neighbor
			if neighbor_stone == go_.Board.CurrentPlayer {
				var neighbor_root position = go_.Board.unionFind.find(neighbor)
				var neighbor_group *group = go_.Board.unionFind.groups[neighbor_root]
				neighbor_group.liberties++
			} else { // enemy neighbor
				if _, ok := go_.Board.unionFind.parents[neighbor]; ok {
					dfs(neighbor)
				}
			}
		}
	}
	dfs(captured_group.root)
	go_.Board.unionFind.removeGroup(captured_group)
}

func (go_ *Go_) putStone(i, j int) {

	liberties, friendly_shared_liberties, enemy_shared_liberties := go_.getNeighboringLiberties(i, j)

	// Place the Stone
	go_.Board.matrix[i][j] = go_.Board.CurrentPlayer
	go_.Board.unionFind.addStone(newPosition(i, j), liberties)

	var new_stone_pos position = newPosition(i, j)
	var new_stone_root position = go_.Board.unionFind.find(new_stone_pos)
	var new_stone_group *group = go_.Board.unionFind.groups[new_stone_root]

	// Merge with friendly groups
	for friendly_root, shared_liberties := range friendly_shared_liberties {
		var friendly_group *group = go_.Board.unionFind.groups[friendly_root]
		new_stone_group = go_.Board.unionFind.union(friendly_group, new_stone_group, shared_liberties)
	}

	// Check for captures
	for enemy_root, shared_liberties := range enemy_shared_liberties {
		var enemy_group *group = go_.Board.unionFind.groups[enemy_root]
		if enemy_group.liberties-shared_liberties == 0 {
			// Capture the group
			go_.captureGroup(enemy_group)
		} else {
			// Update liberties of the enemy group
			enemy_group.liberties -= shared_liberties
		}
	}

}

func (go_ *Go_) PlayAction(action Action) {
	switch a := action.(type) {
	case putStone:
		go_.putStone(a.i, a.j)
		go_.Board.passes = 0
	case pass:
		go_.Board.passes++
	}

	// Switch current player
	if go_.Board.CurrentPlayer == Black {
		go_.Board.CurrentPlayer = White
	} else {
		go_.Board.CurrentPlayer = Black
	}

	// Recompute legal actions
	go_.computeLegalActions()
}

func (go_ *Go_) DebugLiberties() {
	for root_pos, group := range go_.Board.unionFind.groups {
		println("Group at (", root_pos.i, ",", root_pos.j, ") has", group.liberties, "liberties")
	}
}

func (go_ *Go_) DisplayBoard() {
	var stoneToChar = map[Stone]string{
		Empty: ".",
		Black: "○",
		White: "●",
	}

	// Print board
	for i := 0; i < go_.Board.height; i++ {
		for j := 0; j < go_.Board.width; j++ {
			print(stoneToChar[go_.Board.matrix[i][j]], " ")
		}
		println()
	}

	// Print current player
	println("Current Player:", stoneToChar[go_.Board.CurrentPlayer])
}

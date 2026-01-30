package environment

type board struct {
	height        int
	width         int
	matrix        [][]Stone
	CurrentPlayer Stone
	passes        int
	unionFind     *unionFind
}

// Constructor
func newBoard(height, width int) *board {
	var b *board = &board{
		height:        height,
		width:         width,
		matrix:        make([][]Stone, height),
		CurrentPlayer: Black,
		passes:        0,
		unionFind:     newUnionFind(height, width),
	}
	for i := range b.matrix {
		b.matrix[i] = make([]Stone, width)
	}
	return b
}

// Methods
func (board *board) getNeighbors(i, j int) map[position]Stone {
	var neighbors map[position]Stone = make(map[position]Stone)
	var directions []position = []position{
		{-1, 0}, // Up
		{1, 0},  // Down
		{0, -1}, // Left
		{0, 1},  // Right
	}
	for _, dir := range directions {
		ni, nj := i+dir.i, j+dir.j
		if ni < 0 || ni >= board.height || nj < 0 || nj >= board.width {
			continue
		}
		neighbors[newPosition(ni, nj)] = board.matrix[ni][nj]
	}
	return neighbors
}

func (board *board) getCapturedStones(captured_group *group) map[position]Stone {
	var capturedStones map[position]Stone = make(map[position]Stone)

	//declare signature of dfs function
	var dfs func(pos position)
	var visited map[position]bool = make(map[position]bool)

	//define dfs function
	dfs = func(pos position) {
		visited[pos] = true
		capturedStones[pos] = board.matrix[pos.i][pos.j]
		// Explore neighbors
		var neighbors map[position]Stone = board.getNeighbors(pos.i, pos.j)
		for neighbor, neighbor_stone := range neighbors {
			if neighbor_stone == board.matrix[pos.i][pos.j] {
				if !visited[neighbor] {
					dfs(neighbor)
				}
			}
		}
	}

	// Start DFS from the root of the captured group
	dfs(captured_group.root)
	return capturedStones
}

package environment

type Passes struct {
	Black bool
	White bool
}

func NewPasses(black, white bool) Passes {
	return Passes{
		Black: black,
		White: white,
	}
}

type board struct {
	Height        int
	Width         int
	Matrix        [][]Stone
	CurrentPlayer Stone
	Passes        Passes
	resigned      Stone
	unionFind     *unionFind
}

// Constructor
func newBoard(height, width int) *board {
	var b *board = &board{
		Height:        height,
		Width:         width,
		Matrix:        make([][]Stone, height),
		CurrentPlayer: Black,
		Passes:        NewPasses(false, false),
		resigned:      Empty,
		unionFind:     newUnionFind(height, width),
	}
	for i := range b.Matrix {
		b.Matrix[i] = make([]Stone, width)
	}
	return b
}

func (b *board) deepCopy() *board {
	var board_copy *board = &board{
		Height:        b.Height,
		Width:         b.Width,
		Matrix:        make([][]Stone, b.Height),
		CurrentPlayer: b.CurrentPlayer,
		Passes:        b.Passes,
		resigned:      b.resigned,
		unionFind:     b.unionFind.deepCopy(),
	}
	for i := range b.Matrix {
		board_copy.Matrix[i] = make([]Stone, b.Width)
		copy(board_copy.Matrix[i], b.Matrix[i])
	}
	return board_copy
}

// Methods
func (board *board) getNeighbors(i, j int) map[Position]Stone {
	var neighbors map[Position]Stone = make(map[Position]Stone)
	var directions []Position = []Position{
		{-1, 0}, // Up
		{1, 0},  // Down
		{0, -1}, // Left
		{0, 1},  // Right
	}
	for _, dir := range directions {
		ni, nj := i+dir.I, j+dir.J
		if ni < 0 || ni >= board.Height || nj < 0 || nj >= board.Width {
			continue
		}
		neighbors[NewPosition(ni, nj)] = board.Matrix[ni][nj]
	}
	return neighbors
}

func (board *board) getCapturedStones(captured_group *group) map[Position]Stone {
	var capturedStones map[Position]Stone = make(map[Position]Stone)

	//declare signature of dfs function
	var dfs func(pos Position)
	var visited map[Position]bool = make(map[Position]bool)

	//define dfs function
	dfs = func(pos Position) {
		visited[pos] = true
		capturedStones[pos] = board.Matrix[pos.I][pos.J]
		// Explore neighbors
		var neighbors map[Position]Stone = board.getNeighbors(pos.I, pos.J)
		for neighbor, neighbor_stone := range neighbors {
			if neighbor_stone == board.Matrix[pos.I][pos.J] {
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

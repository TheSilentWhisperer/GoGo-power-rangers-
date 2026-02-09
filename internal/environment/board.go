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

type Board struct {
	Height        int
	Width         int
	Matrix        [][]Stone
	CurrentPlayer Stone
	Passes        Passes
	Resigned      Stone
	UnionFind     *UnionFind
}

// Constructor
func NewBoard(height, width int) *Board {
	var b *Board = &Board{
		Height:        height,
		Width:         width,
		Matrix:        make([][]Stone, height),
		CurrentPlayer: Black,
		Passes:        NewPasses(false, false),
		Resigned:      Empty,
		UnionFind:     NewUnionFind(height, width),
	}
	for i := range b.Matrix {
		b.Matrix[i] = make([]Stone, width)
	}
	return b
}

func (b *Board) DeepCopy() *Board {
	var board_copy *Board = &Board{
		Height:        b.Height,
		Width:         b.Width,
		Matrix:        make([][]Stone, b.Height),
		CurrentPlayer: b.CurrentPlayer,
		Passes:        b.Passes,
		Resigned:      b.Resigned,
		UnionFind:     b.UnionFind.DeepCopy(),
	}
	for i := range b.Matrix {
		board_copy.Matrix[i] = make([]Stone, b.Width)
		copy(board_copy.Matrix[i], b.Matrix[i])
	}
	return board_copy
}

// Methods
func (board *Board) GetNeighbors(i, j int) map[Position]Stone {
	var neighbors map[Position]Stone = make(map[Position]Stone)
	var directions []Position = []Position{
		NewPosition(-1, 0), // Up
		NewPosition(1, 0),  // Down
		NewPosition(0, -1), // Left
		NewPosition(0, 1),  // Right
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

func (board *Board) GetCapturedStones(captured_group *Group) map[Position]Stone {
	var captured_stones map[Position]Stone = make(map[Position]Stone)

	//declare signature of dfs function
	var dfs func(pos Position)
	var visited map[Position]bool = make(map[Position]bool)

	//define dfs function
	dfs = func(pos Position) {
		visited[pos] = true
		captured_stones[pos] = board.Matrix[pos.I][pos.J]
		// Explore neighbors
		var neighbors map[Position]Stone = board.GetNeighbors(pos.I, pos.J)
		for neighbor, neighbor_stone := range neighbors {
			if neighbor_stone == board.Matrix[pos.I][pos.J] {
				if !visited[neighbor] {
					dfs(neighbor)
				}
			}
		}
	}

	// Start DFS from the root of the captured group
	dfs(captured_group.Root)
	return captured_stones
}

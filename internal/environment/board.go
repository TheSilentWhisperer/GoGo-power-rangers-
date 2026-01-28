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

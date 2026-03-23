package environment

import (
	"github.com/gammazero/deque"
)

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
	HistoryLength int
	//Queue of board states, where the first element is the current board state and the last element is the oldest board state
	Matrix        *deque.Deque[[][]Stone]
	CurrentPlayer Stone
	Passes        Passes
	Winner        Stone
	Resigned      Stone
	UnionFind     *UnionFind
}

// Constructor
func NewBoard(height, width, history_length int) *Board {
	var b *Board = &Board{
		Height:        height,
		Width:         width,
		HistoryLength: history_length,
		Matrix:        new(deque.Deque[[][]Stone]),
		CurrentPlayer: Black,
		Passes:        NewPasses(false, false),
		Winner:        Empty,
		Resigned:      Empty,
		UnionFind:     NewUnionFind(height, width),
	}
	var empty_board [][]Stone = make([][]Stone, height)
	for i := range empty_board {
		empty_board[i] = make([]Stone, width)
	}
	b.Matrix.PushFront(empty_board)

	b.Matrix.SetBaseCap(history_length)
	b.Matrix.Grow(history_length)
	return b
}

func (b *Board) DeepCopy() *Board {
	var board_copy *Board = &Board{
		Height:        b.Height,
		Width:         b.Width,
		HistoryLength: b.HistoryLength,
		Matrix:        new(deque.Deque[[][]Stone]),
		CurrentPlayer: b.CurrentPlayer,
		Passes:        b.Passes,
		Winner:        b.Winner,
		Resigned:      b.Resigned,
		UnionFind:     b.UnionFind.DeepCopy(),
	}
	for m := range b.Matrix.Iter() {
		var m_copy [][]Stone = make([][]Stone, len(m))
		for i := range m {
			m_copy[i] = make([]Stone, len(m[i]))
			copy(m_copy[i], m[i])
		}
		board_copy.Matrix.PushBack(m_copy)
	}
	board_copy.Matrix.SetBaseCap(b.HistoryLength)
	board_copy.Matrix.Grow(b.HistoryLength)
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
		ni, nj := i+dir.First, j+dir.Second
		if ni < 0 || ni >= board.Height || nj < 0 || nj >= board.Width {
			continue
		}
		neighbors[NewPosition(ni, nj)] = board.Matrix.Front()[ni][nj]
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
		captured_stones[pos] = board.Matrix.Front()[pos.First][pos.Second]
		// Explore neighbors
		var neighbors map[Position]Stone = board.GetNeighbors(pos.First, pos.Second)
		for neighbor, neighbor_stone := range neighbors {
			if neighbor_stone == board.Matrix.Front()[pos.First][pos.Second] {
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

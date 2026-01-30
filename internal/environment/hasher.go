package environment

import (
	"math/rand"
)

type boardHasher struct {
	height       int
	width        int
	zobristTable [][][]uint64
	playerHash   uint64
	boardHash    uint64 // Cached hash value
	hashHistory  []uint64
}

// Constructor
func (bh *boardHasher) randomizeHasher() {
	bh.zobristTable = make([][][]uint64, bh.height)
	for i := 0; i < bh.height; i++ {
		bh.zobristTable[i] = make([][]uint64, bh.width)
		for j := 0; j < bh.width; j++ {
			bh.zobristTable[i][j] = make([]uint64, 2) // Two stones: Black and White
			for s := 0; s < 2; s++ {
				bh.zobristTable[i][j][s] = rand.Uint64()
			}
		}
	}
	bh.playerHash = rand.Uint64()
}

func newBoardHasher(height, width int) *boardHasher {
	bh := &boardHasher{
		height:      height,
		width:       width,
		hashHistory: make([]uint64, 0),
	}
	bh.randomizeHasher()
	return bh
}

// Methods
func (bh *boardHasher) updateHash(i, j int, oldStone, newStone Stone, updatePlayer bool) {
	// Remove old stone
	if oldStone != Empty {
		bh.boardHash ^= bh.zobristTable[i][j][int(oldStone)%2]
	}
	// Add new stone
	if newStone != Empty {
		bh.boardHash ^= bh.zobristTable[i][j][int(newStone)%2]
	}
	// Update player hash if needed
	if updatePlayer {
		bh.boardHash ^= bh.playerHash
	}
}

func (bh *boardHasher) updateHashHistory() {
	bh.hashHistory = append(bh.hashHistory, bh.boardHash)
}

func (bh *boardHasher) computeResultingHash(capturedStones map[position]Stone, placedPos position, placedStone Stone) uint64 {
	var resultingHash uint64 = bh.boardHash
	// Remove captured stones
	for pos, stone := range capturedStones {
		var i, j int = pos.i, pos.j
		resultingHash ^= bh.zobristTable[i][j][int(stone)%2]
	}
	// Add placed stone
	var i, j int = placedPos.i, placedPos.j
	resultingHash ^= bh.zobristTable[i][j][int(placedStone)%2]
	// Update player hash
	resultingHash ^= bh.playerHash
	return resultingHash
}

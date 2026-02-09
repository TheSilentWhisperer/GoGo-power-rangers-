package environment

import (
	"math/rand"
)

type BoardHasher struct {
	Height       int
	Width        int
	ZobristTable [][][]uint64
	PlayerHash   uint64
	BoardHash    uint64 // Cached hash value
	HashHistory  []uint64
}

// Constructor
func (bh *BoardHasher) RandomizeHasher() {
	bh.ZobristTable = make([][][]uint64, bh.Height)
	for i := 0; i < bh.Height; i++ {
		bh.ZobristTable[i] = make([][]uint64, bh.Width)
		for j := 0; j < bh.Width; j++ {
			bh.ZobristTable[i][j] = make([]uint64, 2) // Two stones: Black and White
			for s := 0; s < 2; s++ {
				bh.ZobristTable[i][j][s] = rand.Uint64()
			}
		}
	}
	bh.PlayerHash = rand.Uint64()
}

func NewBoardHasher(height, width int) *BoardHasher {
	bh := &BoardHasher{
		Height:      height,
		Width:       width,
		HashHistory: make([]uint64, 0),
	}
	bh.RandomizeHasher()
	return bh
}

func (bh *BoardHasher) DeepCopy() *BoardHasher {
	bh_copy := &BoardHasher{
		Height:       bh.Height,
		Width:        bh.Width,
		ZobristTable: make([][][]uint64, bh.Height),
		PlayerHash:   bh.PlayerHash,
		BoardHash:    bh.BoardHash,
		HashHistory:  make([]uint64, len(bh.HashHistory)),
	}
	for i := 0; i < bh.Height; i++ {
		bh_copy.ZobristTable[i] = make([][]uint64, bh.Width)
		for j := 0; j < bh.Width; j++ {
			bh_copy.ZobristTable[i][j] = make([]uint64, 2)
			copy(bh_copy.ZobristTable[i][j], bh.ZobristTable[i][j])
		}
	}
	copy(bh_copy.HashHistory, bh.HashHistory)
	return bh_copy
}

// Methods
func (bh *BoardHasher) UpdateHash(i, j int, old_stone, new_stone Stone, update_player bool) {
	// Remove old stone
	if old_stone != Empty {
		bh.BoardHash ^= bh.ZobristTable[i][j][int(old_stone)%2]
	}
	// Add new stone
	if new_stone != Empty {
		bh.BoardHash ^= bh.ZobristTable[i][j][int(new_stone)%2]
	}
	// Update player hash if needed
	if update_player {
		bh.BoardHash ^= bh.PlayerHash
	}
}

func (bh *BoardHasher) UpdateHashHistory() {
	bh.HashHistory = append(bh.HashHistory, bh.BoardHash)
}

func (bh *BoardHasher) ComputeResultingHash(captured_stones map[Position]Stone, placed_pos Position, placed_stone Stone) uint64 {
	var resulting_hash uint64 = bh.BoardHash
	// Remove captured stones
	for pos, stone := range captured_stones {
		var i, j int = pos.I, pos.J
		resulting_hash ^= bh.ZobristTable[i][j][int(stone)%2]
	}
	// Add placed stone
	var i, j int = placed_pos.I, placed_pos.J
	resulting_hash ^= bh.ZobristTable[i][j][int(placed_stone)%2]
	// Update player hash
	resulting_hash ^= bh.PlayerHash
	return resulting_hash
}

package environment

import (
	"fmt"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/utils"
)

// Position is a 2D board coordinate. Use tuples.Pair to represent it.
type Position = utils.Pair[int, int]

func NewPosition(i, j int) Position { return utils.NewPair(i, j) }

type Group struct {
	Root      Position
	Rank      int
	Liberties int
}

// Constructor
func NewGroup(root Position, liberties int) *Group {
	return &Group{
		Root:      root,
		Rank:      0,
		Liberties: liberties,
	}
}

func (g *Group) DeepCopy() *Group {
	return &Group{
		Root:      g.Root,
		Rank:      g.Rank,
		Liberties: g.Liberties,
	}
}

type UnionFind struct {
	Parents map[Position]Position // parent is self for root nodes
	Groups  map[Position]*Group   // key is root position
}

// Constructor
func NewUnionFind(height, width int) *UnionFind {
	return &UnionFind{
		Parents: make(map[Position]Position),
		Groups:  make(map[Position]*Group),
	}
}

func (uf *UnionFind) DeepCopy() *UnionFind {
	uf_copy := &UnionFind{
		Parents: make(map[Position]Position),
		Groups:  make(map[Position]*Group),
	}
	for pos, parent := range uf.Parents {
		uf_copy.Parents[pos] = parent
	}
	for root, group := range uf.Groups {
		uf_copy.Groups[root] = group.DeepCopy()
	}
	return uf_copy
}

// Methods
func (uf *UnionFind) AddStone(pos Position, liberties int) {
	uf.Parents[pos] = pos
	uf.Groups[pos] = NewGroup(pos, liberties)
}

func (uf *UnionFind) Find(pos Position) Position {
	var parent Position
	var ok bool
	if parent, ok = uf.Parents[pos]; !ok {
		panic(fmt.Sprintf("UnionFind::Find; stone at position {%d,%d} does not exist", pos.First, pos.Second))
	}
	if parent != pos {
		uf.Parents[pos] = uf.Find(parent) // Path compression
		return uf.Parents[pos]
	}
	return parent
}

func (uf *UnionFind) MoveGroup(from_group, to_group *Group, shared_liberties int) {
	// Update parent pointers
	uf.Parents[from_group.Root] = to_group.Root
	// Update group info
	to_group.Liberties += from_group.Liberties - 2*shared_liberties
	delete(uf.Groups, from_group.Root)
}

func (uf *UnionFind) Union(group1, group2 *Group, shared_liberties int) *Group {

	if group1.Rank < group2.Rank {
		uf.MoveGroup(group1, group2, shared_liberties)
		return group2
	} else if group1.Rank > group2.Rank {
		uf.MoveGroup(group2, group1, shared_liberties)
		return group1
	} else {
		uf.MoveGroup(group2, group1, shared_liberties)
		group1.Rank += 1
		return group1
	}
}

func (uf *UnionFind) RemoveStone(pos Position) {
	if _, ok := uf.Parents[pos]; !ok {
		panic(fmt.Sprintf("UnionFind::RemoveStone; stone at position {%d,%d} does not exist", pos.First, pos.Second))
	}
	delete(uf.Parents, pos)
}

func (uf *UnionFind) RemoveGroup(group *Group) {
	delete(uf.Groups, group.Root)
}

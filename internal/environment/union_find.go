package environment

import "fmt"

type Position struct {
	I int
	J int
}

func NewPosition(i, j int) Position {
	return Position{I: i, J: j}
}

type group struct {
	root      Position
	rank      int
	liberties int
}

// Constructor
func newGroup(root Position, liberties int) *group {
	return &group{
		root:      root,
		rank:      0,
		liberties: liberties,
	}
}

func (g *group) deepCopy() *group {
	return &group{
		root:      g.root,
		rank:      g.rank,
		liberties: g.liberties,
	}
}

type unionFind struct {
	parents map[Position]Position // parent is self for root nodes
	groups  map[Position]*group   // key is root position
}

// Constructor
func newUnionFind(height, width int) *unionFind {
	return &unionFind{
		parents: make(map[Position]Position),
		groups:  make(map[Position]*group),
	}
}

func (uf *unionFind) deepCopy() *unionFind {
	uf_copy := &unionFind{
		parents: make(map[Position]Position),
		groups:  make(map[Position]*group),
	}
	for pos, parent := range uf.parents {
		uf_copy.parents[pos] = parent
	}
	for root, group := range uf.groups {
		uf_copy.groups[root] = group.deepCopy()
	}
	return uf_copy
}

// Methods
func (uf *unionFind) addStone(pos Position, liberties int) {
	uf.parents[pos] = pos
	uf.groups[pos] = newGroup(pos, liberties)
}

func (uf *unionFind) find(pos Position) Position {
	var parent Position
	var ok bool
	if parent, ok = uf.parents[pos]; !ok {
		panic(fmt.Sprintf("UnionFind::find; stone at position {%d,%d} does not exist", pos.I, pos.J))
	}
	if parent != pos {
		uf.parents[pos] = uf.find(parent) // Path compression
		return uf.parents[pos]
	}
	return parent
}

func (uf *unionFind) moveGroup(fromGroup, toGroup *group, shared_liberties int) {
	// Update parent pointers
	uf.parents[fromGroup.root] = toGroup.root
	// Update group info
	toGroup.liberties += fromGroup.liberties - 2*shared_liberties
	delete(uf.groups, fromGroup.root)
}

func (uf *unionFind) union(group1, group2 *group, shared_liberties int) *group {

	if group1.rank < group2.rank {
		uf.moveGroup(group1, group2, shared_liberties)
		return group2
	} else if group1.rank > group2.rank {
		uf.moveGroup(group2, group1, shared_liberties)
		return group1
	} else {
		uf.moveGroup(group2, group1, shared_liberties)
		group1.rank += 1
		return group1
	}
}

func (uf *unionFind) removeStone(pos Position) {
	if _, ok := uf.parents[pos]; !ok {
		panic(fmt.Sprintf("UnionFind::removeStone; stone at position {%d,%d} does not exist", pos.I, pos.J))
	}
	delete(uf.parents, pos)
}

func (uf *unionFind) removeGroup(group *group) {
	delete(uf.groups, group.root)
}

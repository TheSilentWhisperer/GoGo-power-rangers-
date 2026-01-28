package environment

import "fmt"

type position struct {
	i int
	j int
}

func newPosition(i, j int) position {
	return position{i: i, j: j}
}

type group struct {
	root      position
	rank      int
	liberties int
}

func newGroup(root position, liberties int) *group {
	return &group{
		root:      root,
		rank:      0,
		liberties: liberties,
	}
}

type unionFind struct {
	parents map[position]position // parent is self for root nodes
	groups  map[position]*group   // key is root position
}

// Constructor
func newUnionFind(height, width int) *unionFind {
	return &unionFind{
		parents: make(map[position]position),
		groups:  make(map[position]*group),
	}
}

// Methods

func (uf *unionFind) addStone(pos position, liberties int) {
	uf.parents[pos] = pos
	uf.groups[pos] = newGroup(pos, liberties)
}

func (uf *unionFind) find(pos position) position {
	var parent position
	var ok bool
	if parent, ok = uf.parents[pos]; !ok {
		panic(fmt.Sprintf("UnionFind::find; stone at position {%d,%d} does not exist", pos.i, pos.j))
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

func (uf *unionFind) removeStone(pos position) {
	if _, ok := uf.parents[pos]; !ok {
		panic(fmt.Sprintf("UnionFind::removeStone; stone at position {%d,%d} does not exist", pos.i, pos.j))
	}
	delete(uf.parents, pos)
}

func (uf *unionFind) removeGroup(group *group) {
	delete(uf.groups, group.root)
}

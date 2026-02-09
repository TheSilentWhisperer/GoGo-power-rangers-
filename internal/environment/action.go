package environment

type Action interface {
	IsAction()
	String() string
}
type PutStone struct {
	I int
	J int
}

func (p PutStone) IsAction() {}

func (p PutStone) String() string {
	return "PutStone(" + string(rune('A'+p.J)) + "," + string(rune('1'+p.I)) + ")"
}

type Pass struct{}

func (p Pass) IsAction() {}

func (p Pass) String() string {
	return "Pass"
}

type Resign struct{}

func (r Resign) IsAction() {}

func (r Resign) String() string {
	return "Resign"
}

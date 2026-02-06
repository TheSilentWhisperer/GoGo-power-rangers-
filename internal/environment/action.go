package environment

type Action interface {
	isAction()
	String() string
}

type putStone struct {
	i int
	j int
}

func (p putStone) isAction() {}

func (p putStone) String() string {
	return "PutStone(" + string(rune('A'+p.j)) + "," + string(rune('1'+p.i)) + ")"
}

type pass struct{}

func (p pass) isAction() {}

func (p pass) String() string {
	return "Pass"
}

type Resign struct{}

func (r Resign) isAction() {}

func (r Resign) String() string {
	return "Resign"
}

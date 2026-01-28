package environment

type Action interface {
	isAction()
}

type putStone struct {
	i int
	j int
}

func (p putStone) isAction() {}

type pass struct{}

func (p pass) isAction() {}

package agents

func NewUctAgent(simulations_per_move int, nb_routines int, resign_threshold float64) *MctsAgent {
	var expander *UctExpander = NewUctExpander(nb_routines)
	return NewMctsAgent(simulations_per_move, nb_routines, resign_threshold, expander)
}

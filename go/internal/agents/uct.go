package agents

func NewUctAgent(simulations_per_move int, nb_routines int, max_parallel_searches int, resign_threshold float64) *MctsAgent {
	var evaluator *UctEvaluator = NewUctEvaluator(max_parallel_searches)
	return NewMctsAgent(simulations_per_move, nb_routines, max_parallel_searches, resign_threshold, evaluator)
}

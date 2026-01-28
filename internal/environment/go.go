package environment

type Go_ struct {
	Board         *board
	Legal_actions []Action
}

// Constructor
func NewGoGame(height, width int) *Go_ {
	var go_ *Go_ = &Go_{
		Board:         newBoard(height, width),
		Legal_actions: make([]Action, 0),
	}
	go_.computeLegalActions()
	return go_
}

// Methods
func (go_ *Go_) IsTerminal() bool {
	return go_.Board.passes >= 2
}

func (go_ *Go_) isLegalAction(i, j int) bool {
	return go_.Board.matrix[i][j] == Empty
}

func (go_ *Go_) computeLegalActions() {
	var legal_actions []Action = make([]Action, 0, go_.Board.height*go_.Board.width+1)
	// Add pass action
	legal_actions = append(legal_actions, pass{})
	// Add put stone actions
	for i := 0; i < go_.Board.height; i++ {
		for j := 0; j < go_.Board.width; j++ {
			if go_.isLegalAction(i, j) {
				legal_actions = append(legal_actions, putStone{i: i, j: j})
			}
		}
	}
	go_.Legal_actions = legal_actions
}

func (go_ *Go_) PlayAction(action Action) {
	switch a := action.(type) {
	case putStone:
		go_.Board.matrix[a.i][a.j] = go_.Board.CurrentPlayer
		go_.Board.passes = 0
	case pass:
		go_.Board.passes++
	}

	// Switch current player
	if go_.Board.CurrentPlayer == Black {
		go_.Board.CurrentPlayer = White
	} else {
		go_.Board.CurrentPlayer = Black
	}

	// Recompute legal actions
	go_.computeLegalActions()
}

func (go_ *Go_) DisplayBoard() {
	var stoneToChar = map[Stone]string{
		Empty: ".",
		Black: "○",
		White: "●",
	}

	// Print board
	for i := 0; i < go_.Board.height; i++ {
		for j := 0; j < go_.Board.width; j++ {
			print(stoneToChar[go_.Board.matrix[i][j]], " ")
		}
		println()
	}
	// Print current player
	println("Current Player:", stoneToChar[go_.Board.CurrentPlayer])
	println()
}

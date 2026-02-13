package environment

type Stone int

const (
	Empty Stone = iota
	Black
	White
)

// Methods
func (s Stone) Opponent() Stone {
	switch s {
	case Black:
		return White
	case White:
		return Black
	default:
		panic("Empty stone has no opponent")
	}
}

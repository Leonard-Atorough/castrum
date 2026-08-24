package system

type Layer int

const (
	Core Layer = iota
	Player
)

func (l Layer) String() string {
	switch l {
	case Core:
		return "CoreLayer"
	case Player:
		return "PlayerLayer"
	default:
		return "UnknownLayer"
	}
}

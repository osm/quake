package typ

import "fmt"

type Type uint8

const (
	Unknown Type = 0
	Palette Type = 64
	QTex    Type = 65
	QPic    Type = 66
	Sound   Type = 67
	MipTex  Type = 68
)

func (t Type) String() string {
	switch t {
	case Palette:
		return "palette"
	case QTex:
		return "qtex"
	case QPic:
		return "qpic"
	case Sound:
		return "sound"
	case MipTex:
		return "miptex"
	default:
		return fmt.Sprintf("LumpType(%d)", t)
	}
}

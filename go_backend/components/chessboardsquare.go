package components

import "fmt"

type ChessBoardSquare struct {
	Sequence int
	IsLight  bool
	BoxWidth string
}

func NewChessBoardSquare(sequence int) ChessBoardSquare {
	isLight := false

	if (sequence/8)%2 == 0 {
		isLight = sequence%2 == 0
	} else {
		isLight = sequence%2 == 1
	}

	return ChessBoardSquare{
		Sequence: sequence,
		IsLight:  isLight,
		BoxWidth: "30px",
	}
}

func (c ChessBoardSquare) Draw() string {
	shade := "dark"
	if c.IsLight {
		shade = "light"
	}

	return fmt.Sprintf("[%d:%s]", c.Sequence, shade)
}

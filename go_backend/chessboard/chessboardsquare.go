// CM3070 FP code
// chessboardsquare.go defines chessboard square data

package chessboard

// ChessBoardSquare stores one square's metadata
type ChessBoardSquare struct {
	Sequence int
	IsLight  bool
	BoxWidth string
}

// NewChessBoardSquare creates a square with alternating color
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

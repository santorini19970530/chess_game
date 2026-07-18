// CM3070 FP code
// chessboardsquare.go defines chessboard square data

package chessboard

// ChessBoardSquare stores one square's metadata
type ChessBoardSquare struct {
	Sequence int
	IsLight  bool
	BoxWidth string
}

// NewChessBoardSquare creates a square with alternating color for an 8-wide board.
func NewChessBoardSquare(sequence int) ChessBoardSquare {
	return NewBoardSquare(sequence, 8)
}

// NewBoardSquare creates a square; files is the board width used for checker parity.
func NewBoardSquare(sequence, files int) ChessBoardSquare {
	if files <= 0 {
		files = 8
	}
	row := sequence / files
	col := sequence % files
	return ChessBoardSquare{
		Sequence: sequence,
		IsLight:  (row+col)%2 == 0,
		BoxWidth: "30px",
	}
}

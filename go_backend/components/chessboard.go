package components

import "fmt"

type ChessBoard struct {
	squares []ChessBoardSquare
}

func NewChessBoard() *ChessBoard {
	board := &ChessBoard{
		squares: make([]ChessBoardSquare, 0, 64),
	}

	for i := 0; i < 64; i++ {
		board.squares = append(board.squares, NewChessBoardSquare(i))
	}

	return board
}

func (c *ChessBoard) Draw() {
	for i, square := range c.squares {
		fmt.Printf("%s ", square.Draw())
		if (i+1)%8 == 0 {
			fmt.Println()
		}
	}
}

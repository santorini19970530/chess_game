package chesspieces

type PieceColor string

const (
	White PieceColor = "white"
	Black PieceColor = "black"
)

type PieceKind string

const (
	Pawn   PieceKind = "pawn"
	Rook   PieceKind = "rook"
	Knight PieceKind = "knight"
	Bishop PieceKind = "bishop"
	Queen  PieceKind = "queen"
	King   PieceKind = "king"
)

type ChessPiece struct {
	Color   PieceColor
	Kind    PieceKind
	ImgFile string
	File    int
	Rank    int
}

var ChessPieces = []ChessPiece{
	// white pieces
	{Color: White, Kind: Rook, ImgFile: "pic/white_rook.png", File: 1, Rank: 1},
	{Color: White, Kind: Knight, ImgFile: "pic/white_knight.png", File: 2, Rank: 1},
	{Color: White, Kind: Bishop, ImgFile: "pic/white_bishop.png", File: 3, Rank: 1},
	{Color: White, Kind: Queen, ImgFile: "pic/white_queen.png", File: 4, Rank: 1},
	{Color: White, Kind: King, ImgFile: "pic/white_king.png", File: 5, Rank: 1},
	{Color: White, Kind: Bishop, ImgFile: "pic/white_bishop.png", File: 6, Rank: 1},
	{Color: White, Kind: Knight, ImgFile: "pic/white_knight.png", File: 7, Rank: 1},
	{Color: White, Kind: Rook, ImgFile: "pic/white_rook.png", File: 8, Rank: 1},
	{Color: White, Kind: Pawn, ImgFile: "pic/white_pawn.png", File: 1, Rank: 2},
	{Color: White, Kind: Pawn, ImgFile: "pic/white_pawn.png", File: 2, Rank: 2},
	{Color: White, Kind: Pawn, ImgFile: "pic/white_pawn.png", File: 3, Rank: 2},
	{Color: White, Kind: Pawn, ImgFile: "pic/white_pawn.png", File: 4, Rank: 2},
	{Color: White, Kind: Pawn, ImgFile: "pic/white_pawn.png", File: 5, Rank: 2},
	{Color: White, Kind: Pawn, ImgFile: "pic/white_pawn.png", File: 6, Rank: 2},
	{Color: White, Kind: Pawn, ImgFile: "pic/white_pawn.png", File: 7, Rank: 2},
	{Color: White, Kind: Pawn, ImgFile: "pic/white_pawn.png", File: 8, Rank: 2},

	// black pieces
	{Color: Black, Kind: Pawn, ImgFile: "pic/black_pawn.png", File: 1, Rank: 7},
	{Color: Black, Kind: Pawn, ImgFile: "pic/black_pawn.png", File: 2, Rank: 7},
	{Color: Black, Kind: Pawn, ImgFile: "pic/black_pawn.png", File: 3, Rank: 7},
	{Color: Black, Kind: Pawn, ImgFile: "pic/black_pawn.png", File: 4, Rank: 7},
	{Color: Black, Kind: Pawn, ImgFile: "pic/black_pawn.png", File: 5, Rank: 7},
	{Color: Black, Kind: Pawn, ImgFile: "pic/black_pawn.png", File: 6, Rank: 7},
	{Color: Black, Kind: Pawn, ImgFile: "pic/black_pawn.png", File: 7, Rank: 7},
	{Color: Black, Kind: Pawn, ImgFile: "pic/black_pawn.png", File: 8, Rank: 7},
	{Color: Black, Kind: Rook, ImgFile: "pic/black_rook.png", File: 1, Rank: 8},
	{Color: Black, Kind: Knight, ImgFile: "pic/black_knight.png", File: 2, Rank: 8},
	{Color: Black, Kind: Bishop, ImgFile: "pic/black_bishop.png", File: 3, Rank: 8},
	{Color: Black, Kind: Queen, ImgFile: "pic/black_queen.png", File: 4, Rank: 8},
	{Color: Black, Kind: King, ImgFile: "pic/black_king.png", File: 5, Rank: 8},
	{Color: Black, Kind: Bishop, ImgFile: "pic/black_bishop.png", File: 6, Rank: 8},
	{Color: Black, Kind: Knight, ImgFile: "pic/black_knight.png", File: 7, Rank: 8},
	{Color: Black, Kind: Rook, ImgFile: "pic/black_rook.png", File: 8, Rank: 8},
}

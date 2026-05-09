// CM3070 FP code
// pieces.go - defines the initial set of chess pieces
// each chess piece is having color, type, image file, and board position

package pieces

var ChessPieces = []ChessPiece{
	{Color: White, Kind: Rook, ImgFile: "pic/chess_pic/rook_light.png", File: 1, Rank: 1},
	{Color: White, Kind: Knight, ImgFile: "pic/chess_pic/knight_light.png", File: 2, Rank: 1},
	{Color: White, Kind: Bishop, ImgFile: "pic/chess_pic/bishop_light.png", File: 3, Rank: 1},
	{Color: White, Kind: Queen, ImgFile: "pic/chess_pic/queen_light.png", File: 4, Rank: 1},
	{Color: White, Kind: King, ImgFile: "pic/chess_pic/king_light.png", File: 5, Rank: 1},
	{Color: White, Kind: Bishop, ImgFile: "pic/chess_pic/bishop_light.png", File: 6, Rank: 1},
	{Color: White, Kind: Knight, ImgFile: "pic/chess_pic/knight_light.png", File: 7, Rank: 1},
	{Color: White, Kind: Rook, ImgFile: "pic/chess_pic/rook_light.png", File: 8, Rank: 1},
	{Color: White, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_light.png", File: 1, Rank: 2},
	{Color: White, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_light.png", File: 2, Rank: 2},
	{Color: White, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_light.png", File: 3, Rank: 2},
	{Color: White, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_light.png", File: 4, Rank: 2},
	{Color: White, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_light.png", File: 5, Rank: 2},
	{Color: White, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_light.png", File: 6, Rank: 2},
	{Color: White, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_light.png", File: 7, Rank: 2},
	{Color: White, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_light.png", File: 8, Rank: 2},
	{Color: Black, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_dark.png", File: 1, Rank: 7},
	{Color: Black, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_dark.png", File: 2, Rank: 7},
	{Color: Black, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_dark.png", File: 3, Rank: 7},
	{Color: Black, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_dark.png", File: 4, Rank: 7},
	{Color: Black, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_dark.png", File: 5, Rank: 7},
	{Color: Black, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_dark.png", File: 6, Rank: 7},
	{Color: Black, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_dark.png", File: 7, Rank: 7},
	{Color: Black, Kind: Pawn, ImgFile: "pic/chess_pic/pawn_dark.png", File: 8, Rank: 7},
	{Color: Black, Kind: Rook, ImgFile: "pic/chess_pic/rook_dark.png", File: 1, Rank: 8},
	{Color: Black, Kind: Knight, ImgFile: "pic/chess_pic/knight_dark.png", File: 2, Rank: 8},
	{Color: Black, Kind: Bishop, ImgFile: "pic/chess_pic/bishop_dark.png", File: 3, Rank: 8},
	{Color: Black, Kind: Queen, ImgFile: "pic/chess_pic/queen_dark.png", File: 4, Rank: 8},
	{Color: Black, Kind: King, ImgFile: "pic/chess_pic/king_dark.png", File: 5, Rank: 8},
	{Color: Black, Kind: Bishop, ImgFile: "pic/chess_pic/bishop_dark.png", File: 6, Rank: 8},
	{Color: Black, Kind: Knight, ImgFile: "pic/chess_pic/knight_dark.png", File: 7, Rank: 8},
	{Color: Black, Kind: Rook, ImgFile: "pic/chess_pic/rook_dark.png", File: 8, Rank: 8},
}

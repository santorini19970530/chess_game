package session

import (
	pieces "go_backend/game/piece"
	"testing"
)

var initialChessPieces = append([]pieces.ChessPiece(nil), pieces.ChessPieces...)

func resetChessPieces() {
	pieces.ChessPieces = append([]pieces.ChessPiece(nil), initialChessPieces...)
	moveHistory = nil
	lastAppliedMove = nil
	resetCastlingState()
	resetGameSessionForTest()
}

func pieceAt(file, rank int) (pieces.ChessPiece, bool) {
	for _, p := range pieces.ChessPieces {
		if p.File == file && p.Rank == rank {
			return p, true
		}
	}
	return pieces.ChessPiece{}, false
}

func TestApplyMoveByCommand_BlackPawnDoubleStep_UCIRejectedOnWhiteTurn(t *testing.T) {
	resetChessPieces()

	if _, err := ApplyMoveByCommand("e7e5"); err == nil {
		t.Fatalf("expected black pawn double step to fail on white turn")
	}
}

func TestApplyMoveByCommand_BlackPawnDoubleStep_SANAfterWhiteMove(t *testing.T) {
	resetChessPieces()

	if _, err := ApplyMoveByCommand("e4"); err != nil {
		t.Fatalf("expected white SAN move e4 to succeed, got error: %v", err)
	}

	normalized, err := ApplyMoveByCommand("e5")
	if err != nil {
		t.Fatalf("expected black SAN move e5 to succeed, got error: %v", err)
	}
	if normalized != "e7e5" {
		t.Fatalf("expected normalized move e7e5, got %q", normalized)
	}

	blackPawn, ok := pieceAt(5, 5)
	if !ok {
		t.Fatalf("expected piece on e5 after move")
	}
	if blackPawn.Kind != pieces.Pawn || blackPawn.Color != pieces.Black {
		t.Fatalf("expected black pawn on e5, got kind=%v color=%v", blackPawn.Kind, blackPawn.Color)
	}
}

func TestApplyMoveByCommand_BlackPawnDoubleStep_SANFromInitialPositionRejected(t *testing.T) {
	resetChessPieces()

	if _, err := ApplyMoveByCommand("g5"); err == nil {
		t.Fatalf("expected SAN g5 to fail on white turn")
	}
}

func TestApplyMoveByCommand_QueenStrategy(t *testing.T) {
	resetChessPieces()

	// Open d-file so white queen can move from d1 to d3.
	if _, err := ApplyMoveByCommand("d2d4"); err != nil {
		t.Fatalf("expected setup move d2d4 to succeed, got error: %v", err)
	}
	if _, err := ApplyMoveByCommand("a7a6"); err != nil {
		t.Fatalf("expected black reply a7a6 to succeed, got error: %v", err)
	}

	if _, err := ApplyMoveByCommand("d1d3"); err != nil {
		t.Fatalf("expected queen move d1d3 to succeed, got error: %v", err)
	}
	queen, ok := pieceAt(4, 3)
	if !ok || queen.Kind != pieces.Queen || queen.Color != pieces.White {
		t.Fatalf("expected white queen on d3 after move")
	}
	if _, err := ApplyMoveByCommand("a6a5"); err != nil {
		t.Fatalf("expected black reply a6a5 to succeed, got error: %v", err)
	}

	// Queen cannot move in knight pattern.
	if _, err := ApplyMoveByCommand("d3e5"); err == nil {
		t.Fatalf("expected d3e5 to fail for queen movement")
	}
}

func TestApplyMoveByCommand_KnightStrategy(t *testing.T) {
	resetChessPieces()

	if _, err := ApplyMoveByCommand("b1c3"); err != nil {
		t.Fatalf("expected knight move b1c3 to succeed, got error: %v", err)
	}
	knight, ok := pieceAt(3, 3)
	if !ok || knight.Kind != pieces.Knight || knight.Color != pieces.White {
		t.Fatalf("expected white knight on c3 after move")
	}
	if _, err := ApplyMoveByCommand("a7a6"); err != nil {
		t.Fatalf("expected black reply a7a6 to succeed, got error: %v", err)
	}

	if _, err := ApplyMoveByCommand("c3c5"); err == nil {
		t.Fatalf("expected c3c5 to fail for knight movement")
	}
}

func TestApplyMoveByCommand_SANQueenCapture_Qxd3(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Queen, File: 4, Rank: 5}, // Qd5
		{Color: pieces.Black, Kind: pieces.Pawn, File: 4, Rank: 3},  // pd3
	}
	moveHistory = nil // white to move

	normalized, err := ApplyMoveByCommand("Qxd3")
	if err != nil {
		t.Fatalf("expected SAN queen capture Qxd3 to succeed, got error: %v", err)
	}
	if normalized != "d5d3" {
		t.Fatalf("expected normalized move d5d3, got %q", normalized)
	}

	queen, ok := pieceAt(4, 3)
	if !ok || queen.Kind != pieces.Queen || queen.Color != pieces.White {
		t.Fatalf("expected white queen on d3 after capture")
	}
}

func TestApplyMoveByCommand_SANPawnCapture_Pxd4(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.Black, Kind: pieces.Pawn, File: 5, Rank: 5}, // pe5
		{Color: pieces.White, Kind: pieces.Pawn, File: 4, Rank: 4}, // pd4
	}
	moveHistory = []string{"White: e2e4"} // black to move

	normalized, err := ApplyMoveByCommand("Pxd4")
	if err != nil {
		t.Fatalf("expected SAN pawn capture Pxd4 to succeed, got error: %v", err)
	}
	if normalized != "e5d4" {
		t.Fatalf("expected normalized move e5d4, got %q", normalized)
	}

	pawn, ok := pieceAt(4, 4)
	if !ok || pawn.Kind != pieces.Pawn || pawn.Color != pieces.Black {
		t.Fatalf("expected black pawn on d4 after capture")
	}
}

func TestApplyMoveByCommand_EnPassant_UCI(t *testing.T) {
	resetChessPieces()

	if _, err := ApplyMoveByCommand("e2e4"); err != nil {
		t.Fatalf("expected e2e4 to succeed, got error: %v", err)
	}
	if _, err := ApplyMoveByCommand("a7a6"); err != nil {
		t.Fatalf("expected a7a6 to succeed, got error: %v", err)
	}
	if _, err := ApplyMoveByCommand("e4e5"); err != nil {
		t.Fatalf("expected e4e5 to succeed, got error: %v", err)
	}
	if _, err := ApplyMoveByCommand("d7d5"); err != nil {
		t.Fatalf("expected d7d5 to succeed, got error: %v", err)
	}

	normalized, err := ApplyMoveByCommand("e5d6")
	if err != nil {
		t.Fatalf("expected en passant e5d6 to succeed, got error: %v", err)
	}
	if normalized != "e5d6" {
		t.Fatalf("expected normalized move e5d6, got %q", normalized)
	}

	whitePawn, ok := pieceAt(4, 6)
	if !ok || whitePawn.Kind != pieces.Pawn || whitePawn.Color != pieces.White {
		t.Fatalf("expected white pawn on d6 after en passant")
	}
	if _, exists := pieceAt(4, 5); exists {
		t.Fatalf("expected captured black pawn at d5 to be removed")
	}
}

func TestApplyMoveByCommand_PawnPromotion_Queen(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Pawn, ImgFile: "pic/chess_pic/pawn_light.png", File: 5, Rank: 7}, // e7
	}
	moveHistory = nil
	lastAppliedMove = nil

	normalized, err := ApplyMoveByCommand("e7e8q")
	if err != nil {
		t.Fatalf("expected promotion e7e8q to succeed, got error: %v", err)
	}
	if normalized != "e7e8q" {
		t.Fatalf("expected normalized move e7e8q, got %q", normalized)
	}

	p, ok := pieceAt(5, 8)
	if !ok {
		t.Fatalf("expected promoted piece on e8")
	}
	if p.Kind != pieces.Queen || p.Color != pieces.White {
		t.Fatalf("expected white queen on e8, got kind=%v color=%v", p.Kind, p.Color)
	}
	if p.ImgFile != "pic/chess_pic/queen_light.png" {
		t.Fatalf("expected queen image, got %q", p.ImgFile)
	}
}

func TestApplyMoveByCommand_PawnUnderpromotion_Knight(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.Black, Kind: pieces.Pawn, ImgFile: "pic/chess_pic/pawn_dark.png", File: 1, Rank: 2}, // a2
	}
	moveHistory = []string{"White: h2h3"} // black to move
	lastAppliedMove = nil

	normalized, err := ApplyMoveByCommand("a2a1n")
	if err != nil {
		t.Fatalf("expected underpromotion a2a1n to succeed, got error: %v", err)
	}
	if normalized != "a2a1n" {
		t.Fatalf("expected normalized move a2a1n, got %q", normalized)
	}

	p, ok := pieceAt(1, 1)
	if !ok {
		t.Fatalf("expected promoted piece on a1")
	}
	if p.Kind != pieces.Knight || p.Color != pieces.Black {
		t.Fatalf("expected black knight on a1, got kind=%v color=%v", p.Kind, p.Color)
	}
	if p.ImgFile != "pic/chess_pic/knight_dark.png" {
		t.Fatalf("expected knight image, got %q", p.ImgFile)
	}
}

func TestApplyMoveByCommand_PawnPromotion_SAN(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Pawn, ImgFile: "pic/chess_pic/pawn_light.png", File: 5, Rank: 7}, // e7
	}
	moveHistory = nil
	lastAppliedMove = nil

	normalized, err := ApplyMoveByCommand("e8=Q")
	if err != nil {
		t.Fatalf("expected SAN promotion e8=Q to succeed, got error: %v", err)
	}
	if normalized != "e7e8q" {
		t.Fatalf("expected normalized move e7e8q, got %q", normalized)
	}

	p, ok := pieceAt(5, 8)
	if !ok || p.Kind != pieces.Queen || p.Color != pieces.White {
		t.Fatalf("expected white queen on e8 after SAN promotion")
	}
}

func TestApplyMoveByCommand_PawnPromotion_SANWithoutEquals(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Pawn, ImgFile: "pic/chess_pic/pawn_light.png", File: 5, Rank: 7}, // e7
	}
	moveHistory = nil
	lastAppliedMove = nil

	normalized, err := ApplyMoveByCommand("e8Q")
	if err != nil {
		t.Fatalf("expected SAN promotion e8Q to succeed, got error: %v", err)
	}
	if normalized != "e7e8q" {
		t.Fatalf("expected normalized move e7e8q, got %q", normalized)
	}
}

func TestApplyMoveByCommand_PawnPromotion_StockfishUppercase(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Pawn, ImgFile: "pic/chess_pic/pawn_light.png", File: 5, Rank: 7}, // e7
	}
	moveHistory = nil
	lastAppliedMove = nil

	normalized, err := ApplyMoveByCommand("e7e8Q")
	if err != nil {
		t.Fatalf("expected stockfish-style promotion e7e8Q to succeed, got error: %v", err)
	}
	if normalized != "e7e8q" {
		t.Fatalf("expected normalized move e7e8q, got %q", normalized)
	}
}

func TestApplyMoveByCommand_CastlingKingSide_UCI(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.King, ImgFile: "pic/chess_pic/king_light.png", File: 5, Rank: 1},
		{Color: pieces.White, Kind: pieces.Rook, ImgFile: "pic/chess_pic/rook_light.png", File: 8, Rank: 1},
	}
	moveHistory = nil
	lastAppliedMove = nil
	resetCastlingState()

	normalized, err := ApplyMoveByCommand("e1g1")
	if err != nil {
		t.Fatalf("expected castling e1g1 to succeed, got error: %v", err)
	}
	if normalized != "e1g1" {
		t.Fatalf("expected normalized move e1g1, got %q", normalized)
	}

	king, ok := pieceAt(7, 1)
	if !ok || king.Kind != pieces.King || king.Color != pieces.White {
		t.Fatalf("expected white king on g1 after castling")
	}
	rook, ok := pieceAt(6, 1)
	if !ok || rook.Kind != pieces.Rook || rook.Color != pieces.White {
		t.Fatalf("expected white rook on f1 after castling")
	}
}

func TestApplyMoveByCommand_CastlingQueenSide_SAN(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.King, ImgFile: "pic/chess_pic/king_light.png", File: 5, Rank: 1},
		{Color: pieces.White, Kind: pieces.Rook, ImgFile: "pic/chess_pic/rook_light.png", File: 1, Rank: 1},
	}
	moveHistory = nil
	lastAppliedMove = nil
	resetCastlingState()

	normalized, err := ApplyMoveByCommand("O-O-O")
	if err != nil {
		t.Fatalf("expected SAN castling O-O-O to succeed, got error: %v", err)
	}
	if normalized != "e1c1" {
		t.Fatalf("expected normalized move e1c1, got %q", normalized)
	}

	king, ok := pieceAt(3, 1)
	if !ok || king.Kind != pieces.King || king.Color != pieces.White {
		t.Fatalf("expected white king on c1 after castling")
	}
	rook, ok := pieceAt(4, 1)
	if !ok || rook.Kind != pieces.Rook || rook.Color != pieces.White {
		t.Fatalf("expected white rook on d1 after castling")
	}
}

func TestApplyMoveByCommand_RejectsMoveLeavingOwnKingInCheck(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.King, ImgFile: "pic/chess_pic/king_light.png", File: 5, Rank: 1}, // Ke1
		{Color: pieces.White, Kind: pieces.Rook, ImgFile: "pic/chess_pic/rook_light.png", File: 5, Rank: 2}, // Re2
		{Color: pieces.Black, Kind: pieces.Rook, ImgFile: "pic/chess_pic/rook_dark.png", File: 5, Rank: 8},  // re8
		{Color: pieces.Black, Kind: pieces.King, ImgFile: "pic/chess_pic/king_dark.png", File: 1, Rank: 8},  // ka8
	}
	moveHistory = nil
	lastAppliedMove = nil

	if _, err := ApplyMoveByCommand("e2f2"); err == nil {
		t.Fatalf("expected move that exposes own king to be rejected")
	}
}

func TestApplyMoveByCommand_CheckedSideLabel_AfterCheckingMove(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.King, ImgFile: "pic/chess_pic/king_light.png", File: 1, Rank: 1}, // Ka1
		{Color: pieces.White, Kind: pieces.Rook, ImgFile: "pic/chess_pic/rook_light.png", File: 5, Rank: 1}, // Re1
		{Color: pieces.Black, Kind: pieces.King, ImgFile: "pic/chess_pic/king_dark.png", File: 5, Rank: 8},  // ke8
	}
	moveHistory = nil
	lastAppliedMove = nil

	if _, err := ApplyMoveByCommand("e1e7"); err != nil {
		t.Fatalf("expected checking move e1e7 to succeed, got error: %v", err)
	}
	if got := CheckedSideLabel(); got != "black" {
		t.Fatalf("expected checked side to be black, got %q", got)
	}
}

func TestApplyMoveByCommand_CastlingThroughAttackedSquareRejected(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.King, ImgFile: "pic/chess_pic/king_light.png", File: 5, Rank: 1}, // Ke1
		{Color: pieces.White, Kind: pieces.Rook, ImgFile: "pic/chess_pic/rook_light.png", File: 8, Rank: 1}, // Rh1
		{Color: pieces.Black, Kind: pieces.Rook, ImgFile: "pic/chess_pic/rook_dark.png", File: 6, Rank: 8},  // rf8 attacks f1
		{Color: pieces.Black, Kind: pieces.King, ImgFile: "pic/chess_pic/king_dark.png", File: 1, Rank: 8},  // ka8
	}
	moveHistory = nil
	lastAppliedMove = nil
	resetCastlingState()

	if _, err := ApplyMoveByCommand("e1g1"); err == nil {
		t.Fatalf("expected castling through attacked square to be rejected")
	}
}

// CM3070 FP code
// chess_end.go - chess ply-end registry and outcome context

package session

import (
	"go_backend/game/engine"
	pieces "go_backend/game/piece"
)

// chessGameEndStrategies - ordered Chess ply-end strategies; first ended wins
func chessGameEndStrategies() []GameEndStrategy {
	return []GameEndStrategy{
		ChessMissingKingStrategy{},
		ChessMateOrStalemateStrategy{},
		ChessInsufficientMaterialStrategy{},
		ChessThreefoldStrategy{},
		ChessFiftyMoveStrategy{},
	}
}

// ensureChessGameEndStrategies - registers the Chess ply-end chain
func ensureChessGameEndStrategies() {
	registerGameEndStrategies(GameTypeChess, chessGameEndStrategies()...)
}

// buildChessPlyEndContext - fills ply facts from the board plus draw clocks
func buildChessPlyEndContext() *plyEndContext {
	whiteKings, blackKings := 0, 0
	for _, p := range pieces.ChessPieces {
		if p.Kind != pieces.King {
			continue
		}
		if p.Color == pieces.White {
			whiteKings++
		} else {
			blackKings++
		}
	}
	side := CurrentTurnColor()
	legal := countLegalMoves(side)
	return &plyEndContext{
		GameType:             GameTypeChess,
		IdlePly:              GetHalfmoveClock(),
		PositionCount:       GetCurrentPositionRepetitionCount(),
		LegalMoves:           legal,
		InCheck:              engine.IsInCheck(side),
		SideToMove:           string(side),
		WhiteKings:           whiteKings,
		BlackKings:           blackKings,
		InsufficientMaterial: isInsufficientMaterialDraw(),
	}
}

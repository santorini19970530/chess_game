// CM3070 FP code
// session.go - implements game session rules

package session

import pieces "go_backend/game/piece"

// GameSession holds runtime game state metadata.

var initialPiecesSnapshot = append([]pieces.ChessPiece(nil), pieces.ChessPieces...)

// ResetGame resets board, turn state, move history, and session metadata.
func ResetGame() {
	game, err := lockActiveRuntimeState()
	if err == nil {
		defer unlockActiveRuntimeState(game)
	}
	resetGlobalsToInitialState()
}

func resetGlobalsToInitialState() {
	pieces.ChessPieces = append([]pieces.ChessPiece(nil), initialPiecesSnapshot...)
	moveHistory = nil
	moveHistoryDetailed = nil
	lastAppliedMove = nil
	boardFEN = ""
	resetShogiHands()
	resetCastlingState()
	resetTurnOverride()
	resetDrawTracking()
}

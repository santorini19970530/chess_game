// CM3070 FP code
// game_session.go - implements game session rules

package session

import (
	"fmt"
	"sync"
	"time"
)

type GameMode string

const (
	GameModeHumanVsHuman GameMode = "human_vs_human"
	GameModeHumanVsAI    GameMode = "human_vs_ai"
	GameModeAIVsAI       GameMode = "ai_vs_ai"
)

type GameType string

const (
	GameTypeChess   GameType = "chess"
	GameTypeXiangqi GameType = "xianqi"
	GameTypeShogi   GameType = "shogi"
)

type GameResult string

const (
	GameResultInProgress GameResult = "in_progress"
	GameResultWhiteWin   GameResult = "white_win"
	GameResultBlackWin   GameResult = "black_win"
	GameResultDraw       GameResult = "draw"
)

type GameSession struct {
	ID        string      `json:"id"`
	Mode      GameMode    `json:"mode"`
	Type      GameType    `json:"type"`
	Result    GameResult  `json:"result"`
	Outcome   GameOutcome `json:"outcome"`
	CreatedAt string      `json:"createdAt"`
	UpdatedAt string      `json:"updatedAt"`
}

var (
	gameSessionMu sync.RWMutex
	activeGame    = newGameSession(GameModeHumanVsHuman, GameTypeChess)
)

func newGameSession(mode GameMode, gameType GameType) GameSession {
	now := time.Now().UTC().Format(time.RFC3339)
	return GameSession{
		ID:        fmt.Sprintf("game-%d", time.Now().UnixNano()),
		Mode:      mode,
		Type:      gameType,
		Result:    GameResultInProgress,
		Outcome:   GameOutcome{Status: "in_progress"},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func GetGameSession() GameSession {
	gameSessionMu.RLock()
	defer gameSessionMu.RUnlock()
	return activeGame
}

func RefreshGameSessionOutcome() GameSession {
	outcome := EvaluateGameOutcome()

	gameSessionMu.Lock()
	defer gameSessionMu.Unlock()

	activeGame.Outcome = outcome
	activeGame.Result = gameResultFromOutcome(outcome)
	activeGame.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return activeGame
}

func CanAcceptMoves() bool {
	game := RefreshGameSessionOutcome()
	return game.Outcome.Status != "checkmate" && game.Outcome.Status != "stalemate"
}

func resetGameSessionForTest() {
	gameSessionMu.Lock()
	defer gameSessionMu.Unlock()
	activeGame = newGameSession(GameModeHumanVsHuman, GameTypeChess)
}

func gameResultFromOutcome(outcome GameOutcome) GameResult {
	switch outcome.Status {
	case "checkmate":
		if outcome.Winner == "white" {
			return GameResultWhiteWin
		}
		if outcome.Winner == "black" {
			return GameResultBlackWin
		}
		return GameResultInProgress
	case "stalemate":
		return GameResultDraw
	default:
		return GameResultInProgress
	}
}

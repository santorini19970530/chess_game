// CM3070 FP code
// game_session.go - implements game session rules

package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Archived  bool        `json:"-"`
}

type ArchivedSession struct {
	ID        string     `json:"id"`
	Mode      GameMode   `json:"mode"`
	Type      GameType   `json:"type"`
	Result    GameResult `json:"result"`
	CreatedAt string     `json:"createdAt"`
	UpdatedAt string     `json:"updatedAt"`
}

type ArchivedPieceState struct {
	Color string `json:"color"`
	Kind  string `json:"kind"`
	File  int    `json:"file"`
	Rank  int    `json:"rank"`
}

type ArchivedGame struct {
	Game       ArchivedSession      `json:"game"`
	History    []string             `json:"history"`
	State      []ArchivedPieceState `json:"state"`
	Captured   CapturedSummary      `json:"captured"`
	ArchivedAt string               `json:"archivedAt"`
}

var (
	gameSessionMu sync.RWMutex
	activeGame    = newGameSession(GameModeHumanVsHuman, GameTypeChess)
	archivePath   = filepath.Join("data", "game_history.json")
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
		Archived:  false,
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

func ArchiveActiveGameIfNeeded() error {
	gameSessionMu.Lock()
	if activeGame.Archived {
		gameSessionMu.Unlock()
		return nil
	}
	gameSnapshot := activeGame
	if gameSnapshot.Result == GameResultInProgress && len(GetMoveHistory()) == 0 {
		gameSessionMu.Unlock()
		return nil
	}
	history := GetMoveHistory()
	state := GetBoardState()
	captured := GetCapturedSummary()
	gameSessionMu.Unlock()

	records, err := loadArchivedGames()
	if err != nil {
		return err
	}
	records = append(records, ArchivedGame{
		Game: ArchivedSession{
			ID:        gameSnapshot.ID,
			Mode:      gameSnapshot.Mode,
			Type:      gameSnapshot.Type,
			Result:    gameSnapshot.Result,
			CreatedAt: gameSnapshot.CreatedAt,
			UpdatedAt: gameSnapshot.UpdatedAt,
		},
		History:    history,
		State:      toArchivedPieceState(state),
		Captured:   captured,
		ArchivedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err := saveArchivedGames(records); err != nil {
		return err
	}

	gameSessionMu.Lock()
	activeGame.Archived = true
	gameSessionMu.Unlock()
	return nil
}

func loadArchivedGames() ([]ArchivedGame, error) {
	bytes, err := os.ReadFile(archivePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ArchivedGame{}, nil
		}
		return nil, err
	}
	var records []ArchivedGame
	if len(bytes) == 0 {
		return []ArchivedGame{}, nil
	}
	if err := json.Unmarshal(bytes, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func saveArchivedGames(records []ArchivedGame) error {
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(archivePath, bytes, 0o644)
}

func toArchivedPieceState(state []PieceState) []ArchivedPieceState {
	out := make([]ArchivedPieceState, 0, len(state))
	for _, p := range state {
		out = append(out, ArchivedPieceState{
			Color: p.Color,
			Kind:  p.Kind,
			File:  p.File,
			Rank:  p.Rank,
		})
	}
	return out
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

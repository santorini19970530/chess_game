// CM3070 FP code
// game_session.go - implements game session rules

package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Config    GameConfig  `json:"config"`
	Result    GameResult  `json:"result"`
	Outcome   GameOutcome `json:"outcome"`
	CreatedAt string      `json:"createdAt"`
	UpdatedAt string      `json:"updatedAt"`
	Archived  bool        `json:"-"`
}

type GameConfig struct {
	HumanColor     string `json:"humanColor"`
	AIGameCount    int    `json:"aiGameCount"`
	StartFEN       string `json:"startFen"`
	AIProfile      string `json:"aiProfile"`
	WhiteAIProfile string `json:"whiteAIProfile,omitempty"`
	BlackAIProfile string `json:"blackAIProfile,omitempty"`
}

type ArchivedSession struct {
	ID        string     `json:"id"`
	Mode      GameMode   `json:"mode"`
	Type      GameType   `json:"type"`
	Config    GameConfig `json:"config"`
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
	runtimeStateMu sync.Mutex
	sessionStore  = NewSessionStore()
	activeGameID  string
	archivePath   string // resolved at init time to an absolute path under the executable dir (or fallback)
)

func init() {
	initializeSessionStore()
	archivePath = resolveArchivePath()
}

func resolveArchivePath() string {
	// Prefer directory next to the running binary so "go run ." and built binary behave the same.
	if execPath, err := os.Executable(); err == nil {
		if execPath != "" && execPath != "." {
			base := filepath.Dir(execPath)
			return filepath.Join(base, "data", "game_history.json")
		}
	}
	// Fallback: user cache dir (cross-platform, no cwd pollution).
	if cacheDir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(cacheDir, "chess_game", "data", "game_history.json")
	}
	// Last resort: cwd/data (still better than letting handlers/ subdir appear).
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Join(cwd, "data", "game_history.json")
	}
	return filepath.Join("data", "game_history.json")
}

// newUniqueGameID produces a unique ID with nanosecond timestamp + 4 random hex bytes.
// Prevents collisions even if multiple games are created in the same nanosecond.
func newUniqueGameID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// extremely rare; fall back to just nano
		return fmt.Sprintf("game-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("game-%d-%s", time.Now().UnixNano(), hex.EncodeToString(b))
}

// normalizeAIProfile returns a known profile or defaults to "intermediate".
// Allowed values: beginner, intermediate, advanced, master.
func normalizeAIProfile(p string) string {
	if parsed, ok := ParseAIProfile(p); ok {
		return parsed
	}
	return "intermediate"
}

// ParseAIProfile accepts a known profile name. Empty is not ok (caller chooses default).
func ParseAIProfile(p string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "beginner", "intermediate", "advanced", "master":
		return strings.ToLower(strings.TrimSpace(p)), true
	default:
		return "", false
	}
}

// ProfileForSide returns the strength for the side to move.
// Prefers WhiteAIProfile / BlackAIProfile, then AIProfile, then intermediate.
func ProfileForSide(cfg GameConfig, color string) string {
	side := strings.ToLower(strings.TrimSpace(color))
	switch side {
	case "white", "w":
		if cfg.WhiteAIProfile != "" {
			return normalizeAIProfile(cfg.WhiteAIProfile)
		}
	case "black", "b":
		if cfg.BlackAIProfile != "" {
			return normalizeAIProfile(cfg.BlackAIProfile)
		}
	}
	return normalizeAIProfile(cfg.AIProfile)
}

func profilesFromSingle(aiProfile string) (profile, white, black string) {
	profile = normalizeAIProfile(aiProfile)
	return profile, profile, profile
}

func newGameSession(mode GameMode, gameType GameType) GameSession {
	now := time.Now().UTC().Format(time.RFC3339)
	return GameSession{
		ID:   newUniqueGameID(),
		Mode: mode,
		Type: gameType,
		Config: GameConfig{
			HumanColor:     "white",
			AIGameCount:    1,
			StartFEN:       "",
			AIProfile:      "intermediate",
			WhiteAIProfile: "intermediate",
			BlackAIProfile: "intermediate",
		},
		Result:    GameResultInProgress,
		Outcome:   GameOutcome{Status: "in_progress"},
		CreatedAt: now,
		UpdatedAt: now,
		Archived:  false,
	}
}

func GetGameSession() GameSession {
	gameSessionMu.RLock()
	activeID := activeGameID
	gameSessionMu.RUnlock()
	game, ok := sessionStore.Get(activeID)
	if !ok {
		return GameSession{}
	}
	return game.Session
}

func RefreshGameSessionOutcome() GameSession {
	game, err := lockActiveRuntimeState()
	if err != nil {
		return GameSession{}
	}
	defer unlockActiveRuntimeState(game)

	if game.Session.Outcome.Status == "resigned" && game.Session.Result != GameResultInProgress {
		return game.Session
	}

	outcome := EvaluateGameOutcome()
	game.Session.Outcome = outcome
	game.Session.Result = gameResultFromOutcome(outcome)
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return game.Session
}

func CanAcceptMoves() bool {
	game := RefreshGameSessionOutcome()
	return game.Outcome.Status != "checkmate" && game.Outcome.Status != "stalemate"
}

func resetGameSessionForTest() {
	gameSessionMu.Lock()
	defer gameSessionMu.Unlock()
	sessionStore = NewSessionStore()
	initial := sessionStore.Create(newGameSession(GameModeHumanVsHuman, GameTypeChess))
	activeGameID = initial.Session.ID
	resetTurnOverride()
	initial.syncFromGlobals()
}

func ArchiveActiveGameIfNeeded() error {
	game, err := lockActiveRuntimeState()
	if err != nil {
		return err
	}
	if game.Session.Archived {
		unlockActiveRuntimeState(game)
		return nil
	}
	gameSnapshot := game.Session
	if gameSnapshot.Result == GameResultInProgress && len(GetMoveHistory()) == 0 {
		unlockActiveRuntimeState(game)
		return nil
	}
	history := GetMoveHistory()
	if flagEntry := archiveFlagEntry(gameSnapshot); flagEntry != "" {
		history = append(history, flagEntry)
	}
	state := GetBoardState()
	captured := GetCapturedSummary()
	unlockActiveRuntimeState(game)

	records, err := loadArchivedGames()
	if err != nil {
		return err
	}
	records = append(records, ArchivedGame{
		Game: ArchivedSession{
			ID:        gameSnapshot.ID,
			Mode:      gameSnapshot.Mode,
			Type:      gameSnapshot.Type,
			Config:    gameSnapshot.Config,
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

	game, err = lockActiveRuntimeState()
	if err != nil {
		return err
	}
	game.Session.Archived = true
	unlockActiveRuntimeState(game)
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

func UpdateGameConfig(mode GameMode, gameType GameType, humanColor string, aiGameCount int, startFEN string) (GameSession, error) {
	normalizedCount, err := validateGameConfig(mode, gameType, humanColor, aiGameCount, startFEN)
	if err != nil {
		return GameSession{}, err
	}

	game, err := lockActiveRuntimeState()
	if err != nil {
		return GameSession{}, err
	}
	defer unlockActiveRuntimeState(game)
	game.Session.Mode = mode
	game.Session.Type = gameType
	game.Session.Config = GameConfig{
		HumanColor:  humanColor,
		AIGameCount: normalizedCount,
		StartFEN:    startFEN,
	}
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return game.Session, nil
}

func StartConfiguredNewGame() (GameSession, error) {
	currentGame, err := lockActiveRuntimeState()
	if err != nil {
		return GameSession{}, err
	}
	currentMode := currentGame.Session.Mode
	currentType := currentGame.Session.Type
	currentConfig := currentGame.Session.Config
	unlockActiveRuntimeState(currentGame)

	newSession := newGameSession(currentMode, currentType)
	newSession.Config = currentConfig
	created := sessionStore.Create(newSession)

	gameSessionMu.Lock()
	activeGameID = created.Session.ID
	gameSessionMu.Unlock()

	game, err := lockActiveRuntimeState()
	if err != nil {
		return GameSession{}, err
	}
	defer unlockActiveRuntimeState(game)

	resetGlobalsToInitialState()
	if currentConfig.StartFEN != "" {
		if err := applyFENToCurrentGlobals(currentConfig.StartFEN); err != nil {
			return GameSession{}, err
		}
	}
	game.Session.Outcome = EvaluateGameOutcome()
	game.Session.Result = gameResultFromOutcome(game.Session.Outcome)
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return game.Session, nil
}

func piecesReset() {
	ResetGame()
}

func FlagCurrentTurn() GameSession {
	game, err := lockActiveRuntimeState()
	if err != nil {
		return GameSession{}
	}
	defer unlockActiveRuntimeState(game)
	side := CurrentTurnColor()
	winner := opponentOf(side)

	game.Session.Outcome = GameOutcome{
		Status:     "resigned",
		Winner:     string(winner),
		Loser:      string(side),
		LegalMoves: 0,
		Message:    sideLabel(side) + " flagged. " + sideLabel(winner) + " wins.",
	}
	if winner == "white" {
		game.Session.Result = GameResultWhiteWin
	} else {
		game.Session.Result = GameResultBlackWin
	}
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	game.Session.Archived = false
	return game.Session
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

func archiveFlagEntry(game GameSession) string {
	if game.Outcome.Status != "resigned" || game.Outcome.Loser == "" {
		return ""
	}
	return fmt.Sprintf("%s: flag", sideLabelFromText(game.Outcome.Loser))
}

func sideLabelFromText(side string) string {
	if side == "black" {
		return "Black"
	}
	return "White"
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
	case "draw_insufficient_material":
		return GameResultDraw
	case "draw_threefold_repetition":
		return GameResultDraw
	case "draw_fifty_move_rule":
		return GameResultDraw
	default:
		return GameResultInProgress
	}
}

func initializeSessionStore() {
	gameSessionMu.Lock()
	defer gameSessionMu.Unlock()
	sessionStore = NewSessionStore()
	initial := sessionStore.Create(newGameSession(GameModeHumanVsHuman, GameTypeChess))
	activeGameID = initial.Session.ID
	initial.bindToGlobals()
}

func getActiveRuntimeGame() (*RuntimeGame, error) {
	gameSessionMu.RLock()
	activeID := activeGameID
	gameSessionMu.RUnlock()
	game, ok := sessionStore.Get(activeID)
	if !ok {
		return nil, fmt.Errorf("active game session not found: %s", activeID)
	}
	return game, nil
}

func lockActiveRuntimeState() (*RuntimeGame, error) {
	runtimeStateMu.Lock()
	game, err := getActiveRuntimeGame()
	if err != nil {
		runtimeStateMu.Unlock()
		return nil, err
	}
	return game, nil
}

func unlockActiveRuntimeState(game *RuntimeGame) {
	if game != nil {
		game.syncFromGlobals()
	}
	runtimeStateMu.Unlock()
}

func getRuntimeGameByID(gameID string) (*RuntimeGame, error) {
	game, ok := sessionStore.Get(gameID)
	if !ok {
		return nil, fmt.Errorf("game session not found: %s", gameID)
	}
	return game, nil
}

func lockRuntimeStateByID(gameID string) (*RuntimeGame, error) {
	runtimeStateMu.Lock()
	game, err := getRuntimeGameByID(gameID)
	if err != nil {
		runtimeStateMu.Unlock()
		return nil, err
	}
	game.bindToGlobals()
	return game, nil
}

func unlockRuntimeStateByID(game *RuntimeGame) {
	if game != nil {
		game.syncFromGlobals()
	}
	runtimeStateMu.Unlock()
}

func ActivateGame(gameID string) error {
	if _, err := getRuntimeGameByID(gameID); err != nil {
		return err
	}
	gameSessionMu.Lock()
	activeGameID = gameID
	gameSessionMu.Unlock()
	return nil
}

func validateGameConfig(mode GameMode, gameType GameType, humanColor string, aiGameCount int, startFEN string) (int, error) {
	if mode != GameModeHumanVsHuman && mode != GameModeHumanVsAI && mode != GameModeAIVsAI {
		return 0, fmt.Errorf("invalid game mode")
	}
	if gameType != GameTypeChess && gameType != GameTypeXiangqi && gameType != GameTypeShogi {
		return 0, fmt.Errorf("invalid game type")
	}
	if humanColor != "white" && humanColor != "black" {
		return 0, fmt.Errorf("human side must be white or black")
	}
	if aiGameCount < 1 {
		return 0, fmt.Errorf("ai game count must be at least 1")
	}
	if mode != GameModeAIVsAI {
		aiGameCount = 1
	}
	if startFEN != "" {
		aiGameCount = 1
	}
	if gameType != GameTypeChess {
		return 0, fmt.Errorf("only chess is currently supported")
	}
	return aiGameCount, nil
}

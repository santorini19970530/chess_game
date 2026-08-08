// CM3070 FP code
// game_session.go - implements game session rules

package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// game mode for the game
type GameMode string

// game modes for the game
const (
	GameModeHumanVsHuman GameMode = "human_vs_human"
	GameModeHumanVsAI    GameMode = "human_vs_ai"
	GameModeAIVsAI       GameMode = "ai_vs_ai"
)

// game type for the game
type GameType string

// game types for the game
const (
	GameTypeChess   GameType = "chess"
	GameTypeXiangqi GameType = "xianqi"
	GameTypeShogi   GameType = "shogi"
)

// DefaultXiangqiStartFEN - standard xiangqi start position (fs-compatible fen text)
const DefaultXiangqiStartFEN = "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"

// game result for the game
type GameResult string

// game results for the game
const (
	GameResultInProgress GameResult = "in_progress"
	GameResultWhiteWin   GameResult = "white_win"
	GameResultBlackWin   GameResult = "black_win"
	GameResultDraw       GameResult = "draw"
)

// game session for the game
type GameSession struct {
	ID        string      `json:"id"`
	Mode      GameMode    `json:"mode"`
	Type      GameType    `json:"type"`
	Config    GameConfig  `json:"config"`
	Clock     *Clock      `json:"clock"`
	Result    GameResult  `json:"result"`
	Outcome   GameOutcome `json:"outcome"`
	CreatedAt string      `json:"createdAt"`
	UpdatedAt string      `json:"updatedAt"`
	Archived  bool        `json:"-"`
}

// game config for the game
type GameConfig struct {
	HumanColor     string `json:"humanColor"`
	AIGameCount    int    `json:"aiGameCount"`
	StartFEN       string `json:"startFen"`
	AIProfile      string `json:"aiProfile"`
	WhiteAIProfile string `json:"whiteAIProfile,omitempty"`
	BlackAIProfile string `json:"blackAIProfile,omitempty"`
	// skillLevel is coach/explain register: beginner|intermediate|advanced.
	// independent of AIProfile (master AI still maps to advanced when unset).
	SkillLevel string `json:"skillLevel,omitempty"`
}

// archived session for the game
type ArchivedSession struct {
	ID        string     `json:"id"`
	Mode      GameMode   `json:"mode"`
	Type      GameType   `json:"type"`
	Config    GameConfig `json:"config"`
	Result    GameResult `json:"result"`
	CreatedAt string     `json:"createdAt"`
	UpdatedAt string     `json:"updatedAt"`
}

// archived piece state for the game
type ArchivedPieceState struct {
	Color string `json:"color"`
	Kind  string `json:"kind"`
	File  int    `json:"file"`
	Rank  int    `json:"rank"`
}

// archived game for the game
type ArchivedGame struct {
	Game       ArchivedSession      `json:"game"`
	History    []string             `json:"history"`
	State      []ArchivedPieceState `json:"state"`
	Captured   CapturedSummary      `json:"captured"`
	ArchivedAt string               `json:"archivedAt"`
}

// global variables for the game
var (
	gameSessionMu  sync.RWMutex
	runtimeStateMu sync.Mutex
	sessionStore   = NewSessionStore()
	activeGameID   string
)

// init - runs package initialization
func init() {
	initializeSessionStore()
}

// newUniqueGameID - builds a unique game id from nanosecond time plus random hex
func newUniqueGameID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// extremely rare; fall back to just nano
		return fmt.Sprintf("game-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("game-%d-%s", time.Now().UnixNano(), hex.EncodeToString(b))
}

// normalizeAIProfile - returns a known profile or defaults to "intermediate"
func normalizeAIProfile(p string) string {
	if parsed, ok := ParseAIProfile(p); ok {
		return parsed
	}
	return "intermediate"
}

// ParseAIProfile - accepts a known profile name
func ParseAIProfile(p string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "beginner", "intermediate", "advanced", "master":
		return strings.ToLower(strings.TrimSpace(p)), true
	default:
		return "", false
	}
}

// ProfileForSide - returns the strength for the side to move
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

// profilesFromSingle - performs profiles from single
func profilesFromSingle(aiProfile string) (profile, white, black string) {
	profile = normalizeAIProfile(aiProfile)
	return profile, profile, profile
}

// NormalizeSkillLevel - accepts beginner|intermediate|advanced
func NormalizeSkillLevel(level string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "beginner", "intermediate", "advanced":
		return strings.ToLower(strings.TrimSpace(level)), true
	default:
		return "", false
	}
}

// SkillLevelFromAIProfile - maps AI strength (4) → explain skill (3). master → advanced
func SkillLevelFromAIProfile(profile string) string {
	switch normalizeAIProfile(profile) {
	case "beginner", "intermediate", "advanced":
		return normalizeAIProfile(profile)
	case "master":
		return "advanced"
	default:
		return "intermediate"
	}
}

// ResolveSkillLevel - prefers an explicit coach level; otherwise derives from AI profile
func ResolveSkillLevel(explicit, aiProfile string) string {
	if level, ok := NormalizeSkillLevel(explicit); ok {
		return level
	}
	return SkillLevelFromAIProfile(aiProfile)
}

// newGameSession - creates game session
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
			SkillLevel:     "intermediate",
		},
		Clock:     NewClock(0, 0, 0), // disabled = unlimited (today's behavior)
		Result:    GameResultInProgress,
		Outcome:   GameOutcome{Status: "in_progress"},
		CreatedAt: now,
		UpdatedAt: now,
		Archived:  false,
	}
}

// GetGameSession - returns game session
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

// RefreshGameSessionOutcome - refreshes game session outcome
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

// CanAcceptMoves - reports whether accept moves is allowed
func CanAcceptMoves() bool {
	game := RefreshGameSessionOutcome()
	return game.Outcome.Status != "checkmate" && game.Outcome.Status != "stalemate"
}

// resetGameSessionForTest - resets game session for test
func resetGameSessionForTest() {
	gameSessionMu.Lock()
	defer gameSessionMu.Unlock()
	sessionStore = NewSessionStore()
	initial := sessionStore.Create(newGameSession(GameModeHumanVsHuman, GameTypeChess))
	activeGameID = initial.Session.ID
	resetTurnOverride()
	initial.syncFromGlobals()
}

// ArchiveActiveGameIfNeeded - returns archive active game if needed
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

// UpdateGameConfig - updates game config
func UpdateGameConfig(mode GameMode, gameType GameType, humanColor string, aiGameCount int, startFEN string) (GameSession, error) {
	normalizedCount, err := validateGameConfig(mode, gameType, humanColor, aiGameCount, startFEN)
	if err != nil {
		return GameSession{}, err
	}
	startFEN = normalizeStartFEN(gameType, startFEN)

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

// StartConfiguredNewGame - starts configured new game
func StartConfiguredNewGame() (GameSession, error) {
	currentGame, err := lockActiveRuntimeState()
	if err != nil {
		return GameSession{}, err
	}
	currentMode := currentGame.Session.Mode
	currentType := currentGame.Session.Type
	currentConfig := currentGame.Session.Config
	prevClock := currentGame.Session.Clock
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
	if err := materializeStartPosition(currentType, currentConfig.StartFEN); err != nil {
		return GameSession{}, err
	}
	outcome := evaluateOutcomeForGameType(currentType)
	game.Session.Outcome = outcome
	game.Session.Result = gameResultFromOutcome(outcome)
	// carry time control onto the new game; remaining resets to initial bases.
	if prevClock != nil && prevClock.Enabled {
		game.Session.Clock = NewClock(prevClock.WhiteInitialMs, prevClock.BlackInitialMs, prevClock.IncrementMs)
		game.Session.Clock.Start(string(CurrentTurnColor()), time.Now().UTC())
	}
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return game.Session, nil
}

// piecesReset - performs pieces reset
func piecesReset() {
	ResetGame()
}

// FlagCurrentTurn - flags current turn
func FlagCurrentTurn() GameSession {
	game, err := lockActiveRuntimeState()
	if err != nil {
		return GameSession{}
	}
	defer unlockActiveRuntimeState(game)
	applyFlagLossLocked(game, string(CurrentTurnColor()))
	return game.Session
}

// toArchivedPieceState - converts to archived piece state
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

// archiveFlagEntry - returns archive flag entry
func archiveFlagEntry(game GameSession) string {
	if game.Outcome.Status != "resigned" || game.Outcome.Loser == "" {
		return ""
	}
	return fmt.Sprintf("%s: flag", sideLabelFromText(game.Outcome.Loser))
}

// sideLabelFromText - returns side label from text
func sideLabelFromText(side string) string {
	if side == "black" {
		return "Black"
	}
	return "White"
}

// gameResultFromOutcome - performs game result from outcome
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

// initializeSessionStore - performs initialize session store
func initializeSessionStore() {
	gameSessionMu.Lock()
	defer gameSessionMu.Unlock()
	sessionStore = NewSessionStore()
	initial := sessionStore.Create(newGameSession(GameModeHumanVsHuman, GameTypeChess))
	activeGameID = initial.Session.ID
	initial.bindToGlobals()
}

// getActiveRuntimeGame - returns active runtime game
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

// lockActiveRuntimeState - performs lock active runtime state
func lockActiveRuntimeState() (*RuntimeGame, error) {
	runtimeStateMu.Lock()
	game, err := getActiveRuntimeGame()
	if err != nil {
		runtimeStateMu.Unlock()
		return nil, err
	}
	return game, nil
}

// unlockActiveRuntimeState - performs unlock active runtime state
func unlockActiveRuntimeState(game *RuntimeGame) {
	if game != nil {
		game.syncFromGlobals()
	}
	runtimeStateMu.Unlock()
}

// getRuntimeGameByID - returns runtime game by id
func getRuntimeGameByID(gameID string) (*RuntimeGame, error) {
	game, ok := sessionStore.Get(gameID)
	if !ok {
		return nil, fmt.Errorf("game session not found: %s", gameID)
	}
	return game, nil
}

// lockRuntimeStateByID - lock runtime state by id
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

// unlockRuntimeStateByID - unlock runtime state by id
func unlockRuntimeStateByID(game *RuntimeGame) {
	if game != nil {
		game.syncFromGlobals()
	}
	runtimeStateMu.Unlock()
}

// ActivateGame - returns activate game
func ActivateGame(gameID string) error {
	if _, err := getRuntimeGameByID(gameID); err != nil {
		return err
	}
	gameSessionMu.Lock()
	activeGameID = gameID
	gameSessionMu.Unlock()
	return nil
}

// validateGameConfig - validates game config
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
	switch gameType {
	case GameTypeChess:
		// ok
	case GameTypeXiangqi:
		if startFEN != "" && !looksLikeXiangqiFEN(startFEN) {
			return 0, fmt.Errorf("chess FEN is not valid for xianqi")
		}
	case GameTypeShogi:
		if startFEN != "" && !looksLikeShogiFEN(startFEN) {
			return 0, fmt.Errorf("start FEN is not valid for shogi")
		}
	default:
		return 0, fmt.Errorf("unsupported game type")
	}
	return aiGameCount, nil
}

// looksLikeXiangqiFEN - xiangqi boards have 10 ranks (9 '/' separators in the placement field)
func looksLikeXiangqiFEN(fen string) bool {
	parts := strings.Fields(strings.TrimSpace(fen))
	if len(parts) == 0 {
		return false
	}
	return strings.Count(parts[0], "/") == 9
}

// normalizeStartFEN - normalizes start fen
func normalizeStartFEN(gameType GameType, startFEN string) string {
	startFEN = strings.TrimSpace(startFEN)
	if startFEN != "" {
		return startFEN
	}
	switch gameType {
	case GameTypeXiangqi:
		return DefaultXiangqiStartFEN
	case GameTypeShogi:
		return DefaultShogiStartFEN
	default:
		return startFEN
	}
}

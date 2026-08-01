package session

import (
	"fmt"
	"strings"
	"time"

	"go_backend/game/movement"
	pieces "go_backend/game/piece"
)

// game snapshot for the game
type GameSnapshot struct {
	CurrentTurn     string
	CheckedSide     string
	Game            GameSession
	Captured        CapturedSummary
	History         []string
	HistoryDetailed []MoveHistoryEntry
	State           []PieceState
}

// CreateGame - creates a stored game session from the given config
func CreateGame(mode GameMode, gameType GameType, humanColor string, aiGameCount int, startFEN string, aiProfile string) (GameSession, error) {
	normalizedCount, err := validateGameConfig(mode, gameType, humanColor, aiGameCount, startFEN)
	if err != nil {
		return GameSession{}, err
	}
	startFEN = normalizeStartFEN(gameType, startFEN)
	profile, white, black := profilesFromSingle(aiProfile)
	session := newGameSession(mode, gameType)
	session.Config = GameConfig{
		HumanColor:     humanColor,
		AIGameCount:    normalizedCount,
		StartFEN:       startFEN,
		AIProfile:      profile,
		WhiteAIProfile: white,
		BlackAIProfile: black,
		SkillLevel:     ResolveSkillLevel("", profile),
	}
	session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	game := sessionStore.Create(session)
	gameSessionMu.Lock()
	activeGameID = game.Session.ID
	gameSessionMu.Unlock()

	locked, err := lockRuntimeStateByID(game.Session.ID)
	if err != nil {
		return GameSession{}, err
	}
	defer unlockRuntimeStateByID(locked)
	resetGlobalsToInitialState()
	if err := materializeStartPosition(gameType, startFEN); err != nil {
		return GameSession{}, err
	}
	outcome := evaluateOutcomeForGameType(gameType)
	game.Session.Outcome = outcome
	game.Session.Result = gameResultFromOutcome(outcome)
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return game.Session, nil
}

// materializeStartPosition - loads board state for the given game type
func materializeStartPosition(gameType GameType, startFEN string) error {
	switch gameType {
	case GameTypeChess:
		if startFEN != "" {
			return applyFENToCurrentGlobals(startFEN)
		}
		return nil
	case GameTypeXiangqi:
		fen := startFEN
		if fen == "" {
			fen = DefaultXiangqiStartFEN
		}
		return applyXiangqiFENToCurrentGlobals(fen)
	case GameTypeShogi:
		fen := startFEN
		if fen == "" {
			fen = DefaultShogiStartFEN
		}
		return applyShogiFENToCurrentGlobals(fen)
	default:
		return fmt.Errorf("unsupported game type")
	}
}

// GetGameSessionByID - loads a game session by id
func GetGameSessionByID(gameID string) (GameSession, error) {
	game, err := getRuntimeGameByID(gameID)
	if err != nil {
		return GameSession{}, err
	}
	return game.Session, nil
}

// UpdateGameConfigByID - updates game config by id
func UpdateGameConfigByID(gameID string, mode GameMode, gameType GameType, humanColor string, aiGameCount int, startFEN string, aiProfile string) (GameSession, error) {
	normalizedCount, err := validateGameConfig(mode, gameType, humanColor, aiGameCount, startFEN)
	if err != nil {
		return GameSession{}, err
	}
	startFEN = normalizeStartFEN(gameType, startFEN)
	profile, white, black := profilesFromSingle(aiProfile)
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return GameSession{}, err
	}
	defer unlockRuntimeStateByID(game)
	prevSkill := game.Session.Config.SkillLevel
	game.Session.Mode = mode
	game.Session.Type = gameType
	game.Session.Config = GameConfig{
		HumanColor:     humanColor,
		AIGameCount:    normalizedCount,
		StartFEN:       startFEN,
		AIProfile:      profile,
		WhiteAIProfile: white,
		BlackAIProfile: black,
		// keep coach level unless unset (then derive from AI profile).
		SkillLevel: ResolveSkillLevel(prevSkill, profile),
	}
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return game.Session, nil
}

// SetClockByID - configures the session clock (Fischer). bases 0/0 disables (unlimited). when enabled, starts the clock on the side to move
func SetClockByID(gameID string, whiteInitialMs, blackInitialMs, incrementMs int64) (GameSession, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return GameSession{}, err
	}
	defer unlockRuntimeStateByID(game)
	clk := NewClock(whiteInitialMs, blackInitialMs, incrementMs)
	if clk.Enabled {
		clk.Start(string(CurrentTurnColor()), time.Now().UTC())
	}
	game.Session.Clock = clk
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return game.Session, nil
}

// SetSkillLevelByID - sets the coach/explain skill level for a game session
func SetSkillLevelByID(gameID, skillLevel string) (GameSession, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return GameSession{}, err
	}
	defer unlockRuntimeStateByID(game)
	game.Session.Config.SkillLevel = ResolveSkillLevel(skillLevel, game.Session.Config.AIProfile)
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return game.Session, nil
}

// SetAISideProfilesByID - sets white/black ai strengths for ai-vs-ai; mirrors both into AIProfile when equal
func SetAISideProfilesByID(gameID, whiteProfile, blackProfile string) (GameSession, error) {
	white, okW := ParseAIProfile(whiteProfile)
	if !okW {
		return GameSession{}, fmt.Errorf("invalid white_profile %q", whiteProfile)
	}
	black, okB := ParseAIProfile(blackProfile)
	if !okB {
		return GameSession{}, fmt.Errorf("invalid black_profile %q", blackProfile)
	}
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return GameSession{}, err
	}
	defer unlockRuntimeStateByID(game)
	game.Session.Config.WhiteAIProfile = white
	game.Session.Config.BlackAIProfile = black
	game.Session.Config.AIProfile = white
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return game.Session, nil
}

// RefreshGameSessionOutcomeByID - refreshes game session outcome by id
func RefreshGameSessionOutcomeByID(gameID string) (GameSession, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return GameSession{}, err
	}
	defer unlockRuntimeStateByID(game)
	if game.Session.Outcome.Status == "resigned" && game.Session.Result != GameResultInProgress {
		return game.Session, nil
	}
	outcome := evaluateOutcomeForGameType(game.Session.Type)
	game.Session.Outcome = outcome
	game.Session.Result = gameResultFromOutcome(outcome)
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return game.Session, nil
}

// ApplyMoveByCommandByID - applies a move command to the named session
func ApplyMoveByCommandByID(gameID, commandText string) (string, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return "", err
	}
	defer unlockRuntimeStateByID(game)
	if err := rejectIfGameOverLocked(game); err != nil {
		return "", err
	}
	now := time.Now().UTC()
	mover := string(CurrentTurnColor())
	if err := settleClockOrFlagLocked(game, now); err != nil {
		return "", err
	}

	var normalized string
	switch game.Session.Type {
	case GameTypeXiangqi:
		normalized, err = applyXiangqiUCIMove(commandText)
		if err != nil {
			return "", err
		}
		outcome := EvaluateXiangqiGameOutcome()
		game.Session.Outcome = outcome
		game.Session.Result = gameResultFromOutcome(outcome)
	case GameTypeShogi:
		normalized, err = applyShogiUCIMove(commandText)
		if err != nil {
			return "", err
		}
		outcome := evaluateOutcomeForGameType(GameTypeShogi)
		game.Session.Outcome = outcome
		game.Session.Result = gameResultFromOutcome(outcome)
	default:
		normalized, err = applyMoveByCommandCurrentLoaded(commandText)
		if err != nil {
			return "", err
		}
	}
	awardClockAfterMoveLocked(game, mover, now)
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return normalized, nil
}

// FlagCurrentTurnByID - flags the side to move on the named session
func FlagCurrentTurnByID(gameID string) (GameSession, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return GameSession{}, err
	}
	defer unlockRuntimeStateByID(game)
	applyFlagLossLocked(game, string(CurrentTurnColor()))
	return game.Session, nil
}

// rejectIfGameOverLocked - rejects if game over locked
func rejectIfGameOverLocked(game *RuntimeGame) error {
	if game == nil {
		return fmt.Errorf("game session not found")
	}
	if game.Session.Result != GameResultInProgress {
		msg := game.Session.Outcome.Message
		if msg == "" {
			msg = "game already ended"
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// settleClockOrFlagLocked - deducts elapsed time; on flag, ends the game and returns an error
func settleClockOrFlagLocked(game *RuntimeGame, now time.Time) error {
	clk := game.Session.Clock
	if clk == nil || !clk.Enabled {
		return nil
	}
	clk.Settle(now)
	if side, ok := clk.Flagged(); ok {
		applyFlagLossLocked(game, side)
		return fmt.Errorf("%s", game.Session.Outcome.Message)
	}
	return nil
}

// awardClockAfterMoveLocked - awards clock after move locked
func awardClockAfterMoveLocked(game *RuntimeGame, mover string, now time.Time) {
	clk := game.Session.Clock
	if clk == nil || !clk.Enabled || game.Session.Result != GameResultInProgress {
		return
	}
	clk.OnMove(mover, now)
}

// applyFlagLossLocked - applies flag loss locked
func applyFlagLossLocked(game *RuntimeGame, loser string) {
	side := pieces.PieceColor(normalizeClockSide(loser))
	if side == "" {
		side = CurrentTurnColor()
	}
	winner := opponentOf(side)
	game.Session.Outcome = GameOutcome{
		Status:     "resigned",
		Winner:     string(winner),
		Loser:      string(side),
		LegalMoves: 0,
		Message:    sideLabel(side) + " flagged. " + sideLabel(winner) + " wins.",
	}
	if winner == pieces.White {
		game.Session.Result = GameResultWhiteWin
	} else {
		game.Session.Result = GameResultBlackWin
	}
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	game.Session.Archived = false
}

// ArchiveGameIfNeededByID - archive game if needed by id
func ArchiveGameIfNeededByID(gameID string) error {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return err
	}
	if game.Session.Archived {
		unlockRuntimeStateByID(game)
		return nil
	}
	gameSnapshot := game.Session
	if gameSnapshot.Result == GameResultInProgress && len(GetMoveHistory()) == 0 {
		unlockRuntimeStateByID(game)
		return nil
	}
	history := GetMoveHistory()
	if flagEntry := archiveFlagEntry(gameSnapshot); flagEntry != "" {
		history = append(history, flagEntry)
	}
	state := GetBoardState()
	captured := GetCapturedSummary()
	unlockRuntimeStateByID(game)

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

	game, err = lockRuntimeStateByID(gameID)
	if err != nil {
		return err
	}
	game.Session.Archived = true
	unlockRuntimeStateByID(game)
	return nil
}

// BuildSnapshotByID - builds snapshot by id
func BuildSnapshotByID(gameID string) (GameSnapshot, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return GameSnapshot{}, err
	}
	defer unlockRuntimeStateByID(game)
	syncClockLocked(game, time.Now().UTC())
	checked := CheckedSideLabel()
	captured := GetCapturedSummary()
	switch game.Session.Type {
	case GameTypeXiangqi:
		checked = string(movement.XiangqiCheckedColor())
		captured = GetXiangqiCapturedSummary()
	case GameTypeShogi:
		checked = string(movement.ShogiCheckedColor())
		captured = shogiHandsSummary() // captives live in hand (relife)
	}
	return GameSnapshot{
		CurrentTurn:     CurrentTurnLabel(),
		CheckedSide:     checked,
		Game:            game.Session,
		Captured:        captured,
		History:         GetMoveHistory(),
		HistoryDetailed: GetMoveHistoryDetailed(),
		State:           GetBoardState(),
	}, nil
}

// syncClockLocked - updates remaining for reads/snapshots; may end the game on flag
func syncClockLocked(game *RuntimeGame, now time.Time) {
	if game == nil || game.Session.Result != GameResultInProgress {
		return
	}
	clk := game.Session.Clock
	if clk == nil || !clk.Enabled {
		return
	}
	clk.Settle(now)
	if side, ok := clk.Flagged(); ok {
		applyFlagLossLocked(game, side)
	}
}

// AdjustClockLastTickByID - sets LastTick (tests / controlled settle scenarios)
func AdjustClockLastTickByID(gameID string, when time.Time) error {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return err
	}
	defer unlockRuntimeStateByID(game)
	if game.Session.Clock == nil {
		return fmt.Errorf("clock not configured")
	}
	game.Session.Clock.LastTickUnixMs = when.UnixMilli()
	return nil
}

// CurrentFENByID - current fen by id
func CurrentFENByID(gameID string) (string, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return "", err
	}
	defer unlockRuntimeStateByID(game)
	return CurrentFEN(), nil
}

// CurrentTurnColorByID - current turn color by id
func CurrentTurnColorByID(gameID string) (string, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return "", err
	}
	defer unlockRuntimeStateByID(game)
	return string(CurrentTurnColor()), nil
}

// MoveHistoryByID - move history by id
func MoveHistoryByID(gameID string) ([]string, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return nil, err
	}
	defer unlockRuntimeStateByID(game)
	return GetMoveHistory(), nil
}

// LastMoveIsCaptureByID - reports whether the latest history entry was a capture
func LastMoveIsCaptureByID(gameID string) (bool, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return false, err
	}
	defer unlockRuntimeStateByID(game)
	if len(moveHistoryDetailed) == 0 {
		return false, nil
	}
	return moveHistoryDetailed[len(moveHistoryDetailed)-1].IsCapture, nil
}

// LegalMovesForSquareByID - legal moves for square by id
func LegalMovesForSquareByID(gameID string, file, rank int) ([]LegalDestination, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return nil, err
	}
	defer unlockRuntimeStateByID(game)
	switch game.Session.Type {
	case GameTypeXiangqi:
		return xiangqiLegalDestinationsForSquare(file, rank)
	case GameTypeShogi:
		return shogiLegalDestinationsForSquare(file, rank)
	}
	return LegalMovesForSquare(file, rank), nil
}

// LegalDropsForKindByID - returns drop destinations for a hand piece (shogi only)
func LegalDropsForKindByID(gameID string, kind string) ([]LegalDestination, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return nil, err
	}
	defer unlockRuntimeStateByID(game)
	if game.Session.Type != GameTypeShogi {
		return nil, fmt.Errorf("drops only for shogi")
	}
	k := pieces.PieceKind(strings.ToLower(strings.TrimSpace(kind)))
	side := CurrentTurnColor()
	if shogiHandCount(side, k) <= 0 {
		return []LegalDestination{}, nil
	}
	return shogiLegalDropDestinations(k, side), nil
}

// AllLegalUCIMovesByID - returns every legal UCI move for the side to move. chess / Xiangqi / Shogi use Go movement strategies (engine is advice-only)
func AllLegalUCIMovesByID(gameID string) ([]string, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return nil, err
	}
	defer unlockRuntimeStateByID(game)
	if game.Session.Type == GameTypeXiangqi {
		return xiangqiAllLegalUCIMoves()
	}
	if game.Session.Type == GameTypeShogi {
		return shogiAllLegalUCIMoves()
	}
	side := string(CurrentTurnColor())
	state := GetBoardState()
	seen := make(map[string]struct{}, 128)
	for _, p := range state {
		if p.Color != side {
			continue
		}
		dests := LegalMovesForSquare(p.File, p.Rank)
		for _, d := range dests {
			uci := fmt.Sprintf("%c%d%c%d", byte('a'+p.File-1), p.Rank, byte('a'+d.File-1), d.Rank)
			if d.RequiresPromotion {
				uci += "q"
			}
			seen[uci] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for mv := range seen {
		out = append(out, mv)
	}
	return out, nil
}

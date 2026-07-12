package session

import (
	"fmt"
	"time"
)

type GameSnapshot struct {
	CurrentTurn     string
	CheckedSide     string
	Game            GameSession
	Captured        CapturedSummary
	History         []string
	HistoryDetailed []MoveHistoryEntry
	State           []PieceState
}

func CreateGame(mode GameMode, gameType GameType, humanColor string, aiGameCount int, startFEN string, aiProfile string) (GameSession, error) {
	normalizedCount, err := validateGameConfig(mode, gameType, humanColor, aiGameCount, startFEN)
	if err != nil {
		return GameSession{}, err
	}
	profile, white, black := profilesFromSingle(aiProfile)
	session := newGameSession(mode, gameType)
	session.Config = GameConfig{
		HumanColor:     humanColor,
		AIGameCount:    normalizedCount,
		StartFEN:       startFEN,
		AIProfile:      profile,
		WhiteAIProfile: white,
		BlackAIProfile: black,
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
	if startFEN != "" {
		if err := applyFENToCurrentGlobals(startFEN); err != nil {
			return GameSession{}, err
		}
	}
	game.Session.Outcome = EvaluateGameOutcome()
	game.Session.Result = gameResultFromOutcome(game.Session.Outcome)
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return game.Session, nil
}

func GetGameSessionByID(gameID string) (GameSession, error) {
	game, err := getRuntimeGameByID(gameID)
	if err != nil {
		return GameSession{}, err
	}
	return game.Session, nil
}

func UpdateGameConfigByID(gameID string, mode GameMode, gameType GameType, humanColor string, aiGameCount int, startFEN string, aiProfile string) (GameSession, error) {
	normalizedCount, err := validateGameConfig(mode, gameType, humanColor, aiGameCount, startFEN)
	if err != nil {
		return GameSession{}, err
	}
	profile, white, black := profilesFromSingle(aiProfile)
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return GameSession{}, err
	}
	defer unlockRuntimeStateByID(game)
	game.Session.Mode = mode
	game.Session.Type = gameType
	game.Session.Config = GameConfig{
		HumanColor:     humanColor,
		AIGameCount:    normalizedCount,
		StartFEN:       startFEN,
		AIProfile:      profile,
		WhiteAIProfile: white,
		BlackAIProfile: black,
	}
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return game.Session, nil
}

// SetAISideProfilesByID sets White/Black strengths for AI-vs-AI evaluation.
// When white==black, AIProfile is set to that value; otherwise AIProfile is left as the white value for compat.
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

func RefreshGameSessionOutcomeByID(gameID string) (GameSession, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return GameSession{}, err
	}
	defer unlockRuntimeStateByID(game)
	if game.Session.Outcome.Status == "resigned" && game.Session.Result != GameResultInProgress {
		return game.Session, nil
	}
	outcome := EvaluateGameOutcome()
	game.Session.Outcome = outcome
	game.Session.Result = gameResultFromOutcome(outcome)
	game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return game.Session, nil
}

func ApplyMoveByCommandByID(gameID, commandText string) (string, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return "", err
	}
	defer unlockRuntimeStateByID(game)
	return applyMoveByCommandCurrentLoaded(commandText)
}

func FlagCurrentTurnByID(gameID string) (GameSession, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return GameSession{}, err
	}
	defer unlockRuntimeStateByID(game)
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
	return game.Session, nil
}

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

func BuildSnapshotByID(gameID string) (GameSnapshot, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return GameSnapshot{}, err
	}
	defer unlockRuntimeStateByID(game)
	return GameSnapshot{
		CurrentTurn:     CurrentTurnLabel(),
		CheckedSide:     CheckedSideLabel(),
		Game:            game.Session,
		Captured:        GetCapturedSummary(),
		History:         GetMoveHistory(),
		HistoryDetailed: GetMoveHistoryDetailed(),
		State:           GetBoardState(),
	}, nil
}

func CurrentFENByID(gameID string) (string, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return "", err
	}
	defer unlockRuntimeStateByID(game)
	return CurrentFEN(), nil
}

func CurrentTurnColorByID(gameID string) (string, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return "", err
	}
	defer unlockRuntimeStateByID(game)
	return string(CurrentTurnColor()), nil
}

func MoveHistoryByID(gameID string) ([]string, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return nil, err
	}
	defer unlockRuntimeStateByID(game)
	return GetMoveHistory(), nil
}

func LegalMovesForSquareByID(gameID string, file, rank int) ([]LegalDestination, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return nil, err
	}
	defer unlockRuntimeStateByID(game)
	return LegalMovesForSquare(file, rank), nil
}

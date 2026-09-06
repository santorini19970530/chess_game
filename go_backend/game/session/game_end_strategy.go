// CM3070 FP code
// game_end_strategy.go - ply-end strategy interface and per-game registry

package session

// GameEndStrategy - decides whether the ply just applied ends the game
type GameEndStrategy interface {
	Name() string
	// AfterPly - returns ended plus the outcome when this ply is terminal
	AfterPly(ctx *plyEndContext) (ended bool, outcome GameOutcome)
}

// plyEndContext - facts from the ply just applied; strategies must not mutate the board
type plyEndContext struct {
	GameType                 GameType
	IdlePly                  int
	PositionKey              string
	PositionCount            int
	GaveCheck                bool
	IsCapture                bool
	LegalMoves               int
	InCheck                  bool
	SideJustMoved            string
	SideToMove               string
	PerpetualCheckLoser      string
	PerpetualChaseLoser      string
	ChaseIsKingOrSoldierOnly bool
	CycleRepeat              bool
	WhiteKings               int
	BlackKings               int
	InsufficientMaterial     bool
	PlyCount                 int
}

var gameEndByType = map[GameType][]GameEndStrategy{}

// registerGameEndStrategies - replaces the ordered end-strategy list for one game type
func registerGameEndStrategies(gameType GameType, strategies ...GameEndStrategy) {
	if gameEndByType == nil {
		gameEndByType = map[GameType][]GameEndStrategy{}
	}
	copied := make([]GameEndStrategy, len(strategies))
	copy(copied, strategies)
	gameEndByType[gameType] = copied
}

// runGameEndStrategies - runs registered strategies for ctx.GameType; first ended wins
func runGameEndStrategies(ctx *plyEndContext) (ended bool, outcome GameOutcome) {
	if ctx == nil {
		return false, GameOutcome{Status: "in_progress"}
	}
	for _, strategy := range gameEndByType[ctx.GameType] {
		ended, out := strategy.AfterPly(ctx)
		if ended {
			return true, out
		}
	}
	return false, GameOutcome{Status: "in_progress"}
}

// clearGameEndStrategiesForTest - drops all registered ply-end strategies
func clearGameEndStrategiesForTest() {
	gameEndByType = map[GameType][]GameEndStrategy{}
}

// CM3070 FP code
// game_end_max_ply.go - shared ply-limit ply-end strategy

package session

import "fmt"

// DefaultMaxPlies - house ply cap used as MaxPlyStrategy.Limit when unset
var DefaultMaxPlies = 600

// MaxPlyStrategy - ends the game when PlyCount reaches Limit (last in each registry)
type MaxPlyStrategy struct {
	Limit int
}

func (s MaxPlyStrategy) Name() string { return "max_ply" }

// AfterPly - ends as a draw when the ply count reaches the configured limit
func (s MaxPlyStrategy) AfterPly(ctx *plyEndContext) (bool, GameOutcome) {
	limit := s.Limit
	if limit <= 0 {
		limit = DefaultMaxPlies
	}
	if ctx == nil || ctx.PlyCount < limit {
		return false, GameOutcome{}
	}
	return true, GameOutcome{
		Status:     "draw_max_plies",
		LegalMoves: ctx.LegalMoves,
		Message:    fmt.Sprintf("Draw by ply limit (%d).", limit),
	}
}

// currentPlyCount - returns how many plies have been applied in this session
func currentPlyCount() int {
	return len(moveHistory)
}

// maxPlyStrategy - returns the shared last ply-end strategy using DefaultMaxPlies
func maxPlyStrategy() MaxPlyStrategy {
	return MaxPlyStrategy{Limit: DefaultMaxPlies}
}

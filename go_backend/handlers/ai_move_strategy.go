package handlers

import (
	"fmt"
	"log"
	"strings"

	sessionpkg "go_backend/game/session"
)

// AIMoveStrategy - proposes one uci candidate; go legality is enforced by the selector
type AIMoveStrategy interface {
	Name() string
	// Propose - returns a candidate move, or empty to try the next strategy; hardErr aborts SelectAIMove
	Propose(ctx *aiMoveContext) (uci string, hardErr error)
}

// aiMoveContext - shared inputs for ai play strategies
type aiMoveContext struct {
	GameID       string
	FEN          string
	Side         string
	GameType     string
	Profile      string
	AllowDegrade bool
	Clock        *sessionpkg.Clock
	History      []string
	LegalMoves   []string
	LegalByNorm  map[string]string
}

// pickGoLegal - maps a proposed uci to the go-canonical legal string when present
func (ctx *aiMoveContext) pickGoLegal(candidate string) (string, bool) {
	canon, ok := ctx.LegalByNorm[normalizeAIMove(ctx.GameType, candidate)]
	return canon, ok
}

// FairyStockfishPlayStrategy - fairy-stockfish proposal using the side strength profile
type FairyStockfishPlayStrategy struct{}

// Name - returns the ai move strategy name
func (FairyStockfishPlayStrategy) Name() string { return "fairy_stockfish_play" }

// Propose - asks fairy-stockfish for a move; hvai hard-fails on engine timeout
func (FairyStockfishPlayStrategy) Propose(ctx *aiMoveContext) (string, error) {
	move, err := selectMoveWithFairyStockfish(ctx.FEN, ctx.Profile, ctx.Side, ctx.AllowDegrade, ctx.GameType, ctx.Clock)
	if err == nil && strings.TrimSpace(move) != "" {
		return move, nil
	}
	if err != nil && !ctx.AllowDegrade {
		if _, ferr := sessionpkg.FlagCurrentTurnByID(ctx.GameID); ferr != nil {
			log.Printf("warning: failed to flag AI after FS timeout %s: %v", gameIDLabel(ctx.GameID), ferr)
		} else {
			log.Printf("human_vs_ai: AI thinking timeout/failure — AI flagged (%v)", err)
		}
		return "", fmt.Errorf("ai thinking timeout: %w", err)
	}
	if err != nil {
		log.Printf("warning: fairy-stockfish unavailable for %s: %v", gameIDLabel(ctx.GameID), err)
	}
	return "", nil
}

// HPVFallbackStrategy - chess-only python history/policy/value advice path
type HPVFallbackStrategy struct{}

// Name - returns the ai move strategy name
func (HPVFallbackStrategy) Name() string { return "hpv_fallback" }

// Propose - returns the first go-legal policy candidate, or empty when none
func (HPVFallbackStrategy) Propose(ctx *aiMoveContext) (string, error) {
	ai := NewAIClient()
	commonReq := AICommonRequest{
		RequestID:   fmt.Sprintf("%s-ai-%d", ctx.GameID, len(ctx.History)+1),
		GameID:      ctx.GameID,
		GameType:    ctx.GameType,
		Variant:     ctx.GameType,
		FEN:         ctx.FEN,
		Color:       strings.ToLower(ctx.Side),
		MoveNumber:  len(ctx.History) + 1,
		MoveHistory: ctx.History,
		Profile:     ctx.Profile,
	}
	if _, err := ai.History(commonReq); err != nil {
		log.Printf("warning: ai history unavailable for %s: %v", gameIDLabel(ctx.GameID), err)
	}
	if _, err := ai.Value(commonReq); err != nil {
		log.Printf("warning: ai value unavailable for %s: %v", gameIDLabel(ctx.GameID), err)
	}
	topK := len(ctx.LegalMoves)
	if topK < 5 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}
	policyResp, err := ai.Policy(AIPolicyRequest{
		AICommonRequest: commonReq,
		TopK:            topK,
	})
	if err != nil {
		log.Printf("warning: ai policy unavailable for %s: %v", gameIDLabel(ctx.GameID), err)
		return "", nil
	}
	if policyResp == nil {
		return "", nil
	}
	for _, c := range policyResp.Candidates {
		if _, ok := ctx.pickGoLegal(c.UCI); ok {
			return c.UCI, nil
		}
	}
	return "", nil
}

// FirstLegalFallbackStrategy - last resort: first sorted go-legal move
type FirstLegalFallbackStrategy struct{}

// Name - returns the ai move strategy name
func (FirstLegalFallbackStrategy) Name() string { return "first_legal_fallback" }

// Propose - returns the first sorted legal move (always go-legal)
func (FirstLegalFallbackStrategy) Propose(ctx *aiMoveContext) (string, error) {
	if len(ctx.LegalMoves) == 0 {
		return "", fmt.Errorf("no legal moves available")
	}
	if ctx.GameType != "chess" && !useFairyStockfish() {
		log.Printf("warning: %s AI without USE_FAIRY_STOCKFISH — Go first-legal fallback (weak)", ctx.GameType)
	}
	return ctx.LegalMoves[0], nil
}

// aiMoveStrategies - ordered play strategies: fs (optional) → hpv (chess) → first-legal
func aiMoveStrategies(ctx *aiMoveContext) []AIMoveStrategy {
	out := make([]AIMoveStrategy, 0, 3)
	if useFairyStockfish() {
		out = append(out, FairyStockfishPlayStrategy{})
	}
	if ctx.GameType == "chess" {
		out = append(out, HPVFallbackStrategy{})
	}
	out = append(out, FirstLegalFallbackStrategy{})
	return out
}

// runAIMoveStrategies - tries each strategy until a go-legal move is accepted
func runAIMoveStrategies(ctx *aiMoveContext) (string, error) {
	for _, strategy := range aiMoveStrategies(ctx) {
		move, hardErr := strategy.Propose(ctx)
		if hardErr != nil {
			return "", hardErr
		}
		if strings.TrimSpace(move) == "" {
			continue
		}
		canon, ok := ctx.pickGoLegal(move)
		if !ok {
			log.Printf("warning: %s move %q rejected (not in Go legal set) %s", strategy.Name(), move, gameIDLabel(ctx.GameID))
			continue
		}
		log.Printf("ai_move_strategy=%s game_id=%s", strategy.Name(), ctx.GameID)
		return canon, nil
	}
	return "", fmt.Errorf("no legal moves available")
}

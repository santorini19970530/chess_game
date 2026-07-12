package handlers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go_backend/game/engine"
	sessionpkg "go_backend/game/session"
	"go_backend/simulation"
)

var (
	fsEngines  = map[string]*engine.FairyStockfish{}
	fsEngineMu sync.Mutex
)

// useFairyStockfish reports whether the Go UCI path should be used (env flag).
func useFairyStockfish() bool {
	return strings.EqualFold(os.Getenv("USE_FAIRY_STOCKFISH"), "true") ||
		strings.EqualFold(os.Getenv("USE_FAIRY_STOCKFISH"), "1")
}

// fairyStockfishBinary resolves the path to the stockfish binary.
// Default: relative to this module's parent (py_analyser/...).
func fairyStockfishBinary() string {
	if p := os.Getenv("FAIRY_STOCKFISH_PATH"); p != "" {
		return p
	}
	// Fallback default used in the project layout
	return filepath.Join("..", "py_analyser", "Fairy-Stockfish-fairy_sf_14", "src", "stockfish")
}

// getFairyStockfish returns a started engine instance for one side (white|black).
func getFairyStockfish(side string) (*engine.FairyStockfish, error) {
	key := strings.ToLower(strings.TrimSpace(side))
	if key != "black" {
		key = "white"
	}

	fsEngineMu.Lock()
	defer fsEngineMu.Unlock()

	if fs, ok := fsEngines[key]; ok && fs != nil {
		return fs, nil
	}

	bin := fairyStockfishBinary()
	fs, err := engine.NewFairyStockfish(bin)
	if err != nil {
		return nil, err
	}
	if err := fs.Start(); err != nil {
		return nil, err
	}
	fsEngines[key] = fs
	return fs, nil
}

// SelectAIMove picks one legal UCI move for the given game using AI endpoints.
// Policy is the primary signal; history/value are auxiliary and non-blocking.
func SelectAIMove(gameID string) (string, error) {
	fen, err := sessionpkg.CurrentFENByID(gameID)
	if err != nil {
		return "", err
	}
	color, err := sessionpkg.CurrentTurnColorByID(gameID)
	if err != nil {
		return "", err
	}
	history, err := sessionpkg.MoveHistoryByID(gameID)
	if err != nil {
		return "", err
	}
	snapshot, err := sessionpkg.BuildSnapshotByID(gameID)
	if err != nil {
		return "", err
	}

	legalMoves, err := legalUCIMovesByID(gameID, snapshot.State, color)
	if err != nil {
		return "", err
	}
	if len(legalMoves) == 0 {
		return "", fmt.Errorf("no legal moves available")
	}
	sort.Strings(legalMoves)
	legalSet := make(map[string]struct{}, len(legalMoves))
	for _, mv := range legalMoves {
		legalSet[normalizeUCI(mv)] = struct{}{}
	}

	ai := NewAIClient()
	profile := sessionpkg.ProfileForSide(snapshot.Game.Config, color)
	gameType := string(snapshot.Game.Type)
	if gameType == "" {
		gameType = "chess"
	}

	// Optional Go-side Fairy-Stockfish path (env-gated, backward compatible)
	if useFairyStockfish() {
		if move, err := selectMoveWithFairyStockfish(fen, profile, color); err == nil && move != "" {
			if _, ok := legalSet[normalizeUCI(move)]; ok {
				return move, nil
			}
		} else if err != nil {
			log.Printf("warning: fairy-stockfish unavailable for %s: %v (falling back to python)", gameIDLabel(gameID), err)
		}
	}

	commonReq := AICommonRequest{
		RequestID:   fmt.Sprintf("%s-ai-%d", gameID, len(history)+1),
		GameID:      gameID,
		GameType:    gameType,
		Variant:     gameType,
		FEN:         fen,
		Color:       strings.ToLower(color),
		MoveNumber:  len(history) + 1,
		MoveHistory: history,
		Profile:     profile,
	}

	// Context is optional for correctness in this stage.
	if _, err := ai.History(commonReq); err != nil {
		log.Printf("warning: ai history unavailable for %s: %v", gameIDLabel(gameID), err)
	}
	// Value is optional in this stage; called for future tie-break usage.
	if _, err := ai.Value(commonReq); err != nil {
		log.Printf("warning: ai value unavailable for %s: %v", gameIDLabel(gameID), err)
	}

	topK := len(legalMoves)
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
	if err == nil && policyResp != nil {
		if selected := chooseBestLegalCandidate(policyResp.Candidates, legalSet); selected != "" {
			return selected, nil
		}
	}
	if err != nil {
		log.Printf("warning: ai policy unavailable for %s: %v", gameIDLabel(gameID), err)
	}

	// Safe deterministic fallback.
	return legalMoves[0], nil
}

func legalUCIMovesByID(gameID string, pieces []sessionpkg.PieceState, side string) ([]string, error) {
	normalizedSide := strings.ToLower(strings.TrimSpace(side))
	seen := make(map[string]struct{}, 128)
	for _, p := range pieces {
		if strings.ToLower(p.Color) != normalizedSide {
			continue
		}
		destinations, err := sessionpkg.LegalMovesForSquareByID(gameID, p.File, p.Rank)
		if err != nil {
			return nil, err
		}
		for _, d := range destinations {
			uci := toUCIMove(p.File, p.Rank, d.File, d.Rank, d.RequiresPromotion)
			if uci == "" {
				continue
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

func chooseBestLegalCandidate(candidates []AIPolicyCandidate, legalSet map[string]struct{}) string {
	for _, c := range candidates {
		uci := normalizeUCI(c.UCI)
		if uci == "" {
			continue
		}
		if _, ok := legalSet[uci]; ok {
			return uci
		}
	}
	return ""
}

func toUCIMove(fromFile, fromRank, toFile, toRank int, requiresPromotion bool) string {
	if fromFile < 1 || fromFile > 8 || toFile < 1 || toFile > 8 || fromRank < 1 || fromRank > 8 || toRank < 1 || toRank > 8 {
		return ""
	}
	move := fmt.Sprintf("%c%d%c%d", byte('a'+fromFile-1), fromRank, byte('a'+toFile-1), toRank)
	if requiresPromotion {
		move += "q"
	}
	return move
}

func normalizeUCI(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// selectMoveWithFairyStockfish uses the local UCI engine for one legal best move.
// Strength is controlled via SetStrengthProfile (Skill Level + MultiPV).
// side selects which engine process to use (white|black).
func selectMoveWithFairyStockfish(fen, profile, side string) (string, error) {
	fs, err := getFairyStockfish(side)
	if err != nil {
		return "", err
	}
	if err := fs.SetStrengthProfile(profile); err != nil {
		return "", err
	}

	// Strength is primarily controlled by SetStrengthProfile (Skill Level + MultiPV).
	// We still apply a profile-specific time/depth cap for responsiveness.
	limit := engine.Limit{Depth: 20, MoveTime: 1200 * time.Millisecond}

	switch strings.ToLower(profile) {
	case "beginner":
		limit.Depth = 3
		limit.MoveTime = 250 * time.Millisecond
	case "intermediate":
		limit.Depth = 8
		limit.MoveTime = 600 * time.Millisecond
	case "advanced":
		limit.Depth = 14
		limit.MoveTime = 1000 * time.Millisecond
	case "master":
		limit.Depth = 20
		limit.MoveTime = 1800 * time.Millisecond
	}

	move, err := fs.BestMove(fen, limit)
	if err != nil {
		// One retry after restart
		if rerr := fs.Restart(); rerr == nil {
			_ = fs.SetStrengthProfile(profile)
			move, err = fs.BestMove(fen, limit)
		}
	}
	return move, err
}

// RunAIGame is the step-1 entry point for a single AI vs AI game.
// It delegates to simulation.RunSingleAIGame using the existing SelectAIMove.
func RunAIGame(gameID string) (simulation.Result, error) {
	return simulation.RunSingleAIGame(gameID, SelectAIMove)
}

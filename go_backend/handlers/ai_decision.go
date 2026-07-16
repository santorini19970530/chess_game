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

	if fs, ok := fsEngines[key]; ok && fs != nil && fs.IsRunning() {
		return fs, nil
	}
	if fs, ok := fsEngines[key]; ok && fs != nil {
		_ = fs.Close()
		delete(fsEngines, key)
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

// resetFairyStockfish drops a dead/broken engine for that side and starts a new one.
func resetFairyStockfish(side string) (*engine.FairyStockfish, error) {
	key := strings.ToLower(strings.TrimSpace(side))
	if key != "black" {
		key = "white"
	}
	fsEngineMu.Lock()
	if fs, ok := fsEngines[key]; ok && fs != nil {
		_ = fs.Close()
		delete(fsEngines, key)
	}
	fsEngineMu.Unlock()
	return getFairyStockfish(side)
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

	legalMoves, err := sessionpkg.AllLegalUCIMovesByID(gameID)
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
	allowDegrade := snapshot.Game.Mode == sessionpkg.GameModeAIVsAI
	if useFairyStockfish() {
		if move, err := selectMoveWithFairyStockfish(fen, profile, color, allowDegrade, gameType); err == nil && move != "" {
			if _, ok := legalSet[normalizeUCI(move)]; ok {
				return move, nil
			}
		} else if err != nil {
			if !allowDegrade {
				// Human vs AI: engine thinking budget exceeded / crash → AI flags (loss).
				if _, ferr := sessionpkg.FlagCurrentTurnByID(gameID); ferr != nil {
					log.Printf("warning: failed to flag AI after FS timeout %s: %v", gameIDLabel(gameID), ferr)
				} else {
					log.Printf("human_vs_ai: AI thinking timeout/failure — AI flagged (%v)", err)
				}
				return "", fmt.Errorf("ai thinking timeout: %w", err)
			}
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
// allowDegrade=true (AI vs AI): retry, then step down profiles before failing.
// allowDegrade=false (Human vs AI): retry same profile only; caller treats failure as AI timeout/loss.
// gameType is the session type (chess / xianqi / shogi); mapped to UCI_Variant before search.
func selectMoveWithFairyStockfish(fen, profile, side string, allowDegrade bool, gameType string) (string, error) {
	var lastErr error
	chain := []string{strings.ToLower(strings.TrimSpace(profile))}
	if chain[0] == "" {
		chain[0] = "intermediate"
	}
	if allowDegrade {
		chain = fsProfileFallbackChain(profile)
	}
	for _, p := range chain {
		limit := fsLimitForProfile(p)
		for attempt := 1; attempt <= 3; attempt++ {
			fs, err := getFairyStockfish(side)
			if err != nil {
				lastErr = err
				_, _ = resetFairyStockfish(side)
				continue
			}
			if err := fs.SetVariant(gameType); err != nil {
				lastErr = err
				_, _ = resetFairyStockfish(side)
				continue
			}
			if err := fs.SetStrengthProfile(p); err != nil {
				lastErr = err
				_, _ = resetFairyStockfish(side)
				continue
			}
			move, err := fs.BestMove(fen, limit)
			if err == nil && strings.TrimSpace(move) != "" {
				if allowDegrade && (p != strings.ToLower(strings.TrimSpace(profile)) || attempt > 1) {
					log.Printf("fairy-stockfish recovered side=%s profile=%s attempt=%d (requested=%s)",
						side, p, attempt, profile)
				}
				return move, nil
			}
			lastErr = err
			_, _ = resetFairyStockfish(side)
		}
		if allowDegrade {
			log.Printf("fairy-stockfish giving up profile=%s for side=%s; trying weaker profile", p, side)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("fairy-stockfish unavailable")
	}
	return "", lastErr
}

func fsProfileFallbackChain(profile string) []string {
	order := []string{"master", "advanced", "intermediate", "beginner"}
	start := strings.ToLower(strings.TrimSpace(profile))
	idx := 0
	for i, p := range order {
		if p == start {
			idx = i
			break
		}
	}
	return order[idx:]
}

func fsLimitForProfile(profile string) engine.Limit {
	limit := engine.Limit{Depth: 8, MoveTime: 600 * time.Millisecond}
	switch strings.ToLower(strings.TrimSpace(profile)) {
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
		limit.MoveTime = 1200 * time.Millisecond
	}
	return limit
}

// RunAIGame is the step-1 entry point for a single AI vs AI game.
// It delegates to simulation.RunSingleAIGame using the existing SelectAIMove.
func RunAIGame(gameID string) (simulation.Result, error) {
	return simulation.RunSingleAIGame(gameID, SelectAIMove)
}

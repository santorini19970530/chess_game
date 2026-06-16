package handlers

import (
	"fmt"
	"log"
	"sort"
	"strings"

	sessionpkg "go_backend/game/session"
)

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
	commonReq := AICommonRequest{
		RequestID:   fmt.Sprintf("%s-ai-%d", gameID, len(history)+1),
		GameID:      gameID,
		GameType:    "chess",
		Variant:     "chess",
		FEN:         fen,
		Color:       strings.ToLower(color),
		MoveNumber:  len(history) + 1,
		MoveHistory: history,
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

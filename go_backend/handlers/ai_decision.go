package handlers

import (
	"fmt"
	"log"
	"strings"

	sessionpkg "go_backend/game/session"
)

// SelectAIMove is the decision layer entry point (issue0018).
// For now this is a stub that returns a placeholder move or error.
// Replace with the real implementation from issue0018.
func SelectAIMove(gameID string) (string, error) {
	// Try to get at least one legal move as a safe fallback
	snapshot, err := sessionpkg.BuildSnapshotByID(gameID)
	if err != nil {
		return "", err
	}

	currentSide := strings.ToLower(snapshot.CurrentTurn)

	// Collect any legal destination for the current side
	for _, p := range snapshot.State {
		if strings.ToLower(p.Color) != currentSide {
			continue
		}
		dests, derr := sessionpkg.LegalMovesForSquareByID(gameID, p.File, p.Rank)
		if derr != nil {
			continue
		}
		if len(dests) > 0 {
			d := dests[0]
			move := fmt.Sprintf("%c%d%c%d", byte('a'+p.File-1), p.Rank, byte('a'+d.File-1), d.Rank)
			if d.RequiresPromotion {
				move += "q"
			}
			log.Printf("SelectAIMove stub: returning fallback move %s for %s", move, gameIDLabel(gameID))
			return move, nil
		}
	}

	return "", fmt.Errorf("no legal moves available for AI")
}
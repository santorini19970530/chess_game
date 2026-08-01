// CM3070 FP code
// ai_decision.go - thin http-layer wrappers around usecase/aimove

package handlers

import (
	"go_backend/simulation"
	"go_backend/usecase/aimove"
)

// SelectAIMove - picks an ai move via the aimove use case (strategy chain)
func SelectAIMove(gameID string) (string, error) {
	return aimove.SelectAIMove(gameID)
}

// RunAIGame - runs one ai-vs-ai game via simulation.RunSingleAIGame and SelectAIMove
func RunAIGame(gameID string) (simulation.Result, error) {
	return simulation.RunSingleAIGame(gameID, aimove.SelectAIMove)
}

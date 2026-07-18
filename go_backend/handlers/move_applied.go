package handlers

import sessionpkg "go_backend/game/session"

// moveAppliedPayload is the WebSocket data for event "move_applied".
// isCapture comes from the latest history entry so FE can pick capture vs move SFX.
func moveAppliedPayload(gameID, command string) map[string]interface{} {
	isCapture, _ := sessionpkg.LastMoveIsCaptureByID(gameID)
	return map[string]interface{}{
		"command":   command,
		"isCapture": isCapture,
	}
}

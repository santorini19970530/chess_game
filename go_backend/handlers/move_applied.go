package handlers

import sessionpkg "go_backend/game/session"

// moveAppliedPayload is the WebSocket data for event "move_applied".
// isCapture comes from the latest history entry so FE can pick capture vs move SFX.
func moveAppliedPayload(gameID, command string) map[string]interface{} {
	isCapture, _ := sessionpkg.LastMoveIsCaptureByID(gameID)
	payload := map[string]interface{}{
		"command":   command,
		"isCapture": isCapture,
	}
	attachClockFields(payload, gameID, nil)
	return payload
}

// attachClockFields adds clock + remaining to a socket payload.
// If clk is nil, loads/settles via snapshot for gameID.
func attachClockFields(payload map[string]interface{}, gameID string, clk *sessionpkg.Clock) {
	if payload == nil {
		return
	}
	if clk == nil {
		snap, err := sessionpkg.BuildSnapshotByID(gameID)
		if err != nil || snap.Game.Clock == nil {
			return
		}
		clk = snap.Game.Clock
	}
	payload["clock"] = clk
	payload["remaining"] = map[string]int64{
		"white": clk.WhiteRemainingMs,
		"black": clk.BlackRemainingMs,
	}
}

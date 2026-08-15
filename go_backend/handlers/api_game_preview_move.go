// CM3070 FP code
// api_game_preview_move.go - what-if preview without mutating the live session

package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	sessionpkg "go_backend/game/session"
)

// previewMoveRequest is the JSON body for preview-move
type previewMoveRequest struct {
	Command string `json:"command"`
}

// previewMoveResponse is the success payload for a what-if child position
type previewMoveResponse struct {
	Command          string                  `json:"command"`
	FEN              string                  `json:"fen"`
	EvalCPWhite      int                     `json:"eval_cp_white"`
	WinChanceWhite   float64                 `json:"win_chance_white"`
	WinChanceBlack   float64                 `json:"win_chance_black"`
	EvaluationSource string                  `json:"evaluation_source"`
	SuggestedMoves   []analyzerSuggestedMove `json:"suggested_moves,omitempty"`
}

// postAPIGamePreviewMove - evaluates one legal candidate on a temporary child fen without committing the session
func (h *Handler) postAPIGamePreviewMove(w http.ResponseWriter, r *http.Request, gameID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	commandText, err := readPreviewMoveCommand(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if commandText == "" {
		writeJSONError(w, http.StatusBadRequest, "Empty command")
		return
	}

	currentGame, err := sessionpkg.GetGameSessionByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
	}
	if currentGame.Result != sessionpkg.GameResultInProgress {
		message := currentGame.Outcome.Message
		if message == "" {
			message = "Game already ended."
		}
		writeJSONError(w, http.StatusConflict, message)
		return
	}

	normalized, childFEN, err := sessionpkg.PreviewChildFENByCommandByID(gameID, commandText)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	analysis, err := analyzeByRequest(analyzerRequest{
		RequestID: "preview-" + gameID,
		FEN:       childFEN,
		Color:     colorToMoveFromFEN(childFEN),
		TopK:      5,
		GameType:  string(currentGame.Type),
	})
	if err != nil {
		log.Printf("preview-move analyze failed %s: %v", gameIDLabel(gameID), err)
		writeJSONError(w, http.StatusServiceUnavailable, analyzerUserSafeError(err))
		return
	}

	resp := previewMoveResponse{
		Command:          normalized,
		FEN:              childFEN,
		EvalCPWhite:      analysis.EvalCPWhite,
		WinChanceWhite:   analysis.WinChanceWhite,
		WinChanceBlack:   analysis.WinChanceBlack,
		EvaluationSource: analysis.EvaluationSource,
		SuggestedMoves:   analysis.SuggestedMoves,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Response encode error")
	}
}

// readPreviewMoveCommand - reads command from json body or form field
func readPreviewMoveCommand(r *http.Request) (string, error) {
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(ct, "application/json") {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			return "", err
		}
		var req previewMoveRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return "", fmt.Errorf(`expected JSON body {"command":"..."}`)
		}
		return strings.TrimSpace(req.Command), nil
	}
	if err := r.ParseForm(); err != nil {
		return "", fmt.Errorf("invalid command payload")
	}
	return strings.TrimSpace(r.FormValue("command")), nil
}

// colorToMoveFromFEN - maps fen active-color field to white/black for analyzer requests
func colorToMoveFromFEN(fen string) string {
	parts := strings.Fields(strings.TrimSpace(fen))
	if len(parts) >= 2 && parts[1] == "b" {
		return "black"
	}
	return "white"
}

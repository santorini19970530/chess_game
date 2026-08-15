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
	Command          string                       `json:"command"`
	FEN              string                       `json:"fen"`
	EvalCPWhite      int                          `json:"eval_cp_white"`
	WinChanceWhite   float64                      `json:"win_chance_white"`
	WinChanceBlack   float64                      `json:"win_chance_black"`
	EvaluationSource string                       `json:"evaluation_source"`
	SuggestedMoves   []analyzerSuggestedMove      `json:"suggested_moves,omitempty"`
	State            []sessionpkg.PieceState      `json:"state,omitempty"`
	Captured         sessionpkg.CapturedSummary   `json:"captured,omitempty"`
	CurrentTurn      string                       `json:"currentTurn,omitempty"`
	CheckedSide      string                       `json:"checkedSide,omitempty"`
	Explanation      string                       `json:"explanation,omitempty"`
	ExplanationSource string                      `json:"explanation_source,omitempty"`
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

	child, err := sessionpkg.PreviewChildPositionByCommandByID(gameID, commandText)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	analysis, err := analyzeByRequest(analyzerRequest{
		RequestID: "preview-" + gameID,
		FEN:       child.FEN,
		Color:     colorToMoveFromFEN(child.FEN),
		TopK:      5,
		GameType:  string(currentGame.Type),
	})
	if err != nil {
		log.Printf("preview-move analyze failed %s: %v", gameIDLabel(gameID), err)
		writeJSONError(w, http.StatusServiceUnavailable, analyzerUserSafeError(err))
		return
	}

	history, _ := sessionpkg.MoveHistoryByID(gameID)
	skillLevel := sessionpkg.ResolveSkillLevel(currentGame.Config.SkillLevel, currentGame.Config.AIProfile)
	humanColor := ""
	if currentGame.Mode == sessionpkg.GameModeHumanVsAI {
		humanColor = strings.ToLower(strings.TrimSpace(currentGame.Config.HumanColor))
	}
	hints := conceptHintsFromAnalysis(*analysis)
	explanation := ""
	explanationSource := ""
	if explained, eerr := explainPreviewMove(explainRequest{
		RequestID:    "preview-explain-" + gameID,
		FEN:          child.FEN,
		Color:        colorToMoveFromFEN(child.FEN),
		GameType:     string(currentGame.Type),
		SkillLevel:   skillLevel,
		HumanColor:   humanColor,
		ConceptHints: hints,
		MoveUCI:      child.Command,
		MoveSAN:      child.Command,
		MoveHistory:  history,
		Preview:      true,
	}); eerr != nil {
		log.Printf("preview-move explain failed %s: %v", gameIDLabel(gameID), eerr)
	} else if explained != nil {
		explanation = strings.TrimSpace(explained.Explanation)
		explanationSource = explained.Source
	}

	resp := previewMoveResponse{
		Command:           child.Command,
		FEN:               child.FEN,
		EvalCPWhite:       analysis.EvalCPWhite,
		WinChanceWhite:    analysis.WinChanceWhite,
		WinChanceBlack:    analysis.WinChanceBlack,
		EvaluationSource:  analysis.EvaluationSource,
		SuggestedMoves:    analysis.SuggestedMoves,
		State:             child.State,
		Captured:          child.Captured,
		CurrentTurn:       child.CurrentTurn,
		CheckedSide:       child.CheckedSide,
		Explanation:       explanation,
		ExplanationSource: explanationSource,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Response encode error")
	}
}

// explainPreviewMove - asks python for coach text on a what-if candidate (not stored as live explain)
func explainPreviewMove(req explainRequest) (*explainResponse, error) {
	req.Preview = true
	req.Quick = false
	if out, err := explainByRequest(req); err == nil && out != nil && strings.TrimSpace(out.Explanation) != "" {
		return out, nil
	}
	quick := req
	quick.Quick = true
	return explainByRequest(quick)
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

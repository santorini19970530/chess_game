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

// request body for the load-moves endpoint
type loadMovesRequest struct {
	Raw string `json:"raw"`
}

// parseLoadMovesRaw - turns pasted uci text or a saved play json blob into uci moves
func parseLoadMovesRaw(raw string) (moves []string, gameType string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, "", fmt.Errorf("empty move input")
	}
	if strings.HasPrefix(raw, "{") {
		return extractUCIFromPlayJSON(raw)
	}
	moves, err = extractUCIList(raw)
	return moves, "", err
}

// extractUCIList - pulls uci move tokens from free-form pasted text
func extractUCIList(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty move list")
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', ',', ';':
			return true
		default:
			return false
		}
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		tok := strings.ToLower(strings.TrimSpace(f))
		if tok == "" {
			continue
		}
		out = append(out, tok)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty move list")
	}
	return out, nil
}

// extractUCIFromPlayJSON - pulls uci moves from a saved play/export json blob
func extractUCIFromPlayJSON(raw string) ([]string, string, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, "", fmt.Errorf("invalid play JSON: %w", err)
	}

	gameType := ""
	if gt, ok := payload["game_type"]; ok {
		_ = json.Unmarshal(gt, &gameType)
		gameType = strings.TrimSpace(strings.ToLower(gameType))
	}
	if gameType == "" {
		if g, ok := payload["game"]; ok {
			var gameObj struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(g, &gameObj); err == nil {
				gameType = strings.TrimSpace(strings.ToLower(gameObj.Type))
			}
		}
	}

	if histRaw, ok := payload["history"]; ok {
		var history []string
		if err := json.Unmarshal(histRaw, &history); err == nil && len(history) > 0 {
			moves := make([]string, 0, len(history))
			for _, line := range history {
				uci := stripHistorySideLabel(line)
				if uci == "" {
					continue
				}
				moves = append(moves, uci)
			}
			if len(moves) == 0 {
				return nil, gameType, fmt.Errorf("play JSON history had no moves")
			}
			return moves, gameType, nil
		}
	}

	if detRaw, ok := payload["history_detailed"]; ok {
		var detailed []struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(detRaw, &detailed); err == nil && len(detailed) > 0 {
			moves := make([]string, 0, len(detailed))
			for _, entry := range detailed {
				uci := strings.ToLower(strings.TrimSpace(entry.Command))
				if uci == "" {
					continue
				}
				moves = append(moves, uci)
			}
			if len(moves) == 0 {
				return nil, gameType, fmt.Errorf("play JSON history_detailed had no commands")
			}
			return moves, gameType, nil
		}
	}

	return nil, gameType, fmt.Errorf("play JSON missing history or history_detailed moves")
}

// stripHistorySideLabel - turns "White: e2e4" / "Black:e7e5" into "e2e4"
func stripHistorySideLabel(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	if i := strings.Index(line, ":"); i >= 0 {
		line = strings.TrimSpace(line[i+1:])
	}
	return strings.ToLower(line)
}

// postAPIGameLoadMoves - creates a clock-off hvh session and applies a uci/json move list for review
func (h *Handler) postAPIGameLoadMoves(w http.ResponseWriter, r *http.Request, gameID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	template, err := sessionpkg.GetGameSessionByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid load-moves payload")
		return
	}
	var req loadMovesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, `Expected JSON body {"raw":"..."}`)
		return
	}

	// empty raw = start position only (used by review Prev back to ply 0).
	var moves []string
	var jsonGameType string
	if strings.TrimSpace(req.Raw) == "" {
		moves = nil
	} else {
		var parseErr error
		moves, jsonGameType, parseErr = parseLoadMovesRaw(req.Raw)
		if parseErr != nil {
			writeJSONError(w, http.StatusBadRequest, parseErr.Error())
			return
		}
	}

	gameType := template.Type
	if jsonGameType != "" {
		gameType = sessionpkg.GameType(jsonGameType)
	}
	if gameType != sessionpkg.GameTypeChess &&
		gameType != sessionpkg.GameTypeXiangqi &&
		gameType != sessionpkg.GameTypeShogi {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("unsupported game type %q", gameType))
		return
	}

	if err := sessionpkg.ArchiveGameIfNeededByID(gameID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to archive current game")
		return
	}

	// fresh session: HvH + default clock-off so replay cannot flag.
	game, err := sessionpkg.CreateGame(
		sessionpkg.GameModeHumanVsHuman,
		gameType,
		"white",
		1,
		"",
		template.Config.AIProfile,
	)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if skill := strings.TrimSpace(template.Config.SkillLevel); skill != "" {
		if updated, setErr := sessionpkg.SetSkillLevelByID(game.ID, skill); setErr == nil {
			game = updated
		}
	}

	var lastUCI string
	for i, uci := range moves {
		normalized, applyErr := sessionpkg.ApplyMoveByCommandByID(game.ID, uci)
		if applyErr != nil {
			writeJSONError(
				w,
				http.StatusBadRequest,
				fmt.Sprintf("illegal move at ply %d: %s: %v", i+1, uci, applyErr),
			)
			return
		}
		lastUCI = normalized
	}

	if lastUCI != "" {
		enqueueCurrentPositionAnalysis(game.ID, lastUCI)
	}

	snapshot, err := sessionpkg.BuildSnapshotByID(game.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load game state")
		return
	}
	log.Printf(
		"api load-moves %s -> %s plies=%d type=%s",
		gameIDLabel(gameID),
		gameIDLabel(game.ID),
		len(moves),
		game.Type,
	)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(gameStateResponse{
		CurrentTurn:     snapshot.CurrentTurn,
		CheckedSide:     snapshot.CheckedSide,
		Game:            snapshot.Game,
		Captured:        snapshot.Captured,
		History:         snapshot.History,
		HistoryDetailed: snapshot.HistoryDetailed,
		State:           snapshot.State,
	}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Response encode error")
	}
}

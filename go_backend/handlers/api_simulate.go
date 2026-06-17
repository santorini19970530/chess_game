package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	session "go_backend/game/session"
	"go_backend/simulation"
)

type simulateRequest struct {
	Games   int    `json:"games"`
	Profile string `json:"profile"`
}

type gameResult struct {
	Result          string                     `json:"result"`
	Winner          string                     `json:"winner,omitempty"`
	Moves           int                        `json:"moves"`
	HistoryDetailed []session.MoveHistoryEntry `json:"history_detailed,omitempty"`
}

type simulateResponse struct {
	Games     int          `json:"games"`
	WhiteWins int          `json:"white_wins"`
	BlackWins int          `json:"black_wins"`
	Draws     int          `json:"draws"`
	AvgMoves  float64      `json:"avg_moves"`
	Results   []gameResult `json:"results,omitempty"`
}

const maxSimulationPlies = 600

var (
	simulationRunMu       sync.Mutex
	simulationRunInFlight bool
)

func tryStartSimulationRun() bool {
	simulationRunMu.Lock()
	defer simulationRunMu.Unlock()
	if simulationRunInFlight {
		return false
	}
	simulationRunInFlight = true
	return true
}

func finishSimulationRun() {
	simulationRunMu.Lock()
	simulationRunInFlight = false
	simulationRunMu.Unlock()
}

func parseDetailsFlag(r *http.Request) (bool, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("details"))
	if raw == "" {
		return true, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("details must be true or false")
	}
	return value, nil
}

func (h *Handler) APISimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if !tryStartSimulationRun() {
		writeJSONError(w, http.StatusConflict, "simulation run already in progress")
		return
	}
	defer finishSimulationRun()

	originalActiveID := strings.TrimSpace(session.GetGameSession().ID)
	if originalActiveID != "" {
		defer func() {
			if err := session.ActivateGame(originalActiveID); err != nil {
				log.Printf("warning: failed to restore active game %s after simulation: %v", gameIDLabel(originalActiveID), err)
			}
		}()
	}

	includeDetails, detailsErr := parseDetailsFlag(r)
	if detailsErr != nil {
		writeJSONError(w, http.StatusBadRequest, detailsErr.Error())
		return
	}

	var req simulateRequest

	// Support both JSON and application/x-www-form-urlencoded
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if ct == "" || strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid form payload")
			return
		}
		if g := strings.TrimSpace(r.FormValue("games")); g != "" {
			if _, scanErr := fmt.Sscanf(g, "%d", &req.Games); scanErr != nil {
				writeJSONError(w, http.StatusBadRequest, "games must be a valid integer")
				return
			}
		}
		req.Profile = strings.TrimSpace(r.FormValue("profile"))
	}

	if req.Games == 0 {
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	if req.Games < 1 {
		writeJSONError(w, http.StatusBadRequest, "games must be >= 1")
		return
	}
	if req.Games > 1000 {
		writeJSONError(w, http.StatusBadRequest, "games must be <= 1000")
		return
	}

	profile := strings.TrimSpace(req.Profile)
	if profile == "" {
		profile = "intermediate"
	}

	var white, black, draws, totalMoves int
	results := make([]gameResult, 0, req.Games)
	archiveItems := make([]simulation.ResultWithGameID, 0, req.Games)

	for i := 0; i < req.Games; i++ {
		gameNum := i + 1
		log.Printf("=== simulate game %d/%d started (profile=%s) ===", gameNum, req.Games, profile)

		game, err := session.CreateGame(session.GameModeAIVsAI, session.GameTypeChess, "white", 1, "", profile)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to create game")
			return
		}

		// Broadcast game start (global so any connected client sees it)
		gameSocketHub.BroadcastGlobal(socketEventSimulationGameEnd, map[string]interface{}{
			"game_num": gameNum,
			"status":   "started",
		})

		start := time.Now()

		// Run game move-by-move for live streaming
		moveCount := 0
		var runErr error
		for ply := 0; ply < maxSimulationPlies; ply++ {
			if ctxErr := r.Context().Err(); ctxErr != nil {
				runErr = ctxErr
				break
			}
			currentGame, _ := session.RefreshGameSessionOutcomeByID(game.ID)
			if currentGame.Result != session.GameResultInProgress {
				break
			}

			move, err := SelectAIMove(game.ID)
			if err != nil {
				runErr = fmt.Errorf("failed to select AI move: %w", err)
				break
			}
			move = strings.TrimSpace(move)
			if move == "" {
				runErr = fmt.Errorf("failed to select AI move: empty move")
				break
			}

			if _, applyErr := session.ApplyMoveByCommandByID(game.ID, move); applyErr != nil {
				runErr = fmt.Errorf("failed to apply AI move: %w", applyErr)
				break
			}
			moveCount++

			// Emit live move event (global)
			gameSocketHub.BroadcastGlobal(socketEventSimulationMove, map[string]interface{}{
				"game_num": gameNum,
				"move":     move,
				"move_num": moveCount,
			})
		}
		if runErr == nil {
			currentGame, refreshErr := session.RefreshGameSessionOutcomeByID(game.ID)
			if refreshErr != nil {
				runErr = fmt.Errorf("failed to refresh game outcome: %w", refreshErr)
			} else if currentGame.Result == session.GameResultInProgress {
				runErr = fmt.Errorf("simulation exceeded max plies (%d)", maxSimulationPlies)
			}
		}
		if runErr != nil {
			if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) {
				log.Printf("simulation request canceled by client while running game %d/%d", gameNum, req.Games)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("simulation game %d failed: %v", gameNum, runErr))
			return
		}

		snapshot, snapshotErr := session.BuildSnapshotByID(game.ID)
		if snapshotErr != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to snapshot simulation game %d", gameNum))
			return
		}

		elapsed := time.Since(start)
		moveCount = len(snapshot.History)

		log.Printf("=== simulate game %d/%d finished: result=%s winner=%q moves=%d (%.1fs) ===",
			gameNum, req.Games, snapshot.Game.Result, snapshot.Game.Outcome.Winner, moveCount, elapsed.Seconds())

		switch snapshot.Game.Result {
		case session.GameResultWhiteWin:
			white++
		case session.GameResultBlackWin:
			black++
		case session.GameResultDraw:
			draws++
		}
		totalMoves += moveCount

		historyDetailed := []session.MoveHistoryEntry(nil)
		if includeDetails {
			historyDetailed = snapshot.HistoryDetailed
		}

		results = append(results, gameResult{
			Result:          string(snapshot.Game.Result),
			Winner:          snapshot.Game.Outcome.Winner,
			Moves:           moveCount,
			HistoryDetailed: historyDetailed,
		})

		archiveItems = append(archiveItems, simulation.ResultWithGameID{
			GameID:    game.ID,
			Profile:   profile,
			Result:    snapshot.Game.Result,
			Winner:    snapshot.Game.Outcome.Winner,
			MoveCount: moveCount,
			// ponytail: keep archive self-contained; if this grows too large, switch to summary-only
			// files with optional separate move-history export.
			HistoryDetailed: snapshot.HistoryDetailed,
		})
	}

	// Persist each game into its own JSON file under a run folder
	if err := simulation.ArchiveSimulationRun(archiveItems); err != nil {
		log.Printf("warning: failed to archive simulation run: %v", err)
	}

	avg := 0.0
	if req.Games > 0 {
		avg = float64(totalMoves) / float64(req.Games)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(simulateResponse{
		Games:     req.Games,
		WhiteWins: white,
		BlackWins: black,
		Draws:     draws,
		AvgMoves:  avg,
		Results:   results,
	})

	gameSocketHub.BroadcastGlobal(socketEventSimulationDone, map[string]interface{}{
		"games":      req.Games,
		"white_wins": white,
		"black_wins": black,
		"draws":      draws,
		"avg_moves":  avg,
	})
}

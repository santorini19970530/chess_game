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
	Games         int    `json:"games"`
	Profile       string `json:"profile"`
	WhiteProfile  string `json:"white_profile"`
	BlackProfile  string `json:"black_profile"`
	GameType      string `json:"game"`
	GameTypeAlt   string `json:"game_type"`
}

type gameResult struct {
	Result          string                     `json:"result"`
	Winner          string                     `json:"winner,omitempty"`
	Moves           int                        `json:"moves"`
	DurationMs      int64                      `json:"duration_ms"`
	AvgMoveMs       int64                      `json:"avg_move_ms"`
	HistoryDetailed []session.MoveHistoryEntry `json:"history_detailed,omitempty"`
}

type simulateResponse struct {
	Games           int          `json:"games"`
	GameType        string       `json:"game_type"`
	WhiteProfile    string       `json:"white_profile"`
	BlackProfile    string       `json:"black_profile"`
	Profile         string       `json:"profile,omitempty"`
	WhiteWins       int          `json:"white_wins"`
	BlackWins       int          `json:"black_wins"`
	Draws           int          `json:"draws"`
	AvgMoves        float64      `json:"avg_moves"`
	AvgDurationMs   float64      `json:"avg_duration_ms"`
	P95DurationMs   int64        `json:"p95_duration_ms"`
	Results         []gameResult `json:"results,omitempty"`
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
		req.WhiteProfile = strings.TrimSpace(r.FormValue("white_profile"))
		req.BlackProfile = strings.TrimSpace(r.FormValue("black_profile"))
		req.GameType = strings.TrimSpace(r.FormValue("game"))
		if req.GameType == "" {
			req.GameType = strings.TrimSpace(r.FormValue("game_type"))
		}
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

	whiteProfile, blackProfile, profileErr := resolveSimulateProfiles(req.Profile, req.WhiteProfile, req.BlackProfile)
	if profileErr != nil {
		writeJSONError(w, http.StatusBadRequest, profileErr.Error())
		return
	}

	gameTypeRaw := strings.TrimSpace(req.GameType)
	if gameTypeRaw == "" {
		gameTypeRaw = strings.TrimSpace(req.GameTypeAlt)
	}
	if gameTypeRaw == "" {
		gameTypeRaw = "chess"
	}
	gameType, err := parseSupportedGameType(gameTypeRaw)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	gameTypeRaw = string(gameType)

	var white, black, draws, totalMoves int
	results := make([]gameResult, 0, req.Games)
	archiveItems := make([]simulation.ResultWithGameID, 0, req.Games)
	durations := make([]int64, 0, req.Games)

	for i := 0; i < req.Games; i++ {
		gameNum := i + 1
		log.Printf("=== simulate game %d/%d started (game=%s white=%s black=%s) ===", gameNum, req.Games, gameTypeRaw, whiteProfile, blackProfile)

		game, err := session.CreateGame(session.GameModeAIVsAI, gameType, "white", 1, "", whiteProfile)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to create game")
			return
		}
		if _, err := session.SetAISideProfilesByID(game.ID, whiteProfile, blackProfile); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to set side profiles")
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
		durationMs := elapsed.Milliseconds()
		avgMoveMs := simulation.ComputeAvgMoveMs(durationMs, moveCount)
		durations = append(durations, durationMs)

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
			DurationMs:      durationMs,
			AvgMoveMs:       avgMoveMs,
			HistoryDetailed: historyDetailed,
		})

		archiveItems = append(archiveItems, simulation.ResultWithGameID{
			GameID:       game.ID,
			GameType:     gameTypeRaw,
			WhiteProfile: whiteProfile,
			BlackProfile: blackProfile,
			Result:       snapshot.Game.Result,
			Winner:       snapshot.Game.Outcome.Winner,
			MoveCount:    moveCount,
			DurationMs:   durationMs,
			AvgMoveMs:    avgMoveMs,
			HistoryDetailed: snapshot.HistoryDetailed,
		})
		if whiteProfile == blackProfile {
			archiveItems[len(archiveItems)-1].Profile = whiteProfile
		}
	}

	// Persist each game into its own JSON file under a run folder
	if err := simulation.ArchiveSimulationRun(archiveItems); err != nil {
		log.Printf("warning: failed to archive simulation run: %v", err)
	}

	avg := 0.0
	if req.Games > 0 {
		avg = float64(totalMoves) / float64(req.Games)
	}
	avgDuration := simulation.MeanMs(durations)
	p95Duration := simulation.PercentileMs(durations, 95)

	resp := simulateResponse{
		Games:         req.Games,
		GameType:      gameTypeRaw,
		WhiteProfile:  whiteProfile,
		BlackProfile:  blackProfile,
		WhiteWins:     white,
		BlackWins:     black,
		Draws:         draws,
		AvgMoves:      avg,
		AvgDurationMs: avgDuration,
		P95DurationMs: p95Duration,
		Results:       results,
	}
	if whiteProfile == blackProfile {
		resp.Profile = whiteProfile
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)

	gameSocketHub.BroadcastGlobal(socketEventSimulationDone, map[string]interface{}{
		"games":            req.Games,
		"game_type":        gameTypeRaw,
		"white_profile":    whiteProfile,
		"black_profile":    blackProfile,
		"white_wins":       white,
		"black_wins":       black,
		"draws":            draws,
		"avg_moves":        avg,
		"avg_duration_ms":  avgDuration,
		"p95_duration_ms":  p95Duration,
	})
}

// resolveSimulateProfiles applies profile shorthand + per-side overrides.
// Unknown names are rejected (eval boundary).
func resolveSimulateProfiles(profile, whiteRaw, blackRaw string) (white, black string, err error) {
	profile = strings.TrimSpace(profile)
	whiteRaw = strings.TrimSpace(whiteRaw)
	blackRaw = strings.TrimSpace(blackRaw)

	fallback := "intermediate"
	if profile != "" {
		parsed, ok := session.ParseAIProfile(profile)
		if !ok {
			return "", "", fmt.Errorf("invalid profile %q", profile)
		}
		fallback = parsed
	}

	white = fallback
	black = fallback
	if whiteRaw != "" {
		parsed, ok := session.ParseAIProfile(whiteRaw)
		if !ok {
			return "", "", fmt.Errorf("invalid white_profile %q", whiteRaw)
		}
		white = parsed
	}
	if blackRaw != "" {
		parsed, ok := session.ParseAIProfile(blackRaw)
		if !ok {
			return "", "", fmt.Errorf("invalid black_profile %q", blackRaw)
		}
		black = parsed
	}
	return white, black, nil
}

func parseSupportedGameType(raw string) (session.GameType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "chess":
		return session.GameTypeChess, nil
	case "xianqi", "xiangqi":
		return session.GameTypeXiangqi, nil
	default:
		return "", fmt.Errorf("game must be chess or xianqi")
	}
}

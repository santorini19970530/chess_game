package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

func (h *Handler) APISimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req simulateRequest

	// Support both JSON and application/x-www-form-urlencoded
	ct := r.Header.Get("Content-Type")
	if ct == "" || ct == "application/x-www-form-urlencoded" {
		if err := r.ParseForm(); err == nil {
			if g := r.FormValue("games"); g != "" {
				fmt.Sscanf(g, "%d", &req.Games)
			}
			req.Profile = r.FormValue("profile")
		}
	}

	if req.Games == 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	profile := req.Profile
	if profile == "" {
		profile = "intermediate"
	}

	// Lightweight mode: skip heavy history_detailed unless explicitly requested
	includeDetails := true
	if d := r.URL.Query().Get("details"); d == "false" || d == "0" {
		includeDetails = false
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
		start := time.Now()
		res, err := simulation.RunSingleAIGame(game.ID, SelectAIMove)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "simulation failed")
			return
		}
		elapsed := time.Since(start)

		log.Printf("=== simulate game %d/%d finished: result=%s winner=%q moves=%d (%.1fs) ===",
			gameNum, req.Games, res.Result, res.Winner, res.MoveCount, elapsed.Seconds())

		switch res.Result {
		case session.GameResultWhiteWin:
			white++
		case session.GameResultBlackWin:
			black++
		case session.GameResultDraw:
			draws++
		}
		totalMoves += res.MoveCount

		var hist []session.MoveHistoryEntry
		if includeDetails {
			hist = res.HistoryDetailed
		}

		results = append(results, gameResult{
			Result:          string(res.Result),
			Winner:          res.Winner,
			Moves:           res.MoveCount,
			HistoryDetailed: hist,
		})

		archiveItems = append(archiveItems, simulation.ResultWithGameID{
			GameID:          game.ID,
			Profile:         profile,
			Result:          res.Result,
			Winner:          res.Winner,
			MoveCount:       res.MoveCount,
			HistoryDetailed: res.HistoryDetailed, // always keep full history for archive
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
}
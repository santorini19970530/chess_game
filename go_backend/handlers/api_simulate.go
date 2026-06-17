package handlers

import (
	"encoding/json"
	"log"
	"net/http"

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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
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

	var white, black, draws, totalMoves int
	results := make([]gameResult, 0, req.Games)

	for i := 0; i < req.Games; i++ {
		game, err := session.CreateGame(session.GameModeAIVsAI, session.GameTypeChess, "white", 1, "", profile)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to create game")
			return
		}
		res, err := simulation.RunSingleAIGame(game.ID, SelectAIMove)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "simulation failed")
			return
		}

		log.Printf("simulate game %d/%d result=%s winner=%q moves=%d",
			i+1, req.Games, res.Result, res.Winner, res.MoveCount)

		switch res.Result {
		case session.GameResultWhiteWin:
			white++
		case session.GameResultBlackWin:
			black++
		case session.GameResultDraw:
			draws++
		}
		totalMoves += res.MoveCount

		results = append(results, gameResult{
			Result:          string(res.Result),
			Winner:          res.Winner,
			Moves:           res.MoveCount,
			HistoryDetailed: res.HistoryDetailed,
		})
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
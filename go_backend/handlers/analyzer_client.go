package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	sessionpkg "go_backend/game/session"
)

var pyAnalyzerHTTPClient = &http.Client{
	Timeout: 2500 * time.Millisecond,
}

type analyzerRequest struct {
	RequestID string `json:"request_id"`
	FEN       string `json:"fen"`
	Color     string `json:"color"`
	TopK      int    `json:"top_k"`
}

func analyzerBaseURL() string {
	v := os.Getenv("PY_ANALYSER_URL")
	if v == "" {
		return "http://127.0.0.1:8001"
	}
	return v
}

func analyzeCurrentPosition() {
	game := sessionpkg.GetGameSession()
	history := sessionpkg.GetMoveHistory()
	reqPayload := analyzerRequest{
		RequestID: fmt.Sprintf("%s-move-%d", game.ID, len(history)),
		FEN:       sessionpkg.CurrentFEN(),
		Color:     string(sessionpkg.CurrentTurnColor()),
		TopK:      5,
	}

	body, err := json.Marshal(reqPayload)
	if err != nil {
		log.Printf("warning: analyzer request marshal failed: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, analyzerBaseURL()+"/analyze", bytes.NewReader(body))
	if err != nil {
		log.Printf("warning: analyzer request build failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := pyAnalyzerHTTPClient.Do(req)
	if err != nil {
		log.Printf("warning: analyzer request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("warning: analyzer response read failed: %v", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("warning: analyzer returned status=%d body=%s", resp.StatusCode, string(respBody))
		return
	}

	// Printed for testing as requested.
	log.Printf("analyzer response: %s", string(respBody))
}

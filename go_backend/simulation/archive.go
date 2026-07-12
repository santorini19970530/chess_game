package simulation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	session "go_backend/game/session"
)

// simulationArchiveRoot resolves data/simulations relative to the go.mod directory
// so archives land in one place whether the server or tests run from subpackages.
func simulationArchiveRoot() string {
	if v := strings.TrimSpace(os.Getenv("SIMULATION_ARCHIVE_DIR")); v != "" {
		return v
	}

	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Join("data", "simulations")
	}

	dir := cwd
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			if resolved, resolveErr := filepath.EvalSymlinks(dir); resolveErr == nil {
				dir = resolved
			}
			return filepath.Join(dir, "data", "simulations")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join("data", "simulations")
}

// ArchiveSimulationRun saves each completed game into its own JSON file
// inside a timestamped folder for this run. Future runs create new folders.
func ArchiveSimulationRun(results []ResultWithGameID) error {
	if len(results) == 0 {
		return nil
	}

	runID := fmt.Sprintf("%d-%dgames", time.Now().UnixNano(), len(results))
	runDir := filepath.Join(simulationArchiveRoot(), runID)

	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}

	for _, r := range results {
		filename := filepath.Join(runDir, r.GameID+".json")

		gameType := strings.TrimSpace(r.GameType)
		if gameType == "" {
			gameType = "chess"
		}
		white := strings.TrimSpace(r.WhiteProfile)
		black := strings.TrimSpace(r.BlackProfile)
		if white == "" {
			white = strings.TrimSpace(r.Profile)
		}
		if black == "" {
			black = strings.TrimSpace(r.Profile)
		}

		payload := map[string]interface{}{
			"game_id":          r.GameID,
			"game_type":        gameType,
			"mode":             "ai_vs_ai",
			"white_profile":    white,
			"black_profile":    black,
			"result":           r.Result,
			"winner":           r.Winner,
			"move_count":       r.MoveCount,
			"duration_ms":      r.DurationMs,
			"avg_move_ms":      r.AvgMoveMs,
			"history_detailed": r.HistoryDetailed,
			"archived_at":      time.Now().UTC().Format(time.RFC3339),
		}
		if white != "" && white == black {
			payload["profile"] = white
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filename, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// ResultWithGameID augments the normal Result with identifiers needed for archiving.
type ResultWithGameID struct {
	GameID          string
	GameType        string
	Profile         string // set when white==black (compat)
	WhiteProfile    string
	BlackProfile    string
	Result          session.GameResult
	Winner          string
	MoveCount       int
	DurationMs      int64
	AvgMoveMs       int64
	HistoryDetailed []session.MoveHistoryEntry
}

// AvgMoveMs returns wall-clock ms per move (0 if no moves).
func ComputeAvgMoveMs(durationMs int64, moveCount int) int64 {
	if moveCount <= 0 || durationMs < 0 {
		return 0
	}
	return durationMs / int64(moveCount)
}

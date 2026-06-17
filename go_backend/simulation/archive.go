package simulation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	session "go_backend/game/session"
)

const simulationArchiveDir = "data/simulations"

// ArchiveSimulationRun saves each completed game into its own JSON file
// inside a timestamped folder for this run. Future runs create new folders.
func ArchiveSimulationRun(results []ResultWithGameID) error {
	if len(results) == 0 {
		return nil
	}

	runID := fmt.Sprintf("%d-%dgames", time.Now().UnixNano(), len(results))
	runDir := filepath.Join(simulationArchiveDir, runID)

	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}

	for _, r := range results {
		filename := filepath.Join(runDir, r.GameID+".json")

		payload := map[string]interface{}{
			"game_id":          r.GameID,
			"mode":             "ai_vs_ai",
			"profile":          r.Profile,
			"result":           r.Result,
			"winner":           r.Winner,
			"move_count":       r.MoveCount,
			"history_detailed": r.HistoryDetailed,
			"archived_at":      time.Now().UTC().Format(time.RFC3339),
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
	Profile         string
	Result          session.GameResult
	Winner          string
	MoveCount       int
	HistoryDetailed []session.MoveHistoryEntry
}

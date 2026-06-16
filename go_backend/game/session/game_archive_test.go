package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	pieces "go_backend/game/piece"
)

func TestArchiveActiveGameIfNeeded_WritesJSONSnapshot(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	tempDir := t.TempDir()
	previousArchivePath := archivePath
	archivePath = filepath.Join(tempDir, "game_history.json")
	t.Cleanup(func() { archivePath = previousArchivePath })

	if _, err := ApplyMoveByCommand("e2e4"); err != nil {
		t.Fatalf("expected setup move to succeed: %v", err)
	}
	if _, err := ApplyMoveByCommand("e7e5"); err != nil {
		t.Fatalf("expected setup move to succeed: %v", err)
	}
	RefreshGameSessionOutcome()

	if err := ArchiveActiveGameIfNeeded(); err != nil {
		t.Fatalf("expected archive to succeed, got: %v", err)
	}

	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("expected archive file to exist: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected archive json to contain data")
	}
	raw := string(data)
	if strings.Contains(raw, "\"imgFile\"") {
		t.Fatalf("expected archive json to omit imgFile")
	}
	if strings.Contains(raw, "\"outcome\"") {
		t.Fatalf("expected archive json to omit outcome")
	}
	if strings.Contains(raw, "\"message\"") {
		t.Fatalf("expected archive json to omit message")
	}

	records, err := loadArchivedGames()
	if err != nil {
		t.Fatalf("expected archive json to be readable: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 archived game, got %d", len(records))
	}
	if len(records[0].History) != 2 {
		t.Fatalf("expected archived move history of 2, got %d", len(records[0].History))
	}
	if records[0].Game.Type != GameTypeChess {
		t.Fatalf("expected archived game type chess, got %q", records[0].Game.Type)
	}
	if records[0].Game.Result != GameResultInProgress {
		t.Fatalf("expected in-progress result for archived snapshot, got %q", records[0].Game.Result)
	}

	// Also verify board snapshot was persisted.
	foundWhitePawnE4 := false
	for _, piece := range records[0].State {
		if piece.Color == string(pieces.White) && piece.Kind == string(pieces.Pawn) && piece.File == 5 && piece.Rank == 4 {
			foundWhitePawnE4 = true
			break
		}
	}
	if !foundWhitePawnE4 {
		t.Fatalf("expected white pawn on e4 in archived state")
	}
}

func TestArchiveActiveGameIfNeeded_IncludesFlaggedByInHistory(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	tempDir := t.TempDir()
	previousArchivePath := archivePath
	archivePath = filepath.Join(tempDir, "game_history.json")
	t.Cleanup(func() { archivePath = previousArchivePath })

	FlagCurrentTurn() // White flags at game start.
	if err := ArchiveActiveGameIfNeeded(); err != nil {
		t.Fatalf("expected archive to succeed, got: %v", err)
	}

	records, err := loadArchivedGames()
	if err != nil {
		t.Fatalf("expected archive json to be readable: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 archived game, got %d", len(records))
	}
	found := false
	for _, entry := range records[0].History {
		if entry == "White: flag" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected history to include flag entry with side")
	}
}

// TestNewUniqueGameID_RapidCreationYieldsDistinctIDs verifies that many games created
// in quick succession all receive distinct IDs (covers the strengthened ID generator).
func TestNewUniqueGameID_RapidCreationYieldsDistinctIDs(t *testing.T) {
	resetGameSessionForTest()
	ids := make(map[string]bool)
	const n = 100
	for i := 0; i < n; i++ {
		g, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, "")
		if err != nil {
			t.Fatalf("create game %d failed: %v", i, err)
		}
		if ids[g.ID] {
			t.Fatalf("duplicate ID generated: %s", g.ID)
		}
		ids[g.ID] = true
	}
	if len(ids) != n {
		t.Fatalf("expected %d unique IDs, got %d", n, len(ids))
	}
}

// TestArchiveActiveGameIfNeeded_DoesNotDuplicateOnSecondCall ensures the archive
// logic (guarded by the Archived flag) never writes the same game ID twice.
func TestArchiveActiveGameIfNeeded_DoesNotDuplicateOnSecondCall(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	tempDir := t.TempDir()
	previousArchivePath := archivePath
	archivePath = filepath.Join(tempDir, "game_history.json")
	t.Cleanup(func() { archivePath = previousArchivePath })

	// Make a move so it qualifies for archiving
	if _, err := ApplyMoveByCommand("e2e4"); err != nil {
		t.Fatalf("setup move failed: %v", err)
	}
	RefreshGameSessionOutcome()

	if err := ArchiveActiveGameIfNeeded(); err != nil {
		t.Fatalf("first archive failed: %v", err)
	}
	records1, _ := loadArchivedGames()
	if len(records1) != 1 {
		t.Fatalf("expected 1 record after first archive, got %d", len(records1))
	}

	// Second call should be no-op (Archived flag set)
	if err := ArchiveActiveGameIfNeeded(); err != nil {
		t.Fatalf("second archive should succeed but do nothing: %v", err)
	}
	records2, _ := loadArchivedGames()
	if len(records2) != 1 {
		t.Fatalf("expected still 1 record after second archive, got %d", len(records2))
	}
	if records2[0].Game.ID != records1[0].Game.ID {
		t.Fatalf("ID changed unexpectedly")
	}
}

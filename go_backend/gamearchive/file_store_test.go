// CM3070 FP code
// file_store_test.go - tests for file store

package gamearchive

import (
	"path/filepath"
	"testing"

	sessionpkg "go_backend/game/session"
)

// TestJSONFileStore_RoundTrip - checks load/save of archived games on a temp path
func TestJSONFileStore_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "game_history.json")
	store := NewJSONFileStore(path)

	in := []sessionpkg.ArchivedGame{{
		Game: sessionpkg.ArchivedSession{ID: "game-1", Type: sessionpkg.GameTypeChess},
		History: []string{"e2e4"},
	}}
	if err := store.SaveArchivedGames(in); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := store.LoadArchivedGames()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(out) != 1 || out[0].Game.ID != "game-1" || len(out[0].History) != 1 {
		t.Fatalf("unexpected round-trip: %+v", out)
	}
	missing := NewJSONFileStore(filepath.Join(t.TempDir(), "missing.json"))
	empty, err := missing.LoadArchivedGames()
	if err != nil || len(empty) != 0 {
		t.Fatalf("missing file should load empty, got %v %v", empty, err)
	}
}

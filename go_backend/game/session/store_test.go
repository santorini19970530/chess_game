package session

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestCreateGame_ReturnsUniqueIDsAndIsolatedState(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	gameA, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, "")
	if err != nil {
		t.Fatalf("expected first game create success, got %v", err)
	}
	gameB, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, "")
	if err != nil {
		t.Fatalf("expected second game create success, got %v", err)
	}
	if gameA.ID == gameB.ID {
		t.Fatalf("expected unique game ids, got same id %q", gameA.ID)
	}

	if _, err := ApplyMoveByCommandByID(gameA.ID, "e2e4"); err != nil {
		t.Fatalf("expected move on first game to succeed, got %v", err)
	}
	snapshotA, err := BuildSnapshotByID(gameA.ID)
	if err != nil {
		t.Fatalf("expected first snapshot success, got %v", err)
	}
	snapshotB, err := BuildSnapshotByID(gameB.ID)
	if err != nil {
		t.Fatalf("expected second snapshot success, got %v", err)
	}
	if len(snapshotA.History) != 1 {
		t.Fatalf("expected first game history length 1, got %d", len(snapshotA.History))
	}
	if len(snapshotB.History) != 0 {
		t.Fatalf("expected second game history length 0, got %d", len(snapshotB.History))
	}
}

func TestSessionStore_GetExistingAndNonExisting(t *testing.T) {
	store := NewSessionStore()
	game := store.Create(GameSession{ID: "game-a"})

	got, ok := store.Get("game-a")
	if !ok {
		t.Fatalf("expected existing game to be found")
	}
	if got != game {
		t.Fatalf("expected same runtime game pointer for existing game")
	}

	if _, ok := store.Get("missing"); ok {
		t.Fatalf("expected missing game lookup to return false")
	}
}

func TestSessionStore_UpdateOnlyTargetGame(t *testing.T) {
	store := NewSessionStore()
	store.Create(GameSession{ID: "game-a"})
	store.Create(GameSession{ID: "game-b"})

	if err := store.Update("game-a", func(g *RuntimeGame) error {
		g.Session.Config.HumanColor = "black"
		g.State.MoveHistory = append(g.State.MoveHistory, "White: e2e4")
		return nil
	}); err != nil {
		t.Fatalf("expected update success, got %v", err)
	}

	gameA, ok := store.Get("game-a")
	if !ok {
		t.Fatalf("expected game-a to exist")
	}
	gameB, ok := store.Get("game-b")
	if !ok {
		t.Fatalf("expected game-b to exist")
	}
	if gameA.Session.Config.HumanColor != "black" {
		t.Fatalf("expected updated color for game-a")
	}
	if len(gameA.State.MoveHistory) != 1 {
		t.Fatalf("expected updated move history for game-a")
	}
	if gameB.Session.Config.HumanColor == "black" {
		t.Fatalf("expected game-b config to remain unchanged")
	}
	if len(gameB.State.MoveHistory) != 0 {
		t.Fatalf("expected game-b move history to remain unchanged")
	}
}

func TestSessionStore_DeleteRemovesTarget(t *testing.T) {
	store := NewSessionStore()
	store.Create(GameSession{ID: "game-a"})
	store.Create(GameSession{ID: "game-b"})

	if !store.Delete("game-a") {
		t.Fatalf("expected delete existing game to return true")
	}
	if _, ok := store.Get("game-a"); ok {
		t.Fatalf("expected deleted game-a to be missing")
	}
	if _, ok := store.Get("game-b"); !ok {
		t.Fatalf("expected other game to remain after delete")
	}
	if store.Delete("game-a") {
		t.Fatalf("expected delete missing game to return false")
	}
}

func TestSessionStore_ParallelUpdatesAcrossTwoGamesAreIsolated(t *testing.T) {
	store := NewSessionStore()
	store.Create(GameSession{ID: "game-a"})
	store.Create(GameSession{ID: "game-b"})

	const iterations = 120
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			if err := store.Update("game-a", func(g *RuntimeGame) error {
				g.State.MoveHistory = append(g.State.MoveHistory, fmt.Sprintf("A:%d", i))
				return nil
			}); err != nil {
				t.Errorf("unexpected update failure for game-a: %v", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			if err := store.Update("game-b", func(g *RuntimeGame) error {
				g.State.MoveHistory = append(g.State.MoveHistory, fmt.Sprintf("B:%d", i))
				return nil
			}); err != nil {
				t.Errorf("unexpected update failure for game-b: %v", err)
				return
			}
		}
	}()

	wg.Wait()

	gameA, _ := store.Get("game-a")
	gameB, _ := store.Get("game-b")
	if len(gameA.State.MoveHistory) != iterations {
		t.Fatalf("expected %d updates for game-a, got %d", iterations, len(gameA.State.MoveHistory))
	}
	if len(gameB.State.MoveHistory) != iterations {
		t.Fatalf("expected %d updates for game-b, got %d", iterations, len(gameB.State.MoveHistory))
	}
	for _, move := range gameA.State.MoveHistory {
		if !strings.HasPrefix(move, "A:") {
			t.Fatalf("expected game-a history to contain only A-prefixed updates, got %q", move)
		}
	}
	for _, move := range gameB.State.MoveHistory {
		if !strings.HasPrefix(move, "B:") {
			t.Fatalf("expected game-b history to contain only B-prefixed updates, got %q", move)
		}
	}
}

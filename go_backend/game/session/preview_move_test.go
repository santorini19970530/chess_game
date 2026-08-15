// CM3070 FP code
// preview_move_test.go - tests for restore-safe child fen preview

package session

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// previewObservable - captures live session fields that must survive preview
type previewObservable struct {
	fen         string
	history     []string
	turn        string
	result      GameResult
	updatedAt   string
	whiteMs     int64
	blackMs     int64
	tickMs      int64
	clockActive string
	capturedW   map[string]int
	capturedB   map[string]int
}

// capturePreviewObservable - reads observable session state for before/after compare
func capturePreviewObservable(t *testing.T, gameID string) previewObservable {
	t.Helper()
	fen, err := CurrentFENByID(gameID)
	if err != nil {
		t.Fatalf("fen: %v", err)
	}
	hist, err := MoveHistoryByID(gameID)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	turn, err := CurrentTurnColorByID(gameID)
	if err != nil {
		t.Fatalf("turn: %v", err)
	}
	sess, err := GetGameSessionByID(gameID)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	snap, err := BuildSnapshotByID(gameID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	out := previewObservable{
		fen:       fen,
		history:   append([]string(nil), hist...),
		turn:      turn,
		result:    sess.Result,
		updatedAt: sess.UpdatedAt,
		capturedW: copyStringIntMap(snap.Captured.White),
		capturedB: copyStringIntMap(snap.Captured.Black),
	}
	if sess.Clock != nil {
		out.whiteMs = sess.Clock.WhiteRemainingMs
		out.blackMs = sess.Clock.BlackRemainingMs
		out.tickMs = sess.Clock.LastTickUnixMs
		out.clockActive = sess.Clock.Active
	}
	return out
}

// assertPreviewObservableEqual - fails when preview mutated live session observables
func assertPreviewObservableEqual(t *testing.T, before, after previewObservable) {
	t.Helper()
	if before.fen != after.fen {
		t.Fatalf("fen changed: before=%q after=%q", before.fen, after.fen)
	}
	if !reflect.DeepEqual(before.history, after.history) {
		t.Fatalf("history changed: before=%v after=%v", before.history, after.history)
	}
	if before.turn != after.turn {
		t.Fatalf("turn changed: before=%q after=%q", before.turn, after.turn)
	}
	if before.result != after.result {
		t.Fatalf("result changed: before=%q after=%q", before.result, after.result)
	}
	if before.updatedAt != after.updatedAt {
		t.Fatalf("updatedAt changed: before=%q after=%q", before.updatedAt, after.updatedAt)
	}
	if before.whiteMs != after.whiteMs || before.blackMs != after.blackMs || before.tickMs != after.tickMs || before.clockActive != after.clockActive {
		t.Fatalf("clock changed: before=%+v after=%+v", before, after)
	}
	if !reflect.DeepEqual(before.capturedW, after.capturedW) || !reflect.DeepEqual(before.capturedB, after.capturedB) {
		t.Fatalf("captured/hands changed: before W=%v B=%v after W=%v B=%v", before.capturedW, before.capturedB, after.capturedW, after.capturedB)
	}
}

// TestPreviewChildFEN_Chess_RestoresIdentity - checks chess preview returns child fen and restores live state
func TestPreviewChildFEN_Chess_RestoresIdentity(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	before := capturePreviewObservable(t, game.ID)

	normalized, childFEN, err := PreviewChildFENByCommandByID(game.ID, "e2e4")
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if normalized != "e2e4" {
		t.Fatalf("normalized=%q", normalized)
	}
	if childFEN == before.fen {
		t.Fatal("child fen should differ from live fen")
	}
	if !strings.Contains(childFEN, " b ") {
		t.Fatalf("expected black to move in child fen, got %q", childFEN)
	}

	after := capturePreviewObservable(t, game.ID)
	assertPreviewObservableEqual(t, before, after)
}

// TestPreviewChildFEN_Chess_Promotion - checks chess promotion preview leaves start position intact
func TestPreviewChildFEN_Chess_Promotion(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	fen := "8/4P3/8/8/8/8/8/4K2k w - - 0 1"
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, fen, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	before := capturePreviewObservable(t, game.ID)

	normalized, childFEN, err := PreviewChildFENByCommandByID(game.ID, "e7e8q")
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if normalized != "e7e8q" {
		t.Fatalf("normalized=%q", normalized)
	}
	if !strings.Contains(childFEN, "Q") {
		t.Fatalf("expected queen on child fen, got %q", childFEN)
	}

	after := capturePreviewObservable(t, game.ID)
	assertPreviewObservableEqual(t, before, after)
}

// TestPreviewChildFEN_Chess_IllegalRejected - checks illegal preview fails without mutation
func TestPreviewChildFEN_Chess_IllegalRejected(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	before := capturePreviewObservable(t, game.ID)

	if _, _, err := PreviewChildFENByCommandByID(game.ID, "e2e5"); err == nil {
		t.Fatal("expected illegal preview to fail")
	}
	after := capturePreviewObservable(t, game.ID)
	assertPreviewObservableEqual(t, before, after)
}

// TestPreviewChildFEN_Chess_ClockUnchanged - checks preview does not settle or award clock
func TestPreviewChildFEN_Chess_ClockUnchanged(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := SetClockByID(game.ID, 60_000, 60_000, 1_000); err != nil {
		t.Fatalf("clock: %v", err)
	}
	if err := sessionStore.Update(game.ID, func(rg *RuntimeGame) error {
		rg.Session.Clock.LastTickUnixMs = time.Now().UTC().Add(-2 * time.Second).UnixMilli()
		return nil
	}); err != nil {
		t.Fatalf("backdate clock: %v", err)
	}
	before := capturePreviewObservable(t, game.ID)

	if _, _, err := PreviewChildFENByCommandByID(game.ID, "e2e4"); err != nil {
		t.Fatalf("preview: %v", err)
	}
	after := capturePreviewObservable(t, game.ID)
	assertPreviewObservableEqual(t, before, after)
}

// TestPreviewChildFEN_Xiangqi_RestoresIdentity - checks xiangqi preview restores live fen/history/turn
func TestPreviewChildFEN_Xiangqi_RestoresIdentity(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeXiangqi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	before := capturePreviewObservable(t, game.ID)

	normalized, childFEN, err := PreviewChildFENByCommandByID(game.ID, "a4a5")
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if normalized != "a4a5" {
		t.Fatalf("normalized=%q", normalized)
	}
	if childFEN == before.fen || !strings.Contains(childFEN, " b ") {
		t.Fatalf("bad child fen=%q", childFEN)
	}

	after := capturePreviewObservable(t, game.ID)
	assertPreviewObservableEqual(t, before, after)
}

// TestPreviewChildFEN_Shogi_RestoresHands - checks shogi capture preview restores fen and hands
func TestPreviewChildFEN_Shogi_RestoresHands(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	fen := "4k4/4p4/9/9/4R4/9/9/9/4K4[] w - - 0 1"
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, fen, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	before := capturePreviewObservable(t, game.ID)

	normalized, childFEN, err := PreviewChildFENByCommandByID(game.ID, "e5e8")
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if normalized != "e5e8" {
		t.Fatalf("normalized=%q", normalized)
	}
	if !strings.Contains(childFEN, "[P]") {
		t.Fatalf("expected white hand pawn in child fen, got %q", childFEN)
	}

	after := capturePreviewObservable(t, game.ID)
	assertPreviewObservableEqual(t, before, after)
	if after.capturedW["pawn"] != 0 {
		t.Fatalf("live hand should stay empty, got %+v", after.capturedW)
	}
}

// TestPreviewChildFEN_TerminalRejected - checks terminal games cannot preview
func TestPreviewChildFEN_TerminalRejected(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := FlagCurrentTurnByID(game.ID); err != nil {
		t.Fatalf("flag: %v", err)
	}
	before := capturePreviewObservable(t, game.ID)
	if _, _, err := PreviewChildFENByCommandByID(game.ID, "e2e4"); err == nil {
		t.Fatal("expected terminal preview to fail")
	}
	after := capturePreviewObservable(t, game.ID)
	assertPreviewObservableEqual(t, before, after)
}

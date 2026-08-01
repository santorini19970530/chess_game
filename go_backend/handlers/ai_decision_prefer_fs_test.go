package handlers

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestUseFairyStockfish_EnvFlag - maps USE_FAIRY_STOCKFISH true/1 to on and other values to off
func TestUseFairyStockfish_EnvFlag(t *testing.T) {
	t.Setenv("USE_FAIRY_STOCKFISH", "")
	if useFairyStockfish() {
		t.Fatal("empty env should be off")
	}
	t.Setenv("USE_FAIRY_STOCKFISH", "true")
	if !useFairyStockfish() {
		t.Fatal("true should be on")
	}
	t.Setenv("USE_FAIRY_STOCKFISH", "1")
	if !useFairyStockfish() {
		t.Fatal("1 should be on")
	}
	t.Setenv("USE_FAIRY_STOCKFISH", "false")
	if useFairyStockfish() {
		t.Fatal("false should be off")
	}
}

// TestSelectAIMove_SourceStillPrefersFairyStockfishPath - fails if the fs play strategy is removed from the chain
func TestSelectAIMove_SourceStillPrefersFairyStockfishPath(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(thisFile)
	decisionRaw, err := os.ReadFile(filepath.Join(dir, "ai_decision.go"))
	if err != nil {
		t.Fatalf("read ai_decision.go: %v", err)
	}
	strategyRaw, err := os.ReadFile(filepath.Join(dir, "ai_move_strategy.go"))
	if err != nil {
		t.Fatalf("read ai_move_strategy.go: %v", err)
	}
	decision := string(decisionRaw)
	strategy := string(strategyRaw)

	if !strings.Contains(decision, "func SelectAIMove(") {
		t.Fatal("SelectAIMove missing")
	}
	if !strings.Contains(decision, "runAIMoveStrategies(") {
		t.Fatal("SelectAIMove must delegate to runAIMoveStrategies")
	}
	for _, snip := range []string{
		"type AIMoveStrategy interface",
		"type FairyStockfishPlayStrategy struct",
		"type HPVFallbackStrategy struct",
		"type FirstLegalFallbackStrategy struct",
		"selectMoveWithFairyStockfish(",
		"func aiMoveStrategies(",
	} {
		if !strings.Contains(strategy, snip) {
			t.Fatalf("ai_move_strategy.go missing %q — FS-prefer path may have been removed", snip)
		}
	}

	chainFn := "func aiMoveStrategies("
	idx := strings.Index(strategy, chainFn)
	if idx < 0 {
		t.Fatal("aiMoveStrategies missing")
	}
	rest := strategy[idx:]
	next := strings.Index(rest[len(chainFn):], "\nfunc ")
	body := rest
	if next >= 0 {
		body = rest[:len(chainFn)+next]
	}
	fsAt := strings.Index(body, "FairyStockfishPlayStrategy{}")
	hpvAt := strings.Index(body, "HPVFallbackStrategy{}")
	firstAt := strings.Index(body, "FirstLegalFallbackStrategy{}")
	if fsAt < 0 || hpvAt < 0 || firstAt < 0 {
		t.Fatal("aiMoveStrategies must append FS, HPV, and first-legal strategies")
	}
	if !(fsAt < hpvAt && hpvAt < firstAt) {
		t.Fatal("aiMoveStrategies must order FS before HPV before first-legal")
	}
	if !strings.Contains(body, "useFairyStockfish()") {
		t.Fatal("aiMoveStrategies must gate FairyStockfishPlayStrategy with useFairyStockfish()")
	}
}

// TestAiMoveStrategies_OrderByEnvAndGameType - checks chain composition for fs on/off and chess/variant
func TestAiMoveStrategies_OrderByEnvAndGameType(t *testing.T) {
	chess := &aiMoveContext{GameType: "chess"}
	t.Setenv("USE_FAIRY_STOCKFISH", "true")
	names := strategyNames(aiMoveStrategies(chess))
	if strings.Join(names, ",") != "fairy_stockfish_play,hpv_fallback,first_legal_fallback" {
		t.Fatalf("chess+fs: %v", names)
	}
	t.Setenv("USE_FAIRY_STOCKFISH", "false")
	names = strategyNames(aiMoveStrategies(chess))
	if strings.Join(names, ",") != "hpv_fallback,first_legal_fallback" {
		t.Fatalf("chess no fs: %v", names)
	}
	xq := &aiMoveContext{GameType: "xianqi"}
	t.Setenv("USE_FAIRY_STOCKFISH", "true")
	names = strategyNames(aiMoveStrategies(xq))
	if strings.Join(names, ",") != "fairy_stockfish_play,first_legal_fallback" {
		t.Fatalf("xianqi+fs: %v", names)
	}
}

// strategyNames - returns strategy names for assertions
func strategyNames(list []AIMoveStrategy) []string {
	out := make([]string, len(list))
	for i, s := range list {
		out[i] = s.Name()
	}
	return out
}

// TestNormalizeAIMove_ShogiDropAtToStar - maps shogi drop @ to * after lowercasing
func TestNormalizeAIMove_ShogiDropAtToStar(t *testing.T) {
	got := normalizeAIMove("shogi", "P@5e")
	if got != "p*5e" {
		t.Fatalf("got %q want p*5e", got)
	}
	got = normalizeAIMove("chess", "E2E4")
	if got != "e2e4" {
		t.Fatalf("got %q want e2e4", got)
	}
}

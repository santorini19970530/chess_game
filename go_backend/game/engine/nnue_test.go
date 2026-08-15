// CM3070 FP code
// nnue_test.go - resolve NNUE path from env / missing file

package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveNNUEPath_EnvFileWins - env path is used when the file exists
func TestResolveNNUEPath_EnvFileWins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultNNUEFilename)
	if err := os.WriteFile(path, []byte("not-a-real-net"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAIRY_STOCKFISH_NNUE_PATH", path)
	got := ResolveNNUEPath()
	if got != path {
		t.Fatalf("ResolveNNUEPath() = %q, want %q", got, path)
	}
}

// TestResolveNNUEPath_MissingEnvFileReturnsEmpty - missing env file is not a fallback hit
func TestResolveNNUEPath_MissingEnvFileReturnsEmpty(t *testing.T) {
	t.Setenv("FAIRY_STOCKFISH_NNUE_PATH", filepath.Join(t.TempDir(), "missing.nnue"))
	if got := ResolveNNUEPath(); got != "" {
		t.Fatalf("ResolveNNUEPath() = %q, want empty when env path is missing", got)
	}
}

// TestEvidenceCheckNNUE_UnsetPathFails - unset env is not pretrained-model evidence
func TestEvidenceCheckNNUE_UnsetPathFails(t *testing.T) {
	t.Setenv("FAIRY_STOCKFISH_NNUE_PATH", "")
	err := EvidenceCheckNNUE()
	if err == nil || !strings.Contains(err.Error(), "unset") {
		t.Fatalf("EvidenceCheckNNUE() = %v, want unset error", err)
	}
}

// TestEvidenceCheckNNUE_MissingFileFails - requested path that does not exist fails evidence
func TestEvidenceCheckNNUE_MissingFileFails(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nn-missing000000.nnue")
	t.Setenv("FAIRY_STOCKFISH_NNUE_PATH", missing)
	err := EvidenceCheckNNUE()
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("EvidenceCheckNNUE() = %v, want missing error", err)
	}
}

// TestEvidenceCheckNNUE_TinyFileRejected - placeholder bytes are not a real SF14 net
func TestEvidenceCheckNNUE_TinyFileRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), DefaultNNUEFilename)
	if err := os.WriteFile(path, []byte("not-a-real-net"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAIRY_STOCKFISH_NNUE_PATH", path)
	err := EvidenceCheckNNUE()
	if err == nil || !strings.Contains(err.Error(), "rejected") {
		t.Fatalf("EvidenceCheckNNUE() = %v, want rejected error", err)
	}
}

// TestClassifyEvalMode_NNUE - verify() NNUE line is pretrained-model evidence
func TestClassifyEvalMode_NNUE(t *testing.T) {
	mode := ClassifyEvalMode([]string{
		"info string NNUE evaluation using /tmp/nn-3475407dc199.nnue enabled",
		"Final evaluation       +0.16 (white side) [with scaled NNUE, hybrid, ...]",
	})
	if mode != EvalModeNNUE {
		t.Fatalf("ClassifyEvalMode() = %q, want %q", mode, EvalModeNNUE)
	}
}

// TestClassifyEvalMode_Classical - verify() classical line is not pretrained evidence
func TestClassifyEvalMode_Classical(t *testing.T) {
	mode := ClassifyEvalMode([]string{"info string classical evaluation enabled"})
	if mode != EvalModeClassical {
		t.Fatalf("ClassifyEvalMode() = %q, want %q", mode, EvalModeClassical)
	}
}

// TestEvalProbe_ChessUsesNNUE - live UCI eval on Chess startpos uses the SF14 net
func TestEvalProbe_ChessUsesNNUE(t *testing.T) {
	bin, nnue := liveEvalAssets()
	if bin == "" || nnue == "" {
		t.Skip("Fairy-Stockfish binary or nn-3475407dc199.nnue not present")
	}
	t.Setenv("FAIRY_STOCKFISH_NNUE_PATH", nnue)
	fs, err := NewFairyStockfish(bin)
	if err != nil {
		t.Fatal(err)
	}
	if err := fs.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fs.Close() })
	mode, lines, err := fs.EvalProbe("")
	if err != nil {
		t.Fatalf("EvalProbe() error: %v lines=%v", err, lines)
	}
	if mode != EvalModeNNUE {
		t.Fatalf("EvalProbe chess mode = %q, want %q lines=%v", mode, EvalModeNNUE, lines)
	}
}

// TestEvalProbe_XiangqiAndShogiStayClassical - this Chess net does not apply to those variants
func TestEvalProbe_XiangqiAndShogiStayClassical(t *testing.T) {
	bin, nnue := liveEvalAssets()
	if bin == "" || nnue == "" {
		t.Skip("Fairy-Stockfish binary or nn-3475407dc199.nnue not present")
	}
	t.Setenv("FAIRY_STOCKFISH_NNUE_PATH", nnue)
	fs, err := NewFairyStockfish(bin)
	if err != nil {
		t.Fatal(err)
	}
	if err := fs.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fs.Close() })
	for _, variant := range []string{"xiangqi", "shogi"} {
		if err := fs.SetVariant(variant); err != nil {
			t.Fatalf("SetVariant(%s): %v", variant, err)
		}
		mode, lines, err := fs.EvalProbe("")
		if err != nil {
			t.Fatalf("EvalProbe(%s) error: %v lines=%v", variant, err, lines)
		}
		if mode != EvalModeClassical {
			t.Fatalf("EvalProbe %s mode = %q, want %q lines=%v", variant, mode, EvalModeClassical, lines)
		}
	}
}

// liveEvalAssets - FS binary and real SF14 net paths, or empty when missing
func liveEvalAssets() (string, string) {
	bin := strings.TrimSpace(os.Getenv("FAIRY_STOCKFISH_PATH"))
	if bin == "" || !fileExists(bin) {
		for _, cand := range []string{
			filepath.Join("..", "..", "..", "py_analyser", "Fairy-Stockfish-fairy_sf_14", "src", "stockfish"),
			filepath.Join("..", "..", "py_analyser", "Fairy-Stockfish-fairy_sf_14", "src", "stockfish"),
		} {
			if fileExists(cand) {
				bin = cand
				break
			}
		}
	}
	if !fileExists(bin) {
		bin = ""
	}
	nnue := strings.TrimSpace(os.Getenv("FAIRY_STOCKFISH_NNUE_PATH"))
	if nnue == "" || !fileExists(nnue) {
		for _, cand := range []string{
			filepath.Join("..", "..", "..", "..", "_local_nnue", DefaultNNUEFilename),
			filepath.Join("..", "..", "..", "_local_nnue", DefaultNNUEFilename),
		} {
			if fileExists(cand) {
				nnue = cand
				break
			}
		}
	}
	if !fileExists(nnue) {
		nnue = ""
	}
	return bin, nnue
}

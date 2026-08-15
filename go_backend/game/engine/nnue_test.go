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

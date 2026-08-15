// CM3070 FP code
// nnue.go - resolve and apply Fairy-Stockfish EvalFile / Use NNUE

package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Fairy-Stockfish 14 Chess default net (evaluate.h EvalFileDefaultName)
const DefaultNNUEFilename = "nn-3475407dc199.nnue"

// UCI eval used the neural net
const EvalModeNNUE = "nnue"

// UCI eval used hand-crafted evaluation
const EvalModeClassical = "classical"

// UCI eval did not print a verify() mode line
const EvalModeUnknown = "unknown"

// junk/placeholder files are smaller than a real SF14 net (~45 MiB)
const minNNUEEvidenceBytes int64 = 1_000_000

// ResolveNNUEPath - returns an existing .nnue path, or empty if none is configured or found
func ResolveNNUEPath() string {
	if p := strings.TrimSpace(os.Getenv("FAIRY_STOCKFISH_NNUE_PATH")); p != "" {
		if fileExists(p) {
			return p
		}
		return ""
	}
	for _, cand := range defaultNNUECandidates() {
		if fileExists(cand) {
			return cand
		}
	}
	return ""
}

// RequestedNNUEPath - FAIRY_STOCKFISH_NNUE_PATH as requested, even if the file is missing
func RequestedNNUEPath() string {
	return strings.TrimSpace(os.Getenv("FAIRY_STOCKFISH_NNUE_PATH"))
}

// EvidenceCheckNNUE - fails when the requested net is unset, missing, or too small to be the SF14 net
func EvidenceCheckNNUE() error {
	requested := RequestedNNUEPath()
	if requested == "" {
		return fmt.Errorf("evidence: FAIRY_STOCKFISH_NNUE_PATH unset; classical eval is not pretrained-model evidence")
	}
	st, err := os.Stat(requested)
	if err != nil || st.IsDir() {
		return fmt.Errorf("evidence: NNUE file missing: %s", requested)
	}
	base := filepath.Base(requested)
	if !strings.HasPrefix(base, "nn-") || !strings.HasSuffix(base, ".nnue") {
		return fmt.Errorf("evidence: NNUE file rejected (Chess net name must be nn-*.nnue): %s", base)
	}
	if st.Size() < minNNUEEvidenceBytes {
		return fmt.Errorf("evidence: NNUE file rejected (size %d < %d): %s", st.Size(), minNNUEEvidenceBytes, requested)
	}
	return nil
}

// ApplyNNUEFromEnv - sets Use NNUE and EvalFile when a weights file is available
func (fs *FairyStockfish) ApplyNNUEFromEnv() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if !fs.running {
		return fmt.Errorf("engine not running")
	}
	return fs.applyNNUELocked()
}

// ClassifyEvalMode - nnue if verify printed NNUE evaluation using, else classical
func ClassifyEvalMode(lines []string) string {
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "NNUE evaluation using") {
		return EvalModeNNUE
	}
	if strings.Contains(joined, "classical evaluation enabled") {
		return EvalModeClassical
	}
	return EvalModeUnknown
}

// EvalProbe - sends UCI eval and returns the verify() mode plus collected lines
func (fs *FairyStockfish) EvalProbe(fen string) (string, []string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if !fs.running {
		return "", nil, fmt.Errorf("engine not running")
	}
	pos := "position startpos"
	if strings.TrimSpace(fen) != "" {
		pos = fmt.Sprintf("position fen %s", fen)
	}
	if err := fs.send(pos); err != nil {
		return "", nil, err
	}
	if err := fs.send("eval"); err != nil {
		return "", nil, err
	}
	lines, err := fs.collectEvalLines(8 * time.Second)
	if err != nil {
		return "", lines, err
	}
	return ClassifyEvalMode(lines), lines, nil
}

// collectEvalLines - reads stdout until Final evaluation or timeout
func (fs *FairyStockfish) collectEvalLines(timeout time.Duration) ([]string, error) {
	deadline := time.Now().Add(timeout)
	var lines []string
	for time.Now().Before(deadline) {
		line, err := fs.stdout.ReadString('\n')
		if err != nil {
			return lines, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
		if strings.Contains(line, "Final evaluation") {
			return lines, nil
		}
	}
	return lines, fmt.Errorf("timeout waiting for Final evaluation")
}

// applyNNUELocked - UCI EvalFile / Use NNUE while fs.mu is held
func (fs *FairyStockfish) applyNNUELocked() error {
	path := ResolveNNUEPath()
	if path == "" {
		return nil
	}
	if err := fs.send("setoption name Use NNUE value true"); err != nil {
		return fmt.Errorf("Use NNUE: %w", err)
	}
	if err := fs.send(fmt.Sprintf("setoption name EvalFile value %s", path)); err != nil {
		return fmt.Errorf("EvalFile: %w", err)
	}
	if err := fs.send("isready"); err != nil {
		return err
	}
	return fs.waitFor("readyok", 15*time.Second)
}

// defaultNNUECandidates - local _local_nnue paths when FAIRY_STOCKFISH_NNUE_PATH is unset
func defaultNNUECandidates() []string {
	return []string{
		filepath.Join("..", "..", "_local_nnue", DefaultNNUEFilename),
		filepath.Join("..", "_local_nnue", DefaultNNUEFilename),
		filepath.Join(".", DefaultNNUEFilename),
	}
}

// fileExists - true when path is a regular file
func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

package engine

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	rulesMu sync.Mutex
	rulesFS *FairyStockfish
)

// DefaultBinaryPath resolves Fairy-Stockfish binary from env or known project layouts.
func DefaultBinaryPath() string {
	if p := os.Getenv("FAIRY_STOCKFISH_PATH"); p != "" {
		return p
	}
	candidates := []string{
		filepath.Join("..", "py_analyser", "Fairy-Stockfish-fairy_sf_14", "src", "stockfish"),
		filepath.Join("..", "..", "py_analyser", "Fairy-Stockfish-fairy_sf_14", "src", "stockfish"),
		filepath.Join("..", "..", "..", "py_analyser", "Fairy-Stockfish-fairy_sf_14", "src", "stockfish"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return candidates[0]
}

// RulesEngine returns a shared started Fairy-Stockfish process for legality / FEN updates.
func RulesEngine() (*FairyStockfish, error) {
	rulesMu.Lock()
	defer rulesMu.Unlock()
	if rulesFS != nil && rulesFS.IsRunning() {
		return rulesFS, nil
	}
	if rulesFS != nil {
		_ = rulesFS.Close()
		rulesFS = nil
	}
	fs, err := NewFairyStockfish(DefaultBinaryPath())
	if err != nil {
		return nil, err
	}
	if err := fs.Start(); err != nil {
		return nil, err
	}
	rulesFS = fs
	return rulesFS, nil
}

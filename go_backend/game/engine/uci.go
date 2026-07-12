package engine

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// UCIResult represents a parsed info line from the engine.
type UCIResult struct {
	Move   string
	Score  int // centipawn score, positive good for side to move
	PV     []string
	Depth  int
	MultiPV int // multipv index (1-based)
}

// FairyStockfish wraps a Fairy-Stockfish UCI process.
type FairyStockfish struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     *bufio.Reader
	stderr     io.ReadCloser
	mu         sync.Mutex
	running    bool
	binaryPath string
}

// NewFairyStockfish creates a new wrapper but does not start the process.
func NewFairyStockfish(binaryPath string) (*FairyStockfish, error) {
	if binaryPath == "" {
		return nil, errors.New("binaryPath must be provided")
	}
	return &FairyStockfish{
		binaryPath: binaryPath,
	}, nil
}

// Start launches the engine, sends "uci", and waits for "uciok".
func (fs *FairyStockfish) Start() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.running {
		return errors.New("engine already running")
	}

	fs.cmd = exec.Command(fs.binaryPath)
	var err error
	fs.stdin, err = fs.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdoutPipe, err := fs.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	fs.stdout = bufio.NewReader(stdoutPipe)
	fs.stderr, _ = fs.cmd.StderrPipe()

	if err := fs.cmd.Start(); err != nil {
		return fmt.Errorf("start engine: %w", err)
	}
	fs.running = true

	// Initialize UCI
	if err := fs.send("uci"); err != nil {
		fs.closeLocked()
		return err
	}
	if err := fs.waitFor("uciok", 5*time.Second); err != nil {
		fs.closeLocked()
		return err
	}
	return nil
}

// IsReady sends isready and waits for readyok.
func (fs *FairyStockfish) IsReady() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if !fs.running {
		return errors.New("engine not running")
	}
	if err := fs.send("isready"); err != nil {
		return err
	}
	return fs.waitFor("readyok", 2*time.Second)
}

// Restart stops and restarts the engine (useful after crashes or timeouts).
func (fs *FairyStockfish) Restart() error {
	_ = fs.Close()
	return fs.Start()
}

// SetOption sends setoption name ... value ...
func (fs *FairyStockfish) SetOption(name, value string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if !fs.running {
		return errors.New("engine not running")
	}
	cmd := fmt.Sprintf("setoption name %s value %s", name, value)
	return fs.send(cmd)
}

// SetStrengthProfile maps profile to UCI options (Skill Level + MultiPV).
func (fs *FairyStockfish) SetStrengthProfile(profile string) error {
	p := strings.ToLower(strings.TrimSpace(profile))
	var skill, multipv int

	switch p {
	case "beginner":
		skill, multipv = 0, 1
	case "intermediate":
		skill, multipv = 5, 3
	case "advanced":
		skill, multipv = 15, 3
	case "master":
		// MultiPV>1 is for suggestion UI; BestMove only needs one line and
		// MultiPV 5 made long eval runs crash more often (EOF / broken pipe).
		skill, multipv = 20, 1
	default:
		skill, multipv = 5, 3
	}

	if err := fs.SetOption("Skill Level", fmt.Sprintf("%d", skill)); err != nil {
		return err
	}
	if err := fs.SetOption("MultiPV", fmt.Sprintf("%d", multipv)); err != nil {
		return err
	}
	return nil
}

// BestMove sends position + go and returns the bestmove.
func (fs *FairyStockfish) BestMove(fen string, limit Limit) (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if !fs.running {
		return "", errors.New("engine not running")
	}

	if err := fs.send(fmt.Sprintf("position fen %s", fen)); err != nil {
		return "", err
	}
	goCmd := fs.buildGoCmd(limit)
	if err := fs.send(goCmd); err != nil {
		return "", err
	}

	wait := 10 * time.Second
	if limit.MoveTime > 0 {
		wait = limit.MoveTime + 5*time.Second
	}
	move, err := fs.waitForBestMove(wait)
	if err != nil {
		// Kill the process so a later Start/replace is clean (don't only flip the flag).
		_ = fs.closeLocked()
		return "", err
	}
	return move, nil
}

// TopK returns up to k best moves (with centipawn scores and PV) using MultiPV.
// It respects the current engine configuration (e.g. Skill Level set via SetStrengthProfile).
func (fs *FairyStockfish) TopK(fen string, k int, limit Limit) ([]UCIResult, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if !fs.running {
		return nil, errors.New("engine not running")
	}
	if k < 1 {
		k = 1
	}
	if k > 10 {
		k = 10 // safety cap
	}

	// Remember current MultiPV so we can restore it
	_ = fs.send(fmt.Sprintf("setoption name MultiPV value %d", k))

	if err := fs.send(fmt.Sprintf("position fen %s", fen)); err != nil {
		return nil, err
	}

	goCmd := fs.buildGoCmd(limit)
	if err := fs.send(goCmd); err != nil {
		return nil, err
	}

	results, err := fs.collectTopKResults(k, 12*time.Second)

	// Restore a reasonable default
	_ = fs.send("setoption name MultiPV value 3")

	if err != nil {
		return nil, err
	}
	return results, nil
}

// TopKWithProfile applies the strength profile first, then calls TopK.
func (fs *FairyStockfish) TopKWithProfile(fen string, k int, profile string, limit Limit) ([]UCIResult, error) {
	if err := fs.SetStrengthProfile(profile); err != nil {
		return nil, err
	}
	return fs.TopK(fen, k, limit)
}

// Close sends quit and kills the process.
func (fs *FairyStockfish) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.closeLocked()
}

// closeLocked assumes fs.mu is already held.
// Always tears down pipes/process if present, even when running was already false
// (e.g. after an EOF where BestMove cleared the flag).
func (fs *FairyStockfish) closeLocked() error {
	fs.running = false
	if fs.stdin != nil {
		_ = fs.stdin.Close()
		fs.stdin = nil
	}
	if fs.cmd != nil && fs.cmd.Process != nil {
		_ = fs.cmd.Process.Kill()
		_, _ = fs.cmd.Process.Wait()
		fs.cmd = nil
	}
	fs.stdout = nil
	return nil
}

// IsRunning reports whether the wrapper thinks the process is alive.
func (fs *FairyStockfish) IsRunning() bool {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.running
}

// --- internal helpers ---

func (fs *FairyStockfish) send(cmd string) error {
	if fs.stdin == nil {
		return errors.New("no stdin")
	}
	_, err := fmt.Fprintf(fs.stdin, "%s\n", cmd)
	return err
}

func (fs *FairyStockfish) waitFor(token string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s", token)
		default:
			line, err := fs.stdout.ReadString('\n')
			if err != nil {
				return err
			}
			line = strings.TrimSpace(line)
			if strings.Contains(line, token) {
				return nil
			}
		}
	}
}

func (fs *FairyStockfish) waitForBestMove(timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return "", errors.New("timeout waiting for bestmove")
		default:
			line, err := fs.stdout.ReadString('\n')
			if err != nil {
				return "", err
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "bestmove") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					return parts[1], nil
				}
				return "", errors.New("malformed bestmove")
			}
		}
	}
}

// collectTopKResults reads info lines until bestmove, parsing multipv results.
func (fs *FairyStockfish) collectTopKResults(k int, timeout time.Duration) ([]UCIResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	results := make([]UCIResult, 0, k)
	seen := make(map[int]UCIResult) // multipv index -> result

	for {
		select {
		case <-ctx.Done():
			for i := 1; i <= k; i++ {
				if r, ok := seen[i]; ok {
					results = append(results, r)
				}
			}
			return results, nil
		default:
			line, err := fs.stdout.ReadString('\n')
			if err != nil {
				return nil, err
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "bestmove") {
				for i := 1; i <= k; i++ {
					if r, ok := seen[i]; ok {
						results = append(results, r)
					}
				}
				return results, nil
			}
			if strings.HasPrefix(line, "info") {
				res := parseInfoLine(line)
				if res != nil && res.Move != "" {
					if res.PV != nil && len(res.PV) > 0 {
						res.Move = res.PV[0]
					}
					if res.Depth == 0 {
						res.Depth = 1
					}
					// Prefer the real multipv index if present
					idx := res.MultiPV
					if idx < 1 || idx > k {
						idx = len(seen) + 1
					}
					if idx <= k {
						seen[idx] = *res
					}
				}
			}
		}
	}
}

// parseInfoLine extracts score cp, pv, depth, and multipv index from an info line.
func parseInfoLine(line string) *UCIResult {
	res := &UCIResult{}
	fields := strings.Fields(line)

	for i := 0; i < len(fields); i++ {
		switch fields[i] {
		case "depth":
			if i+1 < len(fields) {
				fmt.Sscanf(fields[i+1], "%d", &res.Depth)
			}
		case "score":
			if i+2 < len(fields) && fields[i+1] == "cp" {
				fmt.Sscanf(fields[i+2], "%d", &res.Score)
			}
		case "multipv":
			if i+1 < len(fields) {
				fmt.Sscanf(fields[i+1], "%d", &res.MultiPV)
			}
		case "pv":
			res.PV = fields[i+1:]
			if len(res.PV) > 0 {
				res.Move = res.PV[0]
			}
		}
	}

	if res.Move == "" && res.PV == nil {
		return nil
	}
	return res
}

func (fs *FairyStockfish) buildGoCmd(limit Limit) string {
	// Prefer movetime when set so searches finish before waitForBestMove's deadline.
	// (Depth-only go depth 20 routinely exceeds the 10s bestmove wait on master.)
	if limit.MoveTime > 0 {
		return fmt.Sprintf("go movetime %d", limit.MoveTime.Milliseconds())
	}
	if limit.Depth > 0 {
		return fmt.Sprintf("go depth %d", limit.Depth)
	}
	return "go depth 8"
}

// Limit mirrors chess/engine.Limit style for compatibility.
type Limit struct {
	Depth    int
	MoveTime time.Duration
	Nodes    int64
}
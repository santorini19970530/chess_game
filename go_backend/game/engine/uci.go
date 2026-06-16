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
	Move  string
	Score int // centipawn score, positive good for side to move
	PV    []string
	Depth int
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
		fs.Close()
		return err
	}
	if err := fs.waitFor("uciok", 5*time.Second); err != nil {
		fs.Close()
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
		skill, multipv = 20, 5
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

	move, err := fs.waitForBestMove(10 * time.Second)
	if err != nil {
		return "", err
	}
	return move, nil
}

// TopK returns up to k best moves with scores by setting MultiPV.
func (fs *FairyStockfish) TopK(fen string, k int, limit Limit) ([]UCIResult, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if !fs.running {
		return nil, errors.New("engine not running")
	}
	if k < 1 {
		k = 1
	}

	// Enable MultiPV for this search
	if err := fs.send(fmt.Sprintf("setoption name MultiPV value %d", k)); err != nil {
		return nil, err
	}
	if err := fs.send(fmt.Sprintf("position fen %s", fen)); err != nil {
		return nil, err
	}
	goCmd := fs.buildGoCmd(limit)
	if err := fs.send(goCmd); err != nil {
		return nil, err
	}

	results, err := fs.collectTopKResults(k, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// Close sends quit and kills the process.
func (fs *FairyStockfish) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if !fs.running {
		return nil
	}
	_ = fs.send("quit")
	fs.running = false
	if fs.stdin != nil {
		fs.stdin.Close()
	}
	if fs.cmd != nil && fs.cmd.Process != nil {
		fs.cmd.Process.Kill()
	}
	return nil
}

// Restart attempts to stop and restart the engine (for crash recovery).
func (fs *FairyStockfish) Restart() error {
	_ = fs.Close()
	time.Sleep(100 * time.Millisecond)
	return fs.Start()
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
			// return what we have so far
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
				// finalize
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
					// assume multipv index or just append first k unique
					idx := len(seen) + 1
					if idx <= k {
						seen[idx] = *res
					}
				}
			}
		}
	}
}

// parseInfoLine extracts score cp, pv, depth from an info line.
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
	if limit.Depth > 0 {
		return fmt.Sprintf("go depth %d", limit.Depth)
	}
	if limit.MoveTime > 0 {
		return fmt.Sprintf("go movetime %d", limit.MoveTime.Milliseconds())
	}
	// default
	return "go depth 8"
}

// Limit mirrors chess/engine.Limit style for compatibility.
type Limit struct {
	Depth    int
	MoveTime time.Duration
	Nodes    int64
}
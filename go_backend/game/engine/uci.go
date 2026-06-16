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

// SetStrengthProfile maps profile to UCI options (Skill Level + limits).
func (fs *FairyStockfish) SetStrengthProfile(profile string) error {
	profile = strings.ToLower(profile)
	var skill int
	switch profile {
	case "beginner":
		skill = 0
	case "intermediate":
		skill = 5
	case "advanced":
		skill = 15
	case "master":
		skill = 20
	default:
		skill = 5
	}
	if err := fs.SetOption("Skill Level", fmt.Sprintf("%d", skill)); err != nil {
		return err
	}
	// Additional options could be set here (e.g. Hash, Threads)
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

// TopK is a placeholder that currently returns only the best move (full multi-pv requires more parsing).
func (fs *FairyStockfish) TopK(fen string, k int, limit Limit) ([]UCIResult, error) {
	// For step 1 we keep simple; later can implement MultiPV
	best, err := fs.BestMove(fen, limit)
	if err != nil {
		return nil, err
	}
	return []UCIResult{{Move: best}}, nil
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
			// TODO: parse info lines for score / pv into internal state if needed
		}
	}
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
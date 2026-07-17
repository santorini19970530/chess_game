package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	session "go_backend/game/session"
	"go_backend/handlers"
	"go_backend/simulation"
)

func main() {
	games := flag.Int("games", 0, "number of games to simulate (required, >=1)")
	profile := flag.String("profile", "", "AI strength for both sides: beginner|intermediate|advanced|master")
	whiteProfile := flag.String("white-profile", "", "White AI strength (overrides -profile for White)")
	blackProfile := flag.String("black-profile", "", "Black AI strength (overrides -profile for Black)")
	gameType := flag.String("game", "chess", "game type: chess|xianqi|shogi")
	format := flag.String("format", "text", "output format: text|json|csv")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: match [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Run N AI vs AI games and print summary.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  match -games 5 -profile intermediate\n")
		fmt.Fprintf(os.Stderr, "  match -games 10 -white-profile beginner -black-profile master -format json\n")
		fmt.Fprintf(os.Stderr, "  match -games 5 -game xianqi -profile beginner -format json\n")
		fmt.Fprintf(os.Stderr, "  match -games 5 -game shogi -profile beginner -format json\n")
		fmt.Fprintf(os.Stderr, "  match -games 20 -format csv > results.csv\n")
	}
	flag.Parse()

	if *games < 1 {
		log.Fatal("error: --games must be >= 1")
	}
	gt, err := parseMatchGameType(*gameType)
	if err != nil {
		log.Fatal(err)
	}

	white, black, err := resolveMatchProfiles(*profile, *whiteProfile, *blackProfile)
	if err != nil {
		log.Fatal(err)
	}

	type GameResult struct {
		Result     string `json:"result"`
		Winner     string `json:"winner,omitempty"`
		Moves      int    `json:"moves"`
		DurationMs int64  `json:"duration_ms"`
		AvgMoveMs  int64  `json:"avg_move_ms"`
	}

	var results []GameResult
	var archiveItems []simulation.ResultWithGameID
	var durations []int64
	var whiteWins, blackWins, draws, totalMoves int

	for i := 0; i < *games; i++ {
		gameNum := i + 1
		log.Printf("=== Game %d/%d started (game=%s white=%s black=%s) ===", gameNum, *games, gt, white, black)

		game, err := session.CreateGame(session.GameModeAIVsAI, gt, "white", 1, "", white)
		if err != nil {
			log.Fatalf("failed to create game: %v", err)
		}
		if _, err := session.SetAISideProfilesByID(game.ID, white, black); err != nil {
			log.Fatalf("failed to set side profiles: %v", err)
		}

		start := time.Now()
		res, err := simulation.RunSingleAIGame(game.ID, handlers.SelectAIMove)
		if err != nil {
			log.Fatalf("simulation failed: %v", err)
		}
		durationMs := time.Since(start).Milliseconds()
		avgMoveMs := simulation.ComputeAvgMoveMs(durationMs, res.MoveCount)
		durations = append(durations, durationMs)

		log.Printf("=== Game %d/%d finished: result=%s winner=%q moves=%d duration_ms=%d ===",
			gameNum, *games, res.Result, res.Winner, res.MoveCount, durationMs)

		switch res.Result {
		case session.GameResultWhiteWin:
			whiteWins++
		case session.GameResultBlackWin:
			blackWins++
		case session.GameResultDraw:
			draws++
		}
		totalMoves += res.MoveCount

		results = append(results, GameResult{
			Result:     string(res.Result),
			Winner:     res.Winner,
			Moves:      res.MoveCount,
			DurationMs: durationMs,
			AvgMoveMs:  avgMoveMs,
		})

		item := simulation.ResultWithGameID{
			GameID:          game.ID,
			GameType:        *gameType,
			WhiteProfile:    white,
			BlackProfile:    black,
			Result:          res.Result,
			Winner:          res.Winner,
			MoveCount:       res.MoveCount,
			DurationMs:      durationMs,
			AvgMoveMs:       avgMoveMs,
			HistoryDetailed: res.HistoryDetailed,
		}
		if white == black {
			item.Profile = white
		}
		archiveItems = append(archiveItems, item)
	}

	if err := simulation.ArchiveSimulationRun(archiveItems); err != nil {
		log.Printf("warning: failed to archive simulation run: %v", err)
	}

	avg := 0.0
	if *games > 0 {
		avg = float64(totalMoves) / float64(*games)
	}
	avgDuration := simulation.MeanMs(durations)
	p95Duration := simulation.PercentileMs(durations, 95)

	if *format == "json" {
		out := struct {
			Games         int          `json:"games"`
			GameType      string       `json:"game_type"`
			WhiteProfile  string       `json:"white_profile"`
			BlackProfile  string       `json:"black_profile"`
			Profile       string       `json:"profile,omitempty"`
			WhiteWins     int          `json:"white_wins"`
			BlackWins     int          `json:"black_wins"`
			Draws         int          `json:"draws"`
			AvgMoves      float64      `json:"avg_moves"`
			AvgDurationMs float64      `json:"avg_duration_ms"`
			P95DurationMs int64        `json:"p95_duration_ms"`
			Results       []GameResult `json:"results"`
		}{
			Games:         *games,
			GameType:      *gameType,
			WhiteProfile:  white,
			BlackProfile:  black,
			WhiteWins:     whiteWins,
			BlackWins:     blackWins,
			Draws:         draws,
			AvgMoves:      avg,
			AvgDurationMs: avgDuration,
			P95DurationMs: p95Duration,
			Results:       results,
		}
		if white == black {
			out.Profile = white
		}
		json.NewEncoder(os.Stdout).Encode(out)
	} else if *format == "csv" {
		fmt.Println("game,result,winner,moves,duration_ms,avg_move_ms,white_profile,black_profile")
		for i, r := range results {
			fmt.Printf("%d,%s,%s,%d,%d,%d,%s,%s\n", i+1, r.Result, r.Winner, r.Moves, r.DurationMs, r.AvgMoveMs, white, black)
		}
		fmt.Printf("# Summary,%d games,White %d,Black %d,Draws %d,AvgMoves %.1f,AvgDurationMs %.1f,P95DurationMs %d,white=%s,black=%s\n",
			*games, whiteWins, blackWins, draws, avg, avgDuration, p95Duration, white, black)
	} else {
		fmt.Println()
		fmt.Printf("Summary: %d games | white=%s black=%s | White %d | Black %d | Draws %d | Avg moves: %.1f | Avg duration: %.1fms | P95: %dms\n",
			*games, white, black, whiteWins, blackWins, draws, avg, avgDuration, p95Duration)
	}
}

func resolveMatchProfiles(profile, whiteRaw, blackRaw string) (white, black string, err error) {
	profile = strings.TrimSpace(profile)
	whiteRaw = strings.TrimSpace(whiteRaw)
	blackRaw = strings.TrimSpace(blackRaw)

	fallback := "intermediate"
	if profile != "" {
		parsed, ok := session.ParseAIProfile(profile)
		if !ok {
			return "", "", fmt.Errorf("error: invalid profile %q", profile)
		}
		fallback = parsed
	}

	white = fallback
	black = fallback
	if whiteRaw != "" {
		parsed, ok := session.ParseAIProfile(whiteRaw)
		if !ok {
			return "", "", fmt.Errorf("error: invalid white-profile %q", whiteRaw)
		}
		white = parsed
	}
	if blackRaw != "" {
		parsed, ok := session.ParseAIProfile(blackRaw)
		if !ok {
			return "", "", fmt.Errorf("error: invalid black-profile %q", blackRaw)
		}
		black = parsed
	}
	return white, black, nil
}

func parseMatchGameType(raw string) (session.GameType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "chess":
		return session.GameTypeChess, nil
	case "xianqi", "xiangqi":
		return session.GameTypeXiangqi, nil
	case "shogi":
		return session.GameTypeShogi, nil
	default:
		return "", fmt.Errorf("error: unsupported game %q (chess, xianqi, or shogi)", raw)
	}
}

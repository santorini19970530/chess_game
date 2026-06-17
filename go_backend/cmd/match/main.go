package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	session "go_backend/game/session"
	"go_backend/handlers"
	"go_backend/simulation"
)

func main() {
	games := flag.Int("games", 0, "number of games to simulate (required, >=1)")
	profile := flag.String("profile", "intermediate", "AI strength: beginner|intermediate|advanced|master")
	format := flag.String("format", "text", "output format: text|json|csv")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: match [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Run N AI vs AI games and print summary.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  match -games 5 -profile intermediate\n")
		fmt.Fprintf(os.Stderr, "  match -games 10 -profile master -format json\n")
		fmt.Fprintf(os.Stderr, "  match -games 20 -format csv > results.csv\n")
	}
	flag.Parse()

	if *games < 1 {
		log.Fatal("error: --games must be >= 1")
	}

	validProfiles := map[string]bool{"beginner": true, "intermediate": true, "advanced": true, "master": true}
	if !validProfiles[*profile] {
		log.Fatalf("error: invalid profile %q (use beginner|intermediate|advanced|master)", *profile)
	}

	type GameResult struct {
		Result string `json:"result"`
		Winner string `json:"winner,omitempty"`
		Moves  int    `json:"moves"`
	}

	var results []GameResult
	var archiveItems []simulation.ResultWithGameID
	var white, black, draws, totalMoves int

	for i := 0; i < *games; i++ {
		gameNum := i + 1
		log.Printf("=== Game %d/%d started (profile=%s) ===", gameNum, *games, *profile)

		game, err := session.CreateGame(session.GameModeAIVsAI, session.GameTypeChess, "white", 1, "", *profile)
		if err != nil {
			log.Fatalf("failed to create game: %v", err)
		}

		res, err := simulation.RunSingleAIGame(game.ID, handlers.SelectAIMove)
		if err != nil {
			log.Fatalf("simulation failed: %v", err)
		}

		log.Printf("=== Game %d/%d finished: result=%s winner=%q moves=%d ===",
			gameNum, *games, res.Result, res.Winner, res.MoveCount)

		switch res.Result {
		case session.GameResultWhiteWin:
			white++
		case session.GameResultBlackWin:
			black++
		case session.GameResultDraw:
			draws++
		}
		totalMoves += res.MoveCount

		results = append(results, GameResult{
			Result: string(res.Result),
			Winner: res.Winner,
			Moves:  res.MoveCount,
		})

		archiveItems = append(archiveItems, simulation.ResultWithGameID{
			GameID:          game.ID,
			Profile:         *profile,
			Result:          res.Result,
			Winner:          res.Winner,
			MoveCount:       res.MoveCount,
			HistoryDetailed: res.HistoryDetailed,
		})

		if *format == "text" {
			// Per-move logs come from the session layer; we only print game boundaries here
		}
	}

	// Archive results (same behaviour as /api/simulate)
	if err := simulation.ArchiveSimulationRun(archiveItems); err != nil {
		log.Printf("warning: failed to archive simulation run: %v", err)
	}

	avg := 0.0
	if *games > 0 {
		avg = float64(totalMoves) / float64(*games)
	}

	if *format == "json" {
		out := struct {
			Games     int          `json:"games"`
			Profile   string       `json:"profile"`
			WhiteWins int          `json:"white_wins"`
			BlackWins int          `json:"black_wins"`
			Draws     int          `json:"draws"`
			AvgMoves  float64      `json:"avg_moves"`
			Results   []GameResult `json:"results"`
		}{
			Games:     *games,
			Profile:   *profile,
			WhiteWins: white,
			BlackWins: black,
			Draws:     draws,
			AvgMoves:  avg,
			Results:   results,
		}
		json.NewEncoder(os.Stdout).Encode(out)
	} else if *format == "csv" {
		fmt.Println("game,result,winner,moves")
		for i, r := range results {
			fmt.Printf("%d,%s,%s,%d\n", i+1, r.Result, r.Winner, r.Moves)
		}
		fmt.Printf("# Summary,%d games,White %d,Black %d,Draws %d,Avg %.1f\n",
			*games, white, black, draws, avg)
	} else {
		fmt.Println()
		fmt.Printf("Summary: %d games | White %d | Black %d | Draws %d | Avg moves: %.1f\n",
			*games, white, black, draws, avg)
	}
}
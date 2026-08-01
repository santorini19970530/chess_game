// CM3070 FP code
// analyzer_types.go - analyzer request/response types and store state

package handlers

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	sessionpkg "go_backend/game/session"
)

// http client for the analyzer
var pyAnalyzerHTTPClient = &http.Client{
	Timeout: 0,
}

// request to the analyzer
type analyzerRequest struct {
	RequestID string `json:"request_id"`
	FEN       string `json:"fen"`
	Color     string `json:"color"`
	TopK      int    `json:"top_k"`
	GameType  string `json:"game_type,omitempty"`
}

// explainRequest mirrors the payload expected by Python /explain
type explainRequest struct {
	RequestID    string   `json:"request_id"`
	FEN          string   `json:"fen"`
	Color        string   `json:"color"` // side to move AFTER the explained move
	GameType     string   `json:"game_type"`
	SkillLevel   string   `json:"skill_level,omitempty"`
	HumanColor   string   `json:"human_color,omitempty"` // human seat in HvAI (white|black)
	ConceptHints []string `json:"concept_hints,omitempty"`
	MoveUCI      string   `json:"move_uci,omitempty"`
	MoveSAN      string   `json:"move_san,omitempty"`
	MoveHistory  []string `json:"move_history,omitempty"`
	Quick        bool     `json:"quick,omitempty"` // instant ground-truth line (no Ollama)
}

// explainSkillLevelFromProfile - maps AI strength (4 levels) → explain skill_level (3)
func explainSkillLevelFromProfile(profile string) string {
	return sessionpkg.SkillLevelFromAIProfile(profile)
}

// conceptHintsFromAnalysis - builds at most 3 short cues for the explain prompt (issue0047). missing fields are skipped; never blocks explain
func conceptHintsFromAnalysis(a analyzerResponse) []string {
	hints := make([]string, 0, 3)
	if t := strings.TrimSpace(a.ThreatSummary); t != "" {
		hints = append(hints, t)
	}

	balanceOK := false
	if a.HealthSummary != nil {
		if raw, ok := a.HealthSummary["material_balance_white_minus_black"]; ok {
			switch v := raw.(type) {
			case float64:
				balanceOK = true
				if v >= 100 {
					hints = append(hints, "White is ahead on material / evaluation.")
				} else if v <= -100 {
					hints = append(hints, "Black is ahead on material / evaluation.")
				}
			case int:
				balanceOK = true
				if v >= 100 {
					hints = append(hints, "White is ahead on material / evaluation.")
				} else if v <= -100 {
					hints = append(hints, "Black is ahead on material / evaluation.")
				}
			}
		}
	}
	if !balanceOK {
		if a.EvalCPWhite >= 100 {
			hints = append(hints, "White is ahead on material / evaluation.")
		} else if a.EvalCPWhite <= -100 {
			hints = append(hints, "Black is ahead on material / evaluation.")
		}
	}

	labels := make([]string, 0, 3)
	for i, sm := range a.SuggestedMoves {
		if i >= 3 {
			break
		}
		lab := strings.TrimSpace(sm.SAN)
		if lab == "" {
			lab = strings.TrimSpace(sm.UCI)
		}
		if lab != "" {
			labels = append(labels, lab)
		}
	}
	if len(labels) == 0 {
		if best := strings.TrimSpace(a.BestMoveUCI); best != "" {
			labels = append(labels, best)
		}
	}
	if len(labels) > 0 {
		hints = append(hints, "Engine suggested replies (side to move): "+strings.Join(labels, ", ")+".")
	}
	if len(hints) > 3 {
		hints = hints[:3]
	}
	return hints
}

// conceptHintsForExplain - only uses analysis for this exact FEN. stale MultiPV from the previous ply must not cue the coach (mismatch with Suggested moves)
func conceptHintsForExplain(latest latestAnalysisState, fen string) []string {
	analysisFEN := strings.TrimSpace(latest.Analysis.FEN)
	if analysisFEN == "" || analysisFEN != strings.TrimSpace(fen) {
		return nil
	}
	return conceptHintsFromAnalysis(latest.Analysis)
}

// explainHintWait - returns how long explain may wait for same-fen analysis cues (default 0)
func explainHintWait() time.Duration {
	raw := strings.TrimSpace(os.Getenv("EXPLAIN_HINT_WAIT_MS"))
	if raw == "" {
		return 0
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms <= 0 {
		return 0
	}
	if ms > 15000 {
		ms = 15000
	}
	return time.Duration(ms) * time.Millisecond
}

// waitConceptHintsForFEN - peeks (and optionally polls) for same-FEN analysis cues. never blocks the game HTTP thread — only the explain goroutine
func waitConceptHintsForFEN(gameID, fen string, timeout time.Duration) []string {
	fen = strings.TrimSpace(fen)
	peek := func() []string {
		latest, ok := getLatestAnalysisByGameID(gameID)
		if !ok {
			return nil
		}
		return conceptHintsForExplain(latest, fen)
	}
	if timeout <= 0 {
		return peek()
	}
	deadline := time.Now().Add(timeout)
	for {
		if hints := peek(); hints != nil {
			return hints
		}
		if time.Now().After(deadline) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// explainResponse is the JSON shape returned by Python /explain.
type explainResponse struct {
	RequestID    string   `json:"request_id"`
	Status       string   `json:"status"`
	Source       string   `json:"source"`
	Explanation  string   `json:"explanation"`
	MoveUCI      string   `json:"move_uci"`
	MoveSAN      string   `json:"move_san"`
	LatencyMS    int      `json:"latency_ms"`
	ConceptHints []string `json:"concept_hints,omitempty"`
}

// job to analyze the current position
type analysisJob struct {
	GameID          string
	MoveNumber      int
	Command         string
	Request         analyzerRequest
	EnqueueQueueLen int
}

// suggested move from the analyzer
type analyzerSuggestedMove struct {
	Rank  int    `json:"rank"`
	UCI   string `json:"uci"`
	SAN   string `json:"san"`
	Score int    `json:"score"`
}

// response from the analyzer
type analyzerResponse struct {
	RequestID      string                  `json:"request_id"`
	Status         string                  `json:"status"`
	Source         string                  `json:"source"`
	FEN            string                  `json:"fen"`
	EvaluatedColor string                  `json:"evaluated_for_color"`
	HealthSummary  map[string]interface{}  `json:"health_summary"`
	IsCheck        bool                    `json:"is_check"`
	IsCheckmate    bool                    `json:"is_checkmate"`
	IsStalemate    bool                    `json:"is_stalemate"`
	EvalCPWhite    int                     `json:"eval_cp_white"`
	WinChanceWhite float64                 `json:"win_chance_white"`
	WinChanceBlack float64                 `json:"win_chance_black"`
	ThreatSummary  string                  `json:"threat_summary"`
	BestMoveUCI    string                  `json:"best_move_uci"`
	SuggestedMoves []analyzerSuggestedMove `json:"suggested_moves"`
	LatencyMS      int                     `json:"latency_ms"`
}

// record of the move analysis
type moveAnalysisRecord struct {
	MoveNumber int              `json:"move_number"`
	Command    string           `json:"command"`
	Analysis   analyzerResponse `json:"analysis"`
}

// moveExplanationRecord is coach evidence for report / QA
type moveExplanationRecord struct {
	MoveNumber   int      `json:"move_number"`
	MoveUCI      string   `json:"move_uci"`
	MoveSAN      string   `json:"move_san,omitempty"`
	FEN          string   `json:"fen"`
	SkillLevel   string   `json:"skill_level"`
	Source       string   `json:"source"`
	Explanation  string   `json:"explanation"`
	ConceptHints []string `json:"concept_hints,omitempty"`
	LatencyMS    int      `json:"latency_ms"`
	RecordedAt   string   `json:"recorded_at"`
}

// latest analysis state
type latestAnalysisState struct {
	GameID     string           `json:"game_id"`
	MoveNumber int              `json:"move_number"`
	Command    string           `json:"command"`
	Analysis   analyzerResponse `json:"analysis"`
	UpdatedAt  string           `json:"updated_at"`
}

// latest analysis status
type analysisLatestStatus struct {
	GameID              string               `json:"game_id"`
	RequestedMoveNumber int                  `json:"requested_move_number"`
	LatestMoveNumber    int                  `json:"latest_move_number"`
	Pending             bool                 `json:"pending"`
	LastError           string               `json:"last_error,omitempty"`
	Latest              *latestAnalysisState `json:"latest,omitempty"`
}

// error from the analyzer call
type analyzerCallError struct {
	Kind       string
	HTTPStatus int
	Err        error
}

// Error - returns the error message
func (e *analyzerCallError) Error() string {
	if e == nil || e.Err == nil {
		return "analyzer call error"
	}
	return e.Err.Error()
}

// Unwrap - returns the wrapped error
func (e *analyzerCallError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// log event for the analyzer
type analysisLogEvent struct {
	Event               string `json:"event"`
	GameID              string `json:"game_id"`
	MoveNumber          int    `json:"move_number"`
	RequestID           string `json:"request_id"`
	QueueLen            int    `json:"queue_len"`
	Pending             bool   `json:"pending"`
	Success             bool   `json:"success"`
	LatencyMS           int64  `json:"latency_ms"`
	ErrorKind           string `json:"error_kind"`
	ErrorMessageSafe    string `json:"error_message_safe"`
	TimestampUTC        string `json:"timestamp_utc"`
	IsStale             bool   `json:"is_stale"`
	LatestRequestedMove int    `json:"latest_requested_move"`
	HTTPStatus          int    `json:"http_status,omitempty"`
	AnalyzerSource      string `json:"analyzer_source,omitempty"`
	AnalyzerLatencyMS   int    `json:"analyzer_latency_ms,omitempty"`
	BestMoveUCI         string `json:"best_move_uci,omitempty"`
}

// global variables for the analyzer
var (
	moveAnalysisByGame      = map[string][]moveAnalysisRecord{}
	moveExplanationByGame   = map[string][]moveExplanationRecord{}
	latestAnalysisByGame    = map[string]latestAnalysisState{}
	latestRequestedByGame   = map[string]int{}
	analysisPendingByGame   = map[string]bool{}
	analysisLastErrorByGame = map[string]string{}
	exportedGames           = map[string]bool{}
	explainLatestByGame     = map[string]int{} // coalesce Ollama: only newest ply runs
	analysisStoreMu         sync.Mutex
	analysisQueue           = make(chan analysisJob, 128)
	analysisWorkerOnce      sync.Once
	// one Ollama /explain at a time — unbounded goroutines melt the machine late-game.
	explainOllamaSem = make(chan struct{}, 1)
)

// error kinds for the analyzer
const (
	analysisErrorKindNone        = "none"
	analysisErrorKindTimeout     = "timeout"
	analysisErrorKindUnavailable = "unavailable"
	analysisErrorKindBadStatus   = "bad_status"
	analysisErrorKindBadJSON     = "bad_json"
	analysisErrorKindOther       = "other"
)

// CM3070 FP code
// index.go - game playing page and state response

package handlers

import (
	chessboard "go_backend/game/board"
	sessionpkg "go_backend/game/session"
	"html/template"
	"log"
	"net/http"
	"strings"
)

// gameStateResponse - status of the game that share with the frontend
type gameStateResponse struct {
	CurrentTurn     string                        `json:"currentTurn"`
	CheckedSide     string                        `json:"checkedSide"`
	Game            sessionpkg.GameSession        `json:"game"`
	Captured        sessionpkg.CapturedSummary    `json:"captured"`
	Analysis        *analyzerResponse             `json:"analysis,omitempty"`
	History         []string                      `json:"history"`
	HistoryDetailed []sessionpkg.MoveHistoryEntry `json:"historyDetailed"`
	State           []sessionpkg.PieceState       `json:"state"`
}

// generateChessBoard - builds the chessboard html for the index page
func generateChessBoard() template.HTML {
	return DrawChessBoard(chessboard.NewChessBoard())
}

// Index - renders the main game page template
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	// reject non-root paths
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// parse the base page and reusable partial templates
	t, err := template.ParseFiles(
		"../frontend/index.html",
		"../frontend/html_puzzles/head.html",
		"../frontend/html_puzzles/header.html",
		"../frontend/html_puzzles/footer.html",
	)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("index template parse error: %v", err)
		return
	}

	// build dynamic main content html in sequence
	var mainHTMLCode strings.Builder
	currentTurnLabel := sessionpkg.CurrentTurnLabel()
	whiteTurnClass := "game_info_col_white"
	blackTurnClass := "game_info_col_black"
	if currentTurnLabel == "White" {
		whiteTurnClass += " game_info_col_active"
	} else {
		blackTurnClass += " game_info_col_active"
	}

	// left panel
	mainHTMLCode.WriteString(`<div class="game_panel">`)

	mainHTMLCode.WriteString(`<div class="game_panel_config">`)
	mainHTMLCode.WriteString(`<details id="game_config_details" class="game_config_details" open>`)
	mainHTMLCode.WriteString(`<summary class="config_panel_title">Setup new game</summary>`)
	mainHTMLCode.WriteString(`<label for="game_type">Game</label>`)
	mainHTMLCode.WriteString(`<select id="game_type"><option value="chess">Chess</option><option value="xianqi">Xiangqi</option><option value="shogi">Shogi</option></select>`)
	mainHTMLCode.WriteString(`<label for="game_mode">Mode</label>`)
	mainHTMLCode.WriteString(`<select id="game_mode"><option value="human_vs_human">Human vs Human</option><option value="human_vs_ai">Human vs AI</option><option value="ai_vs_ai">AI vs AI</option></select>`)
	mainHTMLCode.WriteString(`<label for="human_side">Human's side</label>`)
	mainHTMLCode.WriteString(`<select id="human_side"><option value="white">White</option><option value="black">Black</option></select>`)
	mainHTMLCode.WriteString(`<label for="ai_game_count">AI game count</label>`)
	mainHTMLCode.WriteString(`<input id="ai_game_count" type="number" min="1" value="1" />`)
	mainHTMLCode.WriteString(`<label for="ai_strength">AI strength</label>`)
	mainHTMLCode.WriteString(`<select id="ai_strength"><option value="beginner">Beginner</option><option value="intermediate" selected>Intermediate</option><option value="advanced">Advanced</option><option value="master">Master</option></select>`)
	mainHTMLCode.WriteString(`<label for="coach_level">Coach level</label>`)
	mainHTMLCode.WriteString(`<select id="coach_level" title="Explanation skill level (independent of AI strength)"><option value="beginner">Beginner</option><option value="intermediate" selected>Intermediate</option><option value="advanced">Advanced</option></select>`)
	mainHTMLCode.WriteString(`<label for="fen_input">Starting FEN (optional)</label>`)
	mainHTMLCode.WriteString(`<textarea id="fen_input" rows="3" placeholder="rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"></textarea>`)
	// clock setup: seconds in the form; JS converts to ms for the API.
	mainHTMLCode.WriteString(`<div class="config_clock_enable">`)
	mainHTMLCode.WriteString(`<label for="clock_enabled">Clock</label>`)
	mainHTMLCode.WriteString(`<input id="clock_enabled" type="checkbox" title="Off = unlimited time" />`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<label for="clock_preset">Time control</label>`)
	mainHTMLCode.WriteString(`<select id="clock_preset" title="Presets are base|increment (minutes|seconds per move)">`)
	mainHTMLCode.WriteString(`<option value="5|0">5|0</option>`)
	mainHTMLCode.WriteString(`<option value="10|0">10|0</option>`)
	mainHTMLCode.WriteString(`<option value="15|10">15|10</option>`)
	mainHTMLCode.WriteString(`<option value="5|30" selected>5|30</option>`)
	mainHTMLCode.WriteString(`<option value="custom">Custom</option>`)
	mainHTMLCode.WriteString(`</select>`)
	mainHTMLCode.WriteString(`<label for="clock_base_sec">Base (seconds / side)</label>`)
	mainHTMLCode.WriteString(`<input id="clock_base_sec" type="number" min="0" step="1" value="300" />`)
	mainHTMLCode.WriteString(`<label for="clock_increment_sec">Increment (seconds / move)</label>`)
	mainHTMLCode.WriteString(`<input id="clock_increment_sec" type="number" min="0" step="1" value="30" />`)
	mainHTMLCode.WriteString(`<div id="clock_hvai_fields">`)
	mainHTMLCode.WriteString(`<label for="clock_human_base_sec">Human base (seconds)</label>`)
	mainHTMLCode.WriteString(`<input id="clock_human_base_sec" type="number" min="0" step="1" value="300" />`)
	mainHTMLCode.WriteString(`<label for="clock_ai_base_sec">AI base (seconds)</label>`)
	mainHTMLCode.WriteString(`<input id="clock_ai_base_sec" type="number" min="0" step="1" value="60" />`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<button id="game_config_apply" type="button">Apply Setup</button>`)
	mainHTMLCode.WriteString(`</details>`)
	mainHTMLCode.WriteString(`</div>`)

	mainHTMLCode.WriteString(`<div class="game_panel_left">`)
	mainHTMLCode.WriteString(string(generateChessBoard()))
	mainHTMLCode.WriteString(`</div>`)

	// right panel
	mainHTMLCode.WriteString(`<div class="game_panel_right_top" style="display:flex;flex-direction:column;min-height:0;overflow:hidden;">`)
	mainHTMLCode.WriteString(`<div class="game_info_table" role="table" aria-label="Game information table">`)
	mainHTMLCode.WriteString(`<div class="game_info_winprob_wrapper" role="presentation">`)
	mainHTMLCode.WriteString(`<div class="game_info_winprob_track">`)
	mainHTMLCode.WriteString(`<div id="game_info_winprob_white_bar" class="game_info_winprob_segment game_info_winprob_segment_white" style="width: 50%;">`)
	mainHTMLCode.WriteString(`<span id="game_info_winprob_white" class="game_info_winprob_label">50%</span>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div id="game_info_winprob_black_bar" class="game_info_winprob_segment game_info_winprob_segment_black" style="width: 50%;">`)
	mainHTMLCode.WriteString(`<span id="game_info_winprob_black" class="game_info_winprob_label">50%</span>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="game_info_row game_info_header" role="row">`)
	mainHTMLCode.WriteString(`<div id="game_info_side_white" class="game_info_cell game_info_side ` + whiteTurnClass + `" role="columnheader">White</div>`)
	mainHTMLCode.WriteString(`<div id="game_info_side_black" class="game_info_cell game_info_side ` + blackTurnClass + `" role="columnheader">Black</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="game_info_row" role="row">`)
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + whiteTurnClass + `" role="cell"><span id="game_info_captured_white" class="game_info_item_value game_info_capture_value"></span></div>`)
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + blackTurnClass + `" role="cell"><span id="game_info_captured_black" class="game_info_item_value game_info_capture_value"></span></div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="game_info_row" role="row">`)
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + whiteTurnClass + `" role="cell"><span id="game_info_time_white" class="game_info_item_value">⏱ --:--</span></div>`)
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + blackTurnClass + `" role="cell"><span id="game_info_time_black" class="game_info_item_value">⏱ --:--</span></div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="game_info_row" role="row">`)
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + whiteTurnClass + `" role="cell"><span id="game_info_result_white" class="game_info_item_value">Result: PLAYING</span></div>`)
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + blackTurnClass + `" role="cell"><span id="game_info_result_black" class="game_info_item_value">Result: PLAYING</span></div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="chess_move_history_section" style="margin-top:16px;display:flex;flex-direction:column;flex:1;min-height:0;overflow:hidden;">`)
	mainHTMLCode.WriteString(`<h3 class="chess_move_history_title">Move history</h3>`)
	mainHTMLCode.WriteString(`<div class="chess_move_history_panels" style="display:grid;grid-template-columns:1fr 1fr;gap:10px;min-height:0;flex:1;overflow:hidden;">`)
	mainHTMLCode.WriteString(`<div class="chess_move_history_panel" style="display:flex;flex-direction:column;min-height:0;overflow:hidden;">`)
	mainHTMLCode.WriteString(`<h4 class="chess_move_history_side_title">White</h4>`)
	mainHTMLCode.WriteString(`<ol id="chess_move_history_white" class="chess_move_history_list" style="flex:1;min-height:0;overflow-y:auto;overflow-x:hidden;">`)
	mainHTMLCode.WriteString(`<li class="chess_move_history_placeholder">No moves yet.</li>`)
	mainHTMLCode.WriteString(`</ol>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="chess_move_history_panel" style="display:flex;flex-direction:column;min-height:0;overflow:hidden;">`)
	mainHTMLCode.WriteString(`<h4 class="chess_move_history_side_title">Black</h4>`)
	mainHTMLCode.WriteString(`<ol id="chess_move_history_black" class="chess_move_history_list" style="flex:1;min-height:0;overflow-y:auto;overflow-x:hidden;">`)
	mainHTMLCode.WriteString(`<li class="chess_move_history_placeholder">No moves yet.</li>`)
	mainHTMLCode.WriteString(`</ol>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)

	mainHTMLCode.WriteString(`<div class="game_panel_right_bottom">`)
	mainHTMLCode.WriteString(`<div class="play_chrome">`)
	mainHTMLCode.WriteString(`<label for="chess_command">Chess command</label>`)
	mainHTMLCode.WriteString(`<div class="command_row">`)
	mainHTMLCode.WriteString(`<input id="chess_command" type="text" placeholder="e2e4" />`)
	mainHTMLCode.WriteString(`<button id="chess_command_submit" type="button">Submit</button>`)
	mainHTMLCode.WriteString(`<button id="chess_flag" type="button">Flag</button>`)
	mainHTMLCode.WriteString(`<button id="chess_new_game" type="button">New Game</button>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<p id="chess_command_status" class="command_status" role="status" aria-live="polite"></p>`)
	mainHTMLCode.WriteString(`<div class="command_row review_playback_row">`)
	mainHTMLCode.WriteString(`<button id="review_moves_prev" type="button" disabled>◀ Back</button>`)
	mainHTMLCode.WriteString(`<button id="review_moves_next" type="button" disabled>Forward ▶</button>`)
	mainHTMLCode.WriteString(`<span id="review_moves_ply" class="review_moves_ply" aria-live="polite">Ply 0 / 0</span>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="review_moves_block">`)
	mainHTMLCode.WriteString(`<label for="review_moves_input">Load moves / play JSON</label>`)
	mainHTMLCode.WriteString(`<textarea id="review_moves_input" rows="3" placeholder="e2e4 e7e5 … or paste game-*.json"></textarea>`)
	mainHTMLCode.WriteString(`<div class="command_row review_moves_row">`)
	mainHTMLCode.WriteString(`<input id="review_moves_file" type="file" accept="application/json,.json,text/plain" />`)
	mainHTMLCode.WriteString(`<button id="review_moves_load" type="button">Load moves</button>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div id="simulation_summary_panel" class="simulation_summary_panel simulation_summary_hidden" aria-live="polite">`)
	mainHTMLCode.WriteString(`<table class="simulation_summary_table" aria-label="Simulation summary">`)
	mainHTMLCode.WriteString(`<thead><tr><th>Games</th><th>White wins</th><th>Black wins</th><th>Draws</th><th>Avg moves</th></tr></thead>`)
	mainHTMLCode.WriteString(`<tbody><tr><td id="simulation_summary_games">0</td><td id="simulation_summary_white">0</td><td id="simulation_summary_black">0</td><td id="simulation_summary_draws">0</td><td id="simulation_summary_avg">0.0</td></tr></tbody>`)
	mainHTMLCode.WriteString(`</table>`)
	mainHTMLCode.WriteString(`<details id="simulation_result_details" class="simulation_result_details">`)
	mainHTMLCode.WriteString(`<summary id="simulation_result_summary_text">Per-game results</summary>`)
	mainHTMLCode.WriteString(`<ol id="simulation_result_list" class="simulation_result_list"></ol>`)
	mainHTMLCode.WriteString(`</details>`)
	mainHTMLCode.WriteString(`<div class="simulation_download_actions">`)
	mainHTMLCode.WriteString(`<button id="simulation_download_json_btn" type="button" class="run-simulation-btn simulation_download_btn" disabled>Download JSON</button>`)
	mainHTMLCode.WriteString(`<button id="simulation_download_csv_btn" type="button" class="run-simulation-btn simulation_download_btn" disabled>Download CSV</button>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<input id="active_game_id" type="hidden" value="" />`)
	mainHTMLCode.WriteString(`<div id="promotion_picker" class="promotion_picker_hidden" role="dialog" aria-modal="true" aria-labelledby="promotion_picker_title">`)
	mainHTMLCode.WriteString(`<div class="promotion_picker_panel">`)
	mainHTMLCode.WriteString(`<h4 id="promotion_picker_title">Choose promotion piece</h4>`)
	mainHTMLCode.WriteString(`<div class="promotion_picker_choices">`)
	mainHTMLCode.WriteString(`<button type="button" class="promotion_choice_btn" data-promotion="q">Queen</button>`)
	mainHTMLCode.WriteString(`<button type="button" class="promotion_choice_btn" data-promotion="r">Rook</button>`)
	mainHTMLCode.WriteString(`<button type="button" class="promotion_choice_btn" data-promotion="b">Bishop</button>`)
	mainHTMLCode.WriteString(`<button type="button" class="promotion_choice_btn" data-promotion="n">Knight</button>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<textarea id="game_info_notes" class="game_info_notes_box" placeholder="Reserved for future use" rows="7" readonly></textarea>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<script src="/scripts/chess_command.js"></script>`)

	mainHTMLCode.WriteString(`</div>`)

	// prepare template data for rendering
	data := struct {
		PageTitle   string
		MainContent template.HTML
	}{
		PageTitle:   "Chess Game",
		MainContent: template.HTML(mainHTMLCode.String()),
	}

	// execute the index template with the prepared data
	if err := t.ExecuteTemplate(w, "index", data); err != nil {
		http.Error(w, "Template render error", http.StatusInternalServerError)
		log.Printf("index template execute error: %v", err)
		return
	}
}

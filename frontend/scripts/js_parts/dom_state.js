// CM3070 FP code
// dom_state.js - captures puzzle page DOM refs and seeds shared game state

// DomState - owns puzzle page DOM references, ui constants, and initial state
class DomState {
  constructor(app) {
    this.app = app;
    this.app.ready = true;
    this.bindElements();
    this.bindConstants();
    this.initState();
    if (!this.hasRequiredElements()) {
      this.app.ready = false;
      return;
    }
    this.app.el.gameIdInput = document.getElementById("active_game_id");
    this.app.el.input.focus();
  }

  // bindElements - caches puzzle page element references on app.el
  bindElements() {
    const el = this.app.el;
    el.input = document.getElementById("chess_command");
    el.button = document.getElementById("chess_command_submit");
    el.flagButton = document.getElementById("chess_flag");
    el.status = document.getElementById("chess_command_status");
    el.whiteColumnCells = document.querySelectorAll(".game_info_col_white");
    el.blackColumnCells = document.querySelectorAll(".game_info_col_black");
    el.capturedWhiteValue = document.getElementById("game_info_captured_white");
    el.capturedBlackValue = document.getElementById("game_info_captured_black");
    el.winProbWhiteValue = document.getElementById("game_info_winprob_white");
    el.winProbBlackValue = document.getElementById("game_info_winprob_black");
    el.winProbWhiteBar = document.getElementById("game_info_winprob_white_bar");
    el.winProbBlackBar = document.getElementById("game_info_winprob_black_bar");
    el.resultWhiteValue = document.getElementById("game_info_result_white");
    el.resultBlackValue = document.getElementById("game_info_result_black");
    el.gameInfoNotesBox = document.getElementById("game_info_notes");
    el.moveHistoryWhiteList = document.getElementById("chess_move_history_white");
    el.moveHistoryBlackList = document.getElementById("chess_move_history_black");
    el.newGameButton = document.getElementById("chess_new_game");
    el.reviewMovesInput = document.getElementById("review_moves_input");
    el.reviewMovesFile = document.getElementById("review_moves_file");
    el.reviewMovesLoad = document.getElementById("review_moves_load");
    el.reviewMovesPrev = document.getElementById("review_moves_prev");
    el.reviewMovesNext = document.getElementById("review_moves_next");
    el.reviewMovesPlyLabel = document.getElementById("review_moves_ply");
    el.diagramImportFile = document.getElementById("diagram_import_file");
    el.diagramImportRecognize = document.getElementById("diagram_import_recognize");
    el.diagramImportConfirm = document.getElementById("diagram_import_confirm");
    el.diagramImportFen = document.getElementById("diagram_import_fen");
    el.diagramImportPreview = document.getElementById("diagram_import_preview");
    el.diagramImportNote = document.getElementById("diagram_import_note");
    el.diagramImportConfirmBtn = document.getElementById("diagram_import_confirm_btn");
    el.diagramImportCancelBtn = document.getElementById("diagram_import_cancel_btn");
    el.gameTypeSelect = document.getElementById("game_type");
    el.gameModeSelect = document.getElementById("game_mode");
    el.humanSideSelect = document.getElementById("human_side");
    el.aiGameCountInput = document.getElementById("ai_game_count");
    el.fenInput = document.getElementById("fen_input");
    el.aiStrengthSelect = document.getElementById("ai_strength");
    el.coachLevelSelect = document.getElementById("coach_level");
    el.configApplyButton = document.getElementById("game_config_apply");
    el.clockEnabledInput = document.getElementById("clock_enabled");
    el.clockPresetSelect = document.getElementById("clock_preset");
    el.clockBaseSecInput = document.getElementById("clock_base_sec");
    el.clockIncrementSecInput = document.getElementById("clock_increment_sec");
    el.clockHumanBaseSecInput = document.getElementById("clock_human_base_sec");
    el.clockAiBaseSecInput = document.getElementById("clock_ai_base_sec");
    el.clockHvAIFields = document.getElementById("clock_hvai_fields");
    el.timeWhiteValue = document.getElementById("game_info_time_white");
    el.timeBlackValue = document.getElementById("game_info_time_black");
    el.boardElement = document.querySelector(".chess_board");
    el.boardWrapper = document.querySelector(".chess_board_wrapper");
    el.promotionPicker = document.getElementById("promotion_picker");
    el.simulationSummaryPanel = document.getElementById("simulation_summary_panel");
    el.simulationSummaryGames = document.getElementById("simulation_summary_games");
    el.simulationSummaryWhite = document.getElementById("simulation_summary_white");
    el.simulationSummaryBlack = document.getElementById("simulation_summary_black");
    el.simulationSummaryDraws = document.getElementById("simulation_summary_draws");
    el.simulationSummaryAvg = document.getElementById("simulation_summary_avg");
    el.simulationResultList = document.getElementById("simulation_result_list");
    el.simulationResultDetails = document.getElementById("simulation_result_details");
    el.simulationResultSummaryText = document.getElementById("simulation_result_summary_text");
    el.simulationDownloadJsonBtn = document.getElementById("simulation_download_json_btn");
    el.simulationDownloadCsvBtn = document.getElementById("simulation_download_csv_btn");
  }

  // bindConstants - attaches shared ui class names, audio, and shogi drop maps on app
  bindConstants() {
    const app = this.app;
    app.moveSound = new Audio("/sounds/chess_movement.wav");
    app.captureSound = new Audio("/sounds/capture.wav");
    app.CHECK_CLASS = "game_info_col_in_check";
    app.SELECTED_PIECE_CLASS = "piece_img_selected";
    app.LEGAL_DESTINATION_CLASS = "chess_board_square_legal_destination";
    app.LEGAL_PROMOTION_DESTINATION_CLASS = "chess_board_square_legal_promotion";
    app.LEGAL_CAPTURE_DESTINATION_CLASS = "chess_board_square_legal_capture";
    app.SUGGESTED_MOVE_CLASS = "chess_board_square_suggested";
    app.SUGGESTED_FROM_CLASS = "chess_board_square_suggested_from";
    app.SUGGESTED_DROP_CLASS = "chess_board_square_suggested_drop";
    app.HAND_HINT_CLASS = "shogi_hand_piece_hint";
    app.SHOGI_DROP_KIND_FROM_CHAR = {
      p: "pawn",
      l: "lance",
      n: "knight",
      s: "silver",
      g: "gold",
      b: "bishop",
      r: "rook",
    };
  }

  // initState - seeds mutable session fields on app.state
  initState() {
    const state = this.app.state;
    state.boardFiles = 8;
    state.boardMaxRank = 8;
    state.boardGameType = "chess";
    state.gameOver = false;
    state.currentTurn = "white";
    state.humanColor = "white";
    state.clockEnabledLocal = false;
    state.clockWhiteMs = 0;
    state.clockBlackMs = 0;
    state.clockActiveSide = "white";
    state.clockTickTimer = null;
    state.clockLastTickAt = 0;
    state.clockFlagInFlight = false;
    state.selectedSquareSequence = null;
    state.selectedDropKind = null;
    state.dragSourceSequence = null;
    state.legalMovesRequestVersion = 0;
    state.selectedLegalMoves = [];
    state.isSubmitting = false;
    state.pendingPromotionResolve = null;
    state.analysisPollTimer = null;
    state.analysisPollGeneration = 0;
    state.isSimulationPlayback = false;
    state.simulationData = null;
    state.currentSimGameIdx = 0;
    state.currentSimMoveIdx = 0;
    state.simulationRequestInFlight = false;
    state.simRunBtn = null;
    state.simNextMoveBtn = null;
    state.simNextGameBtn = null;
    state.analysisPollFallbackTimer = null;
    state.pendingAnalysisTargetMove = 0;
    state.pendingAnalysisCapturedSnapshot = null;
    state.cachedAnalysis = null;
    state.cachedCapturedSummary = null;
    state.lastExplanationText = "";
    state.lastSuggestionsText = "";
    state.lastThreatSummary = "";
    state.currentGameId = "";
    state.gameSocket = null;
    state.gameSocketGameId = "";
    state.gameSocketReconnectAttempts = 0;
    state.gameSocketReconnectTimer = null;
    state.gameSocketAllowReconnect = true;
  }

  // hasRequiredElements - checks the critical controls needed to start the page
  hasRequiredElements() {
    const el = this.app.el;
    return Boolean(
      el.input &&
        el.button &&
        el.status &&
        el.moveHistoryWhiteList &&
        el.moveHistoryBlackList &&
        el.boardElement
    );
  }

  // playMoveSound - plays move or capture audio for board updates
  playMoveSound(isCapture) {
    try {
      if (isCapture) {
        this.app.captureSound.currentTime = 0;
        this.app.captureSound.play().catch(() => {});
      } else {
        this.app.moveSound.currentTime = 0;
        this.app.moveSound.play().catch(() => {});
      }
    } catch (_) {}
  }
}

window.DomState = DomState;
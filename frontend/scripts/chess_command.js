// CM3070 FP code
// chess_command.js
// records the movement command from user
// this is operating on frontend level

(() => {
  const input = document.getElementById("chess_command");
  const button = document.getElementById("chess_command_submit");
  const flagButton = document.getElementById("chess_flag");
  const status = document.getElementById("chess_command_status");
  const whiteColumnCells = document.querySelectorAll(".game_info_col_white");
  const blackColumnCells = document.querySelectorAll(".game_info_col_black");
  const capturedWhiteValue = document.getElementById("game_info_captured_white");
  const capturedBlackValue = document.getElementById("game_info_captured_black");
  const winProbWhiteValue = document.getElementById("game_info_winprob_white");
  const winProbBlackValue = document.getElementById("game_info_winprob_black");
  const winProbWhiteBar = document.getElementById("game_info_winprob_white_bar");
  const winProbBlackBar = document.getElementById("game_info_winprob_black_bar");
  const resultWhiteValue = document.getElementById("game_info_result_white");
  const resultBlackValue = document.getElementById("game_info_result_black");
  const gameInfoNotesBox = document.getElementById("game_info_notes");
  const moveHistoryWhiteList = document.getElementById("chess_move_history_white");
  const moveHistoryBlackList = document.getElementById("chess_move_history_black");
  const newGameButton = document.getElementById("chess_new_game");
  const reviewMovesInput = document.getElementById("review_moves_input");
  const reviewMovesFile = document.getElementById("review_moves_file");
  const reviewMovesLoad = document.getElementById("review_moves_load");
  const reviewMovesPrev = document.getElementById("review_moves_prev");
  const reviewMovesNext = document.getElementById("review_moves_next");
  const reviewMovesPlyLabel = document.getElementById("review_moves_ply");
  const gameTypeSelect = document.getElementById("game_type");
  const gameModeSelect = document.getElementById("game_mode");
  const humanSideSelect = document.getElementById("human_side");
  const aiGameCountInput = document.getElementById("ai_game_count");
  const fenInput = document.getElementById("fen_input");
  const aiStrengthSelect = document.getElementById("ai_strength");
  const coachLevelSelect = document.getElementById("coach_level");
  const configApplyButton = document.getElementById("game_config_apply");
  const clockEnabledInput = document.getElementById("clock_enabled");
  const clockPresetSelect = document.getElementById("clock_preset");
  const clockBaseSecInput = document.getElementById("clock_base_sec");
  const clockIncrementSecInput = document.getElementById("clock_increment_sec");
  const clockHumanBaseSecInput = document.getElementById("clock_human_base_sec");
  const clockAiBaseSecInput = document.getElementById("clock_ai_base_sec");
  const clockHvAIFields = document.getElementById("clock_hvai_fields");
  const timeWhiteValue = document.getElementById("game_info_time_white");
  const timeBlackValue = document.getElementById("game_info_time_black");
  const boardElement = document.querySelector(".chess_board");
  const boardWrapper = document.querySelector(".chess_board_wrapper");
  let boardFiles = 8;
  let boardMaxRank = 8;
  let boardGameType = "chess";
  const promotionPicker = document.getElementById("promotion_picker");
  const simulationSummaryPanel = document.getElementById("simulation_summary_panel");
  const simulationSummaryGames = document.getElementById("simulation_summary_games");
  const simulationSummaryWhite = document.getElementById("simulation_summary_white");
  const simulationSummaryBlack = document.getElementById("simulation_summary_black");
  const simulationSummaryDraws = document.getElementById("simulation_summary_draws");
  const simulationSummaryAvg = document.getElementById("simulation_summary_avg");
  const simulationResultList = document.getElementById("simulation_result_list");
  const simulationResultDetails = document.getElementById("simulation_result_details");
  const simulationResultSummaryText = document.getElementById("simulation_result_summary_text");
  const simulationDownloadJsonBtn = document.getElementById("simulation_download_json_btn");
  const simulationDownloadCsvBtn = document.getElementById("simulation_download_csv_btn");
  const moveSound = new Audio("/sounds/chess_movement.wav");
  const captureSound = new Audio("/sounds/capture.wav");
  const CHECK_CLASS = "game_info_col_in_check";
  const SELECTED_PIECE_CLASS = "piece_img_selected";
  const LEGAL_DESTINATION_CLASS = "chess_board_square_legal_destination";
  const LEGAL_PROMOTION_DESTINATION_CLASS = "chess_board_square_legal_promotion";
  const LEGAL_CAPTURE_DESTINATION_CLASS = "chess_board_square_legal_capture";
  const SUGGESTED_MOVE_CLASS = "chess_board_square_suggested";
  const SUGGESTED_FROM_CLASS = "chess_board_square_suggested_from";
  const SUGGESTED_DROP_CLASS = "chess_board_square_suggested_drop";
  const HAND_HINT_CLASS = "shogi_hand_piece_hint";
  const SHOGI_DROP_KIND_FROM_CHAR = {
    p: "pawn",
    l: "lance",
    n: "knight",
    s: "silver",
    g: "gold",
    b: "bishop",
    r: "rook",
  };
  let gameOver = false;
  let currentTurn = "white";
  let humanColor = "white";           // human's chosen color in Human vs AI mode
  let clockEnabledLocal = false;
  let clockWhiteMs = 0;
  let clockBlackMs = 0;
  let clockActiveSide = "white";
  let clockTickTimer = null;
  let clockLastTickAt = 0;
  let clockFlagInFlight = false;
  let selectedSquareSequence = null;
  let selectedDropKind = null; // shogi hand: { side, kind } or null
  let dragSourceSequence = null;
  let legalMovesRequestVersion = 0;
  let selectedLegalMoves = [];
  let isSubmitting = false;
  let pendingPromotionResolve = null;
  let analysisPollTimer = null;
  let isSimulationPlayback = false; // Flag to suppress hints during AI sim
  // Manual simulation playback state
  let simulationData = null;
  let currentSimGameIdx = 0;
  let currentSimMoveIdx = 0;
  let simulationRequestInFlight = false;
  let simRunBtn = null;
  let simNextMoveBtn = null;
  let simNextGameBtn = null;
  let analysisPollFallbackTimer = null;
  let pendingAnalysisTargetMove = 0;
  let pendingAnalysisCapturedSnapshot = null;
  let cachedAnalysis = null;
  let cachedCapturedSummary = null;
  let lastExplanationText = "";
  let lastSuggestionsText = "";
  let lastThreatSummary = "";
  let currentGameId = "";
  let gameSocket = null;

  const playMoveSound = (isCapture) => {
    try {
      if (isCapture) {
        captureSound.currentTime = 0;
        captureSound.play().catch(() => {});
      } else {
        moveSound.currentTime = 0;
        moveSound.play().catch(() => {});
      }
    } catch (_) {}
  };

  let gameSocketGameId = "";
  let gameSocketReconnectAttempts = 0;
  let gameSocketReconnectTimer = null;
  let gameSocketAllowReconnect = true;

  if (!input || !button || !status || !moveHistoryWhiteList || !moveHistoryBlackList || !boardElement) return;

  const gameIdInput = document.getElementById("active_game_id");

  input.focus();

  // set current status
  const setStatus = (message, type) => {
    status.textContent = message;
    status.className = `command_status ${type}`;
  };

  const setNotesText = (text) => {
    if (!gameInfoNotesBox) return;
    gameInfoNotesBox.value = String(text || "");
    gameInfoNotesBox.scrollTop = gameInfoNotesBox.scrollHeight;
  };

  // Keep suggested moves + threat + latest coach note as one composition.
  // Analysis/explain updates must not wipe the suggestions block.
  const composeNotesText = () => {
    const parts = [];
    if (lastSuggestionsText) parts.push(lastSuggestionsText);
    if (lastThreatSummary) parts.push(lastThreatSummary);
    if (lastExplanationText) parts.push(lastExplanationText);
    return parts.join("\n\n");
  };

  const refreshNotesBox = () => {
    if (!gameInfoNotesBox) return;
    setNotesText(composeNotesText());
    if (lastSuggestionsText) gameInfoNotesBox.dataset.fsSuggestions = "1";
    else delete gameInfoNotesBox.dataset.fsSuggestions;
  };

  const clearCoachNotesState = () => {
    lastExplanationText = "";
    lastSuggestionsText = "";
    lastThreatSummary = "";
  };

  const showGameEndedNotes = (message) => {
    clearCoachNotesState();
    clearSuggestedHighlights();
    selectedSuggestedMoves = [];
    lastSuggestionsText = String(message || "Game has ended.").trim();
    refreshNotesBox();
  };

  const appendNotesLine = (line) => {
    if (!gameInfoNotesBox) return;
    const next = String(line || "").trim();
    if (!next) return;
    const current = gameInfoNotesBox.value.trim();
    gameInfoNotesBox.value = current ? `${current}\n${next}` : next;
    gameInfoNotesBox.scrollTop = gameInfoNotesBox.scrollHeight;
  };

  const isAIVsAIModeSelected = () => String(gameModeSelect?.value || "") === "ai_vs_ai";

  const setSimulationDownloadEnabled = (enabled) => {
    if (simulationDownloadJsonBtn) simulationDownloadJsonBtn.disabled = !enabled;
    if (simulationDownloadCsvBtn) simulationDownloadCsvBtn.disabled = !enabled;
  };

  const buildSimulationExportPayload = () => {
    if (!simulationData || !Array.isArray(simulationData.results)) return null;
    return {
      exported_at: new Date().toISOString(),
      profile: String(aiStrengthSelect?.value || "intermediate"),
      mode: String(gameModeSelect?.value || "ai_vs_ai"),
      game_type: String(gameTypeSelect?.value || "chess"),
      summary: {
        games: Number(simulationData.games || 0),
        white_wins: Number(simulationData.white_wins || 0),
        black_wins: Number(simulationData.black_wins || 0),
        draws: Number(simulationData.draws || 0),
        avg_moves: Number(simulationData.avg_moves || 0),
      },
      results: simulationData.results,
    };
  };

  const buildSimulationJSON = () => {
    const payload = buildSimulationExportPayload();
    return payload ? JSON.stringify(payload, null, 2) : "";
  };

  const csvEscape = (value) => {
    const text = String(value ?? "");
    if (/[",\n]/.test(text)) return `"${text.replace(/"/g, '""')}"`;
    return text;
  };

  const buildSimulationCSV = () => {
    const payload = buildSimulationExportPayload();
    if (!payload) return "";
    const lines = ["game,result,winner,moves"];
    for (let i = 0; i < payload.results.length; i++) {
      const row = payload.results[i] || {};
      lines.push([
        i + 1,
        csvEscape(row.result || ""),
        csvEscape(row.winner || ""),
        Number(row.moves || 0),
      ].join(","));
    }
    const summary = payload.summary;
    lines.push(
      `# Summary,${summary.games} games,White ${summary.white_wins},Black ${summary.black_wins},Draws ${summary.draws},Avg ${Number(summary.avg_moves || 0).toFixed(1)}`
    );
    return lines.join("\n");
  };

  const simulationDownloadFilename = (ext) => {
    const profile = String(aiStrengthSelect?.value || "intermediate");
    const stamp = new Date().toISOString().replace(/[:.]/g, "-");
    return `simulation-${profile}-${stamp}.${ext}`;
  };

  const downloadTextFile = (filename, mimeType, content) => {
    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    link.click();
    URL.revokeObjectURL(url);
  };

  const clearSimulationSummary = () => {
    if (simulationSummaryGames) simulationSummaryGames.textContent = "0";
    if (simulationSummaryWhite) simulationSummaryWhite.textContent = "0";
    if (simulationSummaryBlack) simulationSummaryBlack.textContent = "0";
    if (simulationSummaryDraws) simulationSummaryDraws.textContent = "0";
    if (simulationSummaryAvg) simulationSummaryAvg.textContent = "0.0";
    if (simulationResultList) simulationResultList.innerHTML = "";
    if (simulationResultSummaryText) simulationResultSummaryText.textContent = "Per-game results";
    if (simulationResultDetails) simulationResultDetails.open = false;
    if (simulationSummaryPanel) simulationSummaryPanel.classList.add("simulation_summary_hidden");
    setSimulationDownloadEnabled(false);
  };

  const renderSimulationSummary = (summary) => {
    const games = Number(summary?.games || 0);
    const whiteWins = Number(summary?.white_wins || 0);
    const blackWins = Number(summary?.black_wins || 0);
    const draws = Number(summary?.draws || 0);
    const avgMoves = Number(summary?.avg_moves || 0);

    if (simulationSummaryGames) simulationSummaryGames.textContent = String(games);
    if (simulationSummaryWhite) simulationSummaryWhite.textContent = String(whiteWins);
    if (simulationSummaryBlack) simulationSummaryBlack.textContent = String(blackWins);
    if (simulationSummaryDraws) simulationSummaryDraws.textContent = String(draws);
    if (simulationSummaryAvg) simulationSummaryAvg.textContent = Number.isFinite(avgMoves) ? avgMoves.toFixed(1) : "0.0";

    if (simulationResultList) {
      simulationResultList.innerHTML = "";
      const results = Array.isArray(summary?.results) ? summary.results : [];
      if (simulationResultSummaryText) {
        simulationResultSummaryText.textContent = `Per-game results (${results.length})`;
      }
      if (simulationResultDetails) {
        simulationResultDetails.open = false;
      }
      for (let i = 0; i < results.length; i++) {
        const item = document.createElement("li");
        const one = results[i] || {};
        const result = String(one.result || "unknown");
        const winner = String(one.winner || "-");
        const moves = Number(one.moves || 0);
        item.textContent = `Game ${i + 1}: ${result} | winner: ${winner} | moves: ${moves}`;
        simulationResultList.appendChild(item);
      }
    }

    if (simulationSummaryPanel) simulationSummaryPanel.classList.remove("simulation_summary_hidden");
    setSimulationDownloadEnabled(Array.isArray(summary?.results) && summary.results.length > 0);
  };

  const readSimulationCount = () => {
    const raw = String(aiGameCountInput?.value || "").trim();
    const n = Number(raw);
    if (!Number.isInteger(n) || n < 1 || n > 1000) {
      return { ok: false, message: "Please enter an integer game count between 1 and 1000." };
    }
    return { ok: true, count: n };
  };

  const readErrorMessage = async (response, fallback) => {
    try {
      const payload = await response.json();
      const message = String(payload?.message || "").trim();
      return message || fallback;
    } catch (_) {
      const text = (await response.text()).trim();
      return text || fallback;
    }
  };

  const setCatchStatus = (error, networkMsg = "Network error. Please try again.") => {
    console.error(error);
    const msg = String(error?.message || "");
    const isNetwork =
      error instanceof TypeError &&
      (/failed to fetch|networkerror|load failed|network request failed/i.test(msg) ||
        msg === "Failed to fetch");
    if (isNetwork) {
      setStatus(networkMsg, "error");
      return;
    }
    setStatus(msg ? `Error: ${msg}` : "Something went wrong. Check the console.", "error");
  };

  const syncGameIdFromResult = (result) => {
    const nextId = String(result?.game?.id || "").trim();
    if (!nextId) return;
    const changed = nextId !== currentGameId;
    currentGameId = nextId;
    if (gameIdInput) gameIdInput.value = nextId;
    if (changed) {
      connectGameSocket(nextId);
    }
  };

  const stopAnalysisPolling = () => {
    if (analysisPollTimer != null) {
      window.clearInterval(analysisPollTimer);
      analysisPollTimer = null;
    }
    if (analysisPollFallbackTimer != null) {
      window.clearTimeout(analysisPollFallbackTimer);
      analysisPollFallbackTimer = null;
    }
    pendingAnalysisTargetMove = 0;
    pendingAnalysisCapturedSnapshot = null;
  };

  const isSocketConnected = () =>
    Boolean(gameSocket && gameSocket.readyState === WebSocket.OPEN);

  const clearSocketReconnectTimer = () => {
    if (gameSocketReconnectTimer != null) {
      window.clearTimeout(gameSocketReconnectTimer);
      gameSocketReconnectTimer = null;
    }
  };

  const socketURLForGame = (gameId) => {
    const protocol = window.location.protocol === "https:" ? "wss" : "ws";
    return `${protocol}://${window.location.host}/ws/game?gameId=${encodeURIComponent(gameId)}`;
  };

  const closeGameSocket = (allowReconnect) => {
    gameSocketAllowReconnect = Boolean(allowReconnect);
    clearSocketReconnectTimer();
    if (gameSocket) {
      try {
        gameSocket.close();
      } catch (_) {
        // ignore close errors
      }
    }
    gameSocket = null;
  };

  const refreshGameSnapshotFromAPI = async (gameId) => {
    const targetGameId = String(gameId || currentGameId || "").trim();
    if (!targetGameId) return;
    try {
      const response = await fetch(`/api/games/${encodeURIComponent(targetGameId)}`, {
        method: "GET",
      });
      if (!response.ok) return;
      const result = await response.json();
      syncGameIdFromResult(result);
      renderGameConfig(result.game);
      renderBoardFromState(result.state, result.game?.type);
      renderMoveHistory(result.history, result.historyDetailed);
      renderCurrentTurn(result.currentTurn);
      renderCheckState(result.checkedSide || result?.game?.outcome?.checkedSide);
      renderGameOutcome(result.game);
      renderClocks(result.game);
      renderGameInfo(result.captured, gameOver ? null : result.analysis);
      clearSelectedSquare();
      if (gameOver) {
        stopAnalysisPolling();
        return;
      }
      void refreshSuggestedMoves();
      const historyArray = Array.isArray(result.history) ? result.history : [];
      const detailedArray = Array.isArray(result.historyDetailed) ? result.historyDetailed : [];
      if (result.analysis) {
        stopAnalysisPolling();
      } else {
        const targetMoveNumber = Math.max(historyArray.length, detailedArray.length);
        if (targetMoveNumber > 0) startAnalysisPolling(targetMoveNumber, result.captured);
      }
    } catch (_) {
      // ignore transient refresh errors; REST fallback remains available
    }
  };

  const handleSocketMessage = (payload) => {
    const event = String(payload?.event || "");
    const gameId = String(payload?.game_id || "");
    // Allow simulation events even if they don't match currentGameId
    const isSimulationEvent = event.startsWith("simulation_");
    if (!event || (!isSimulationEvent && gameId !== currentGameId)) return;
    const data = payload?.data || {};

    if (event === "move_applied") {
      if (data?.clock || data?.remaining) {
        applyServerClock(data.clock, data.remaining);
      }
      // Update board immediately — the CSS transition on .piece_img (400ms) gives the slide animation
      void refreshGameSnapshotFromAPI(gameId);
      // Play sound at the same time the piece starts moving (capture vs quiet from history)
      playMoveSound(Boolean(data?.isCapture));
      // Coach text arrives later (Ollama); show placeholder now so notes don't jump from empty → wall of text.
      // Do NOT refresh Suggested moves here — top-moves races the new position and clears/flickers the list.
      lastExplanationText = "[coach] Thinking…";
      refreshNotesBox();
      return;
    }
    if (event === "turn_changed") {
      renderCurrentTurn(data?.current_turn);
      renderCheckState(data?.checked_side);
      if (data?.clock || data?.remaining) {
        applyServerClock(data.clock, data.remaining);
      } else if (data?.current_turn && clockEnabledLocal) {
        clockActiveSide = String(data.current_turn).toLowerCase();
        clockLastTickAt = Date.now();
      }
      return;
    }
    if (event === "game_outcome") {
      renderGameOutcome({
        result: data?.result,
        outcome: data?.outcome || {},
      });
      if (gameOver) stopClockTick();
      return;
    }
    if (event === "analysis_status_update") {
      const statusText = String(data?.status || "").toLowerCase();
      if (statusText === "pending") {
        // Do not wipe Suggested moves / coach placeholder with a bare "Analyzing...".
        return;
      }
      if (statusText === "ready" && data?.analysis) {
        renderGameInfo(pendingAnalysisCapturedSnapshot || cachedCapturedSummary, data.analysis);
        stopAnalysisPolling();
        if (gameOver) return;
        // Refresh hints once analysis/position is settled (avoids mid-move flicker).
        void refreshSuggestedMoves();
        return;
      }
      if (statusText === "error") {
        if (gameOver) return;
        const safeMessage = String(data?.last_error || "").trim();
        if (safeMessage) {
          lastThreatSummary = safeMessage;
          refreshNotesBox();
        }
        void refreshSuggestedMoves();
      }
    }
    if (event === "explanation_ready" || event === "explanationReady") {
      if (!gameInfoNotesBox || gameOver) return;
      const expl = String(data?.explanation || data?.analysis_explanation || "").trim();
      if (!expl) return;
      const skill = String(data?.skill_level || "").trim().toLowerCase();
      const bits = [];
      if (skill) bits.push(`coach:${skill}`);
      if (data?.source === "heuristic_fallback") bits.push("heuristic");
      const prefix = bits.length ? `[${bits.join(" · ")}] ` : "";
      lastExplanationText = prefix + expl;
      refreshNotesBox();
    }

    // Optional live socket simulation stream (kept for observers).
    // Manual playback mode uses API response history instead.
    if (event === "simulation_move" && !simulationData && !isSimulationPlayback) {
      const move = data?.move || "";
      const gameNum = data?.game_num || "";

      // Use a timeout to slow down the visual playback so moves are observable
      setTimeout(() => {
        if (move && boardElement) {
          applyUciMoveToBoard(move);
          playMoveSound(false);
          // Explicitly clear any suggested move highlights during simulation
          highlightSuggestedMoves([]);
        }

        // Update move history panels
        // We don't have piece kind easily, so we use a generic approach or just the command.
        // For simplicity in simulation, we append to the correct side based on move parity.
        const moveNum = data?.move_num || 0;
        const side = (moveNum % 2 === 1) ? "white" : "black";
        const listEl = side === "white" ? moveHistoryWhiteList : moveHistoryBlackList;

        if (listEl) {
          // Clear placeholder if exists
          const placeholder = listEl.querySelector(".chess_move_history_placeholder");
          if (placeholder) placeholder.remove();

          const item = document.createElement("li");
          item.textContent = move;
          listEl.appendChild(item);
          listEl.scrollTop = listEl.scrollHeight;
        }

        // Log to notes box as well
        const line = gameNum ? `Game ${gameNum}: ${move}` : move;
        appendNotesLine(line);
      }, 300); // 300ms delay between moves for visibility
      return;
    }
    if (event === "simulation_game_end" && !isSimulationPlayback) {
      const status = data?.status || "finished";
      const gameNum = data?.game_num || 0;

      appendNotesLine(`[Game ${gameNum} ${status}]`);

      if (status === "started" && gameNum > 1) {
        resetBoardToInitialState();
        resetSimulationHistoryPanels();
      }
      return;
    }
    if (event === "simulation_completed" && !isSimulationPlayback && data) {
      renderSimulationSummary(data);
      return;
    }
  };

  const connectGameSocket = (gameId) => {
    const targetGameId = String(gameId || "").trim();
    if (!targetGameId || typeof WebSocket === "undefined") return;
    if (
      gameSocket &&
      gameSocketGameId === targetGameId &&
      (gameSocket.readyState === WebSocket.OPEN || gameSocket.readyState === WebSocket.CONNECTING)
    ) {
      return;
    }

    closeGameSocket(false);
    gameSocketAllowReconnect = true;
    gameSocketGameId = targetGameId;

    try {
      const ws = new WebSocket(socketURLForGame(targetGameId));
      gameSocket = ws;

      ws.addEventListener("open", () => {
        gameSocketReconnectAttempts = 0;
        clearSocketReconnectTimer();
        // Resync clocks/board after connect or reconnect (do not trust stale local remaining).
        void refreshGameSnapshotFromAPI(targetGameId);
      });

      ws.addEventListener("message", (evt) => {
        try {
          const payload = JSON.parse(String(evt.data || "{}"));
          handleSocketMessage(payload);
        } catch (_) {
          // ignore malformed socket payloads
        }
      });

      ws.addEventListener("close", () => {
        const sameSocket = ws === gameSocket;
        if (sameSocket) gameSocket = null;
        if (!gameSocketAllowReconnect || gameSocketGameId !== targetGameId) return;

        // REST polling remains fallback when socket is unavailable.
        if (pendingAnalysisTargetMove > 0) {
          startAnalysisPolling(pendingAnalysisTargetMove, pendingAnalysisCapturedSnapshot || cachedCapturedSummary);
        }

        clearSocketReconnectTimer();
        gameSocketReconnectAttempts += 1;
        const delay = Math.min(4000, 500 * Math.pow(2, gameSocketReconnectAttempts - 1));
        gameSocketReconnectTimer = window.setTimeout(() => {
          connectGameSocket(targetGameId);
        }, delay);
      });

      ws.addEventListener("error", () => {
        try {
          ws.close();
        } catch (_) {
          // ignore
        }
      });
    } catch (_) {
      // If socket init fails, existing REST flow remains source of truth.
    }
  };

  const renderCurrentTurn = (turnText) => {
    if (!turnText) return;
    const isWhiteTurn = turnText.toLowerCase() === "white";
    currentTurn = isWhiteTurn ? "white" : "black";
    whiteColumnCells.forEach((cell) => {
      cell.classList.toggle("game_info_col_active", isWhiteTurn);
    });
    blackColumnCells.forEach((cell) => {
      cell.classList.toggle("game_info_col_active", !isWhiteTurn);
    });
    void refreshSuggestedMoves();
  };

  const renderCheckState = (checkedSide) => {
    const side = String(checkedSide || "").toLowerCase();
    const whiteChecked = side === "white";
    const blackChecked = side === "black";
    whiteColumnCells.forEach((cell) => {
      cell.classList.toggle(CHECK_CLASS, whiteChecked);
    });
    blackColumnCells.forEach((cell) => {
      cell.classList.toggle(CHECK_CLASS, blackChecked);
    });
  };

  const capitalize = (text) => {
    const value = String(text || "").toLowerCase();
    if (!value) return "";
    return value.charAt(0).toUpperCase() + value.slice(1);
  };

  const renderGameOutcome = (game) => {
    const outcome = game?.outcome || game || {};
    const statusValue = String(outcome?.status || "").toLowerCase();
    const gameResult = String(game?.result || "in_progress").toLowerCase();
    const drawReasonText = () => {
      if (statusValue === "stalemate") return "draw by stalemate";
      if (statusValue === "draw_insufficient_material") return "draw by insufficient material";
      if (statusValue === "draw_threefold_repetition") return "draw by threefold repetition";
      if (statusValue === "draw_fifty_move_rule") return "draw by 50-move rule";
      return "draw";
    };
    const resetResultClasses = (el) => {
      if (!el) return;
      el.classList.remove("game_info_result_win", "game_info_result_loss", "game_info_result_draw");
    };

    if (resultWhiteValue && resultBlackValue) {
      resetResultClasses(resultWhiteValue);
      resetResultClasses(resultBlackValue);
      if (gameResult === "white_win") {
        resultWhiteValue.textContent = "Result: WIN";
        resultBlackValue.textContent = "Result: LOSS";
        resultWhiteValue.classList.add("game_info_result_win");
        resultBlackValue.classList.add("game_info_result_loss");
      } else if (gameResult === "black_win") {
        resultWhiteValue.textContent = "Result: LOSS";
        resultBlackValue.textContent = "Result: WIN";
        resultWhiteValue.classList.add("game_info_result_loss");
        resultBlackValue.classList.add("game_info_result_win");
      } else if (gameResult === "draw") {
        const drawLabel = `Result: ${drawReasonText()}`;
        resultWhiteValue.textContent = drawLabel;
        resultBlackValue.textContent = drawLabel;
        resultWhiteValue.classList.add("game_info_result_draw");
        resultBlackValue.classList.add("game_info_result_draw");
      } else {
        resultWhiteValue.textContent = "Result: PLAYING";
        resultBlackValue.textContent = "Result: PLAYING";
      }
    }
    if (statusValue === "checkmate") {
      const winner = capitalize(outcome?.winner);
      const loser = capitalize(outcome?.loser);
      setStatus(`Checkmate! ${winner} wins. ${loser} loses.`, "error");
      input.disabled = true;
      button.disabled = true;
      if (flagButton) flagButton.disabled = true;
      gameOver = true;
      stopClockTick();
      showGameEndedNotes(`Game has ended. Checkmate — ${winner} wins.`);
      updateSetupControlState();
      return;
    }

    if (statusValue === "stalemate") {
      setStatus("Draw by stalemate.", "success");
      input.disabled = true;
      button.disabled = true;
      if (flagButton) flagButton.disabled = true;
      gameOver = true;
      stopClockTick();
      showGameEndedNotes("Game has ended. Draw by stalemate.");
      updateSetupControlState();
      return;
    }
    if (statusValue.startsWith("draw_")) {
      setStatus(outcome?.message || "Game drawn.", "success");
      input.disabled = true;
      button.disabled = true;
      if (flagButton) flagButton.disabled = true;
      gameOver = true;
      stopClockTick();
      showGameEndedNotes(outcome?.message || "Game has ended. Draw.");
      updateSetupControlState();
      return;
    }

    if (statusValue === "resigned") {
      setStatus(outcome?.message || "Game ended by flag.", "error");
      input.disabled = true;
      button.disabled = true;
      if (flagButton) flagButton.disabled = true;
      gameOver = true;
      stopClockTick();
      showGameEndedNotes(outcome?.message || "Game has ended (flag / resign).");
      updateSetupControlState();
      return;
    }

    input.disabled = false;
    button.disabled = false;
    if (flagButton) flagButton.disabled = false;
    gameOver = false;

    if (statusValue === "check") {
      const checked = capitalize(outcome?.checkedSide);
      const legalMoves = Number(outcome?.legalMoves || 0);
      setStatus(`${checked} is in check. Legal moves available: ${legalMoves}.`, "error");
      return;
    }

    setStatus("", "success");
  };

  const CLOCK_PRESETS = {
    "5|0": { baseSec: 300, incrementSec: 0 },
    "10|0": { baseSec: 600, incrementSec: 0 },
    "15|10": { baseSec: 900, incrementSec: 10 },
    "5|30": { baseSec: 300, incrementSec: 30 },
  };

  const formatClockMs = (ms) => {
    const totalSec = Math.max(0, Math.floor(Number(ms) / 1000) || 0);
    const minutes = Math.floor(totalSec / 60);
    const seconds = totalSec % 60;
    return `${minutes}:${String(seconds).padStart(2, "0")}`;
  };

  const stopClockTick = () => {
    if (clockTickTimer != null) {
      window.clearInterval(clockTickTimer);
      clockTickTimer = null;
    }
    clockLastTickAt = 0;
  };

  const paintClockLabels = () => {
    if (!timeWhiteValue || !timeBlackValue) return;
    if (!clockEnabledLocal) {
      timeWhiteValue.textContent = "⏱ --:--";
      timeBlackValue.textContent = "⏱ --:--";
      return;
    }
    timeWhiteValue.textContent = `⏱ ${formatClockMs(clockWhiteMs)}`;
    timeBlackValue.textContent = `⏱ ${formatClockMs(clockBlackMs)}`;
  };

  const startClockTick = () => {
    if (!clockEnabledLocal || gameOver || simulationRequestInFlight || isSimulationPlayback) {
      stopClockTick();
      return;
    }
    if (clockTickTimer != null) return;
    clockLastTickAt = Date.now();
    clockTickTimer = window.setInterval(() => {
      if (!clockEnabledLocal || gameOver) {
        stopClockTick();
        return;
      }
      const now = Date.now();
      const elapsed = now - clockLastTickAt;
      clockLastTickAt = now;
      if (elapsed > 0) {
        if (clockActiveSide === "black") {
          clockBlackMs = Math.max(0, clockBlackMs - elapsed);
        } else {
          clockWhiteMs = Math.max(0, clockWhiteMs - elapsed);
        }
      }
      paintClockLabels();
      const remaining = clockActiveSide === "black" ? clockBlackMs : clockWhiteMs;
      if (remaining <= 0) {
        stopClockTick();
        void flagOnLocalTimeout();
      }
    }, 250);
  };

  const applyServerClock = (clk, remaining) => {
    if (!clk || !clk.enabled) {
      clockEnabledLocal = false;
      clockFlagInFlight = false;
      stopClockTick();
      paintClockLabels();
      return;
    }
    clockEnabledLocal = true;
    if (remaining && remaining.white != null && remaining.black != null) {
      clockWhiteMs = Math.max(0, Number(remaining.white) || 0);
      clockBlackMs = Math.max(0, Number(remaining.black) || 0);
    } else {
      clockWhiteMs = Math.max(0, Number(clk.whiteRemainingMs) || 0);
      clockBlackMs = Math.max(0, Number(clk.blackRemainingMs) || 0);
    }
    const active = String(clk.active || currentTurn || "white").toLowerCase();
    clockActiveSide = active === "black" ? "black" : "white";
    clockFlagInFlight = false;
    clockLastTickAt = Date.now();
    paintClockLabels();
    if (!gameOver) startClockTick();
    else stopClockTick();
  };

  const renderClocks = (game) => {
    applyServerClock(game?.clock, null);
  };

  const flagOnLocalTimeout = async () => {
    if (clockFlagInFlight || gameOver || !currentGameId) return;
    if (simulationRequestInFlight || isSimulationPlayback) return;
    clockFlagInFlight = true;
    try {
      const response = await fetch(`/api/games/${encodeURIComponent(currentGameId)}/flag`, {
        method: "POST",
      });
      if (!response.ok) {
        clockFlagInFlight = false;
        void refreshGameSnapshotFromAPI(currentGameId);
        return;
      }
      const result = await response.json();
      syncGameIdFromResult(result);
      renderMoveHistory(result.history, result.historyDetailed);
      renderCurrentTurn(result.currentTurn);
      renderCheckState(result.checkedSide || result?.game?.outcome?.checkedSide);
      renderGameOutcome(result.game);
      renderClocks(result.game);
      renderGameConfig(result.game);
      if (result.game?.config?.humanColor) {
        humanColor = String(result.game.config.humanColor).toLowerCase();
      }
      cachedAnalysis = null;
      renderGameInfo(result.captured, null);
      stopAnalysisPolling();
      resolvePromotionChoice("");
      clearSelectedSquare();
      if (gameOver) {
        showGameEndedNotes(result?.game?.outcome?.message || "Game has ended (flag / resign).");
      }
    } catch (_) {
      clockFlagInFlight = false;
      void refreshGameSnapshotFromAPI(currentGameId);
    }
  };

  const applyClockPresetToInputs = () => {
    const key = String(clockPresetSelect?.value || "");
    const preset = CLOCK_PRESETS[key];
    if (!preset) return;
    if (clockBaseSecInput) clockBaseSecInput.value = String(preset.baseSec);
    if (clockIncrementSecInput) clockIncrementSecInput.value = String(preset.incrementSec);
    if (clockHumanBaseSecInput) clockHumanBaseSecInput.value = String(preset.baseSec);
  };

  const appendClockFields = (params) => {
    const enabled = Boolean(clockEnabledInput?.checked);
    params.set("clockEnabled", enabled ? "true" : "false");
    if (!enabled) return params;
    const mode = String(gameModeSelect?.value || "human_vs_human");
    const incMs = String(Math.max(0, Math.round(Number(clockIncrementSecInput?.value || 0) * 1000)));
    params.set("incrementMs", incMs);
    if (mode === "human_vs_ai") {
      params.set(
        "humanInitialMs",
        String(Math.max(0, Math.round(Number(clockHumanBaseSecInput?.value || 0) * 1000)))
      );
      params.set(
        "aiInitialMs",
        String(Math.max(0, Math.round(Number(clockAiBaseSecInput?.value || 0) * 1000)))
      );
      return params;
    }
    const baseMs = String(Math.max(0, Math.round(Number(clockBaseSecInput?.value || 0) * 1000)));
    params.set("whiteInitialMs", baseMs);
    params.set("blackInitialMs", baseMs);
    return params;
  };

  const syncClockControlsFromGame = (game) => {
    const clk = game?.clock;
    if (!clockEnabledInput || !clk) return;
    clockEnabledInput.checked = Boolean(clk.enabled);
    if (!clk.enabled) return;
    const whiteSec = Math.max(0, Math.round(Number(clk.whiteInitialMs || 0) / 1000));
    const blackSec = Math.max(0, Math.round(Number(clk.blackInitialMs || 0) / 1000));
    const incSec = Math.max(0, Math.round(Number(clk.incrementMs || 0) / 1000));
    if (clockIncrementSecInput) clockIncrementSecInput.value = String(incSec);
    if (clockBaseSecInput) clockBaseSecInput.value = String(whiteSec);
    const side = String(game?.config?.humanColor || humanColor || "white").toLowerCase();
    if (side === "black") {
      if (clockHumanBaseSecInput) clockHumanBaseSecInput.value = String(blackSec);
      if (clockAiBaseSecInput) clockAiBaseSecInput.value = String(whiteSec);
    } else {
      if (clockHumanBaseSecInput) clockHumanBaseSecInput.value = String(whiteSec);
      if (clockAiBaseSecInput) clockAiBaseSecInput.value = String(blackSec);
    }
    const matched = Object.entries(CLOCK_PRESETS).find(
      ([, preset]) => preset.baseSec === whiteSec && preset.incrementSec === incSec
    );
    if (clockPresetSelect) clockPresetSelect.value = matched ? matched[0] : "custom";
  };

  const updateSetupControlState = () => {
    const mode = String(gameModeSelect?.value || "human_vs_human");
    const isAIVsAI = mode === "ai_vs_ai";
    const isHvAI = mode === "human_vs_ai";
    const simulationBusy = simulationRequestInFlight || isSimulationPlayback;
    const fenProvided = Boolean(String(fenInput?.value || "").trim());
    if (humanSideSelect) humanSideSelect.disabled = isAIVsAI || simulationBusy;
    if (aiGameCountInput) {
      aiGameCountInput.disabled = !isAIVsAI || simulationBusy;
      if (fenProvided) aiGameCountInput.value = "1";
    }
    if (aiStrengthSelect) {
      // Show strength selector only for modes that involve AI
      aiStrengthSelect.disabled = !(mode === "human_vs_ai" || isAIVsAI) || simulationBusy;
    }
    if (gameModeSelect) gameModeSelect.disabled = simulationBusy;
    if (gameTypeSelect) gameTypeSelect.disabled = simulationBusy;
    if (fenInput) fenInput.disabled = simulationBusy;
    if (configApplyButton) configApplyButton.disabled = simulationBusy;
    if (newGameButton) newGameButton.disabled = simulationBusy;
    if (input) input.disabled = simulationBusy || gameOver;
    if (button) button.disabled = simulationBusy || gameOver;
    if (flagButton) flagButton.disabled = simulationBusy || gameOver;

    const clockOn = Boolean(clockEnabledInput?.checked);
    if (clockEnabledInput) clockEnabledInput.disabled = simulationBusy;
    if (clockPresetSelect) clockPresetSelect.disabled = !clockOn || simulationBusy;
    if (clockIncrementSecInput) clockIncrementSecInput.disabled = !clockOn || simulationBusy;
    if (clockBaseSecInput) clockBaseSecInput.disabled = !clockOn || isHvAI || simulationBusy;
    if (clockHumanBaseSecInput) clockHumanBaseSecInput.disabled = !clockOn || !isHvAI || simulationBusy;
    if (clockAiBaseSecInput) clockAiBaseSecInput.disabled = !clockOn || !isHvAI || simulationBusy;
    if (clockHvAIFields) clockHvAIFields.style.display = isHvAI ? "" : "none";

    // Keep simulation controls sane when user leaves AI vs AI mode.
    if (!isAIVsAI && !simulationBusy && simulationData) {
      cleanupSimulationControls();
      clearSimulationSummary();
    }
    if (simRunBtn) {
      simRunBtn.style.display = isAIVsAI ? "inline-block" : "none";
      simRunBtn.disabled = simulationBusy;
    }
  };

  const geometryForGameType = (type) => {
    switch (String(type || "chess").toLowerCase()) {
      case "xianqi":
        return { files: 9, maxRank: 10, type: "xianqi" };
      case "shogi":
        return { files: 9, maxRank: 9, type: "shogi" };
      default:
        return { files: 8, maxRank: 8, type: "chess" };
    }
  };

  const rebuildBoardLabels = () => {
    const ranksEl = boardWrapper.querySelector(".board_ranks");
    if (ranksEl) {
      ranksEl.replaceChildren(
        ...Array.from({ length: boardMaxRank }, (_, i) => {
          const span = document.createElement("span");
          span.className = "board_label";
          span.textContent = String(boardMaxRank - i);
          return span;
        })
      );
    }
    const filesEl = boardWrapper.querySelector(".board_files");
    if (filesEl) {
      // Chess/Xiangqi: a..i. Shogi UI: 1..9 (not letter files).
      const numericFiles = boardGameType === "shogi";
      filesEl.replaceChildren(
        ...Array.from({ length: boardFiles }, (_, i) => {
          const span = document.createElement("span");
          span.className = "board_label";
          span.textContent = numericFiles
            ? String(i + 1)
            : String.fromCharCode("a".charCodeAt(0) + i);
          return span;
        })
      );
    }
  };

  const rebuildXiangqiBoard = () => {
    // Lines at x=i/8, y=j/9 inside .xianqi_field; board padding holds edge-piece overhang.
    boardElement.classList.add("xianqi_board");
    const field = document.createElement("div");
    field.className = "xianqi_field";

    const art = document.createElement("div");
    art.className = "xianqi_artwork";
    art.setAttribute("aria-hidden", "true");

    for (let j = 0; j <= 9; j++) {
      const h = document.createElement("div");
      h.className = "xianqi_h_line";
      h.style.top = `${(j / 9) * 100}%`;
      art.appendChild(h);
    }
    for (let i = 0; i <= 8; i++) {
      const v = document.createElement("div");
      v.className = i === 0 || i === 8 ? "xianqi_v_line xianqi_v_outer" : "xianqi_v_line xianqi_v_inner";
      v.style.left = `${(i / 8) * 100}%`;
      art.appendChild(v);
    }
    for (const side of ["top", "bottom"]) {
      const palace = document.createElement("div");
      palace.className = `xianqi_palace xianqi_palace_${side}`;
      art.appendChild(palace);
    }

    const points = document.createElement("div");
    points.className = "xianqi_points";
    for (let seq = 0; seq < 90; seq++) {
      const file = (seq % 9) + 1;
      const rank = 10 - Math.floor(seq / 9);
      const sq = document.createElement("div");
      sq.className = "chess_board_square chess_board_square_light";
      sq.setAttribute("data-sequence", String(seq));
      sq.setAttribute("data-file", String(file));
      sq.setAttribute("data-rank", String(rank));
      sq.style.left = `${((file - 1) / 8) * 100}%`;
      sq.style.top = `${((10 - rank) / 9) * 100}%`;
      points.appendChild(sq);
    }

    field.append(art, points);
    boardElement.replaceChildren(field);
  };

  const rebuildSquareGridBoard = () => {
    boardElement.classList.remove("xianqi_board");
    const n = boardFiles * boardMaxRank;
    const squares = [];
    for (let seq = 0; seq < n; seq++) {
      const file = (seq % boardFiles) + 1;
      const rank = boardMaxRank - Math.floor(seq / boardFiles);
      const row = Math.floor(seq / boardFiles);
      const col = seq % boardFiles;
      const isLight = (row + col) % 2 === 0;
      const div = document.createElement("div");
      div.className = [
        "chess_board_square",
        isLight ? "chess_board_square_light" : "chess_board_square_dark",
      ].join(" ");
      div.setAttribute("data-sequence", String(seq));
      div.setAttribute("data-file", String(file));
      div.setAttribute("data-rank", String(rank));
      squares.push(div);
    }
    boardElement.replaceChildren(...squares);
  };

  const syncXiangqiCoordGutters = () => {
    if (!boardWrapper || !boardElement) return;
    if (String(boardWrapper.dataset.gameType || "") !== "xianqi") {
      boardWrapper.style.removeProperty("--xq-board-w");
      boardWrapper.style.removeProperty("--xq-board-h");
      boardWrapper.style.removeProperty("--xq-label-pad-x");
      boardWrapper.style.removeProperty("--xq-label-pad-y");
      return;
    }
    const cs = window.getComputedStyle(boardElement);
    const padX = (parseFloat(cs.borderLeftWidth) || 0) + (parseFloat(cs.paddingLeft) || 0);
    const padY = (parseFloat(cs.borderTopWidth) || 0) + (parseFloat(cs.paddingTop) || 0);
    boardWrapper.style.setProperty("--xq-board-w", `${boardElement.offsetWidth}px`);
    boardWrapper.style.setProperty("--xq-board-h", `${boardElement.offsetHeight}px`);
    boardWrapper.style.setProperty("--xq-label-pad-x", `${padX}px`);
    boardWrapper.style.setProperty("--xq-label-pad-y", `${padY}px`);
  };

  const rebuildBoardGrid = () => {
    if (!boardElement || !boardWrapper) return;
    boardWrapper.dataset.gameType = boardGameType;
    boardWrapper.style.setProperty("--board-files", String(boardFiles));
    boardWrapper.style.setProperty("--board-ranks", String(boardMaxRank));
    rebuildBoardLabels();
    if (boardGameType === "xianqi") rebuildXiangqiBoard();
    else rebuildSquareGridBoard();
    // After layout: copy board size/padding so a..i / 10..1 sit on the grid lines.
    window.requestAnimationFrame(() => syncXiangqiCoordGutters());
  };

  const ensureBoardGeometry = (type) => {
    const g = geometryForGameType(type);
    if (g.files === boardFiles && g.maxRank === boardMaxRank && g.type === boardGameType) {
      if (boardWrapper) boardWrapper.dataset.gameType = boardGameType;
      return false;
    }
    boardFiles = g.files;
    boardMaxRank = g.maxRank;
    boardGameType = g.type;
    rebuildBoardGrid();
    return true;
  };

  /** Preview start layout for the selected game type (does not create a session). */
  const previewBoardForGameType = (type) => {
    const t = String(type || gameTypeSelect?.value || boardGameType || "chess").toLowerCase();
    ensureBoardGeometry(t);
    if (t === "xianqi") {
      renderBoardFromState(initialXiangqiState(), "xianqi");
      return;
    }
    if (t === "shogi") {
      renderBoardFromState(initialShogiState(), "shogi");
      return;
    }
    renderBoardFromState(initialChessState(), "chess");
  };

  // syncClockSetup: only after create / Apply / New Game. Mid-game GET/flag must not
  // overwrite the setup form (blocks editing TC for the next game after an ending).
  const renderGameConfig = (game, opts = {}) => {
    if (!game) return;
    // Always sync board geometry from game type (config may be sparse).
    if (gameTypeSelect) gameTypeSelect.value = String(game.type || "chess");
    ensureBoardGeometry(game.type || gameTypeSelect?.value || "chess");
    const cfg = game.config;
    if (!cfg) {
      updateSetupControlState();
      return;
    }
    if (gameModeSelect) gameModeSelect.value = String(game.mode || "human_vs_human");
    if (humanSideSelect) humanSideSelect.value = String(cfg.humanColor || "white");
    if (aiGameCountInput) aiGameCountInput.value = String(cfg.aiGameCount || 1);
    if (fenInput) fenInput.value = String(cfg.startFen || "");
    if (aiStrengthSelect) aiStrengthSelect.value = String(cfg.aiProfile || cfg.aiStrength || "intermediate");
    if (coachLevelSelect) {
      const skill = String(cfg.skillLevel || "").toLowerCase();
      coachLevelSelect.value =
        skill === "beginner" || skill === "intermediate" || skill === "advanced"
          ? skill
          : "intermediate";
    }
    humanColor = String(cfg.humanColor || "white").toLowerCase();
    if (opts.syncClockSetup) syncClockControlsFromGame(game);
    updateSetupControlState();
  };

  const CHESS_PIECE_ORDER = ["queen", "rook", "bishop", "knight", "pawn", "king"];
  const XIANQI_PIECE_ORDER = ["cannon", "rook", "knight", "elephant", "advisor", "pawn", "king"];
  // Unpromoted hand kinds only (promoted captives demote before entering hand).
  const SHOGI_PIECE_ORDER = ["rook", "bishop", "gold", "silver", "knight", "lance", "pawn"];
  const SHOGI_DROP_CHAR = {
    pawn: "P", lance: "L", knight: "N", silver: "S", gold: "G", bishop: "B", rook: "R",
  };
  const capturedPieceOrder = () => {
    if (boardGameType === "xianqi") return XIANQI_PIECE_ORDER;
    if (boardGameType === "shogi") return SHOGI_PIECE_ORDER;
    return CHESS_PIECE_ORDER;
  };

  const isHandSidePlayable = (side) => {
    const s = String(side || "").toLowerCase();
    if (s !== currentTurn) return false;
    const mode = String(gameModeSelect?.value || "");
    if (mode === "human_vs_ai" && s !== humanColor) return false;
    return true;
  };

  // Shogi: icons are droppable hand pieces (own colour). Chess/Xiangqi: icons are captives (opponent colour).
  const renderCapturedIcons = (el, side, captured) => {
    if (!el) return;
    el.replaceChildren();
    el.classList.add("shogi_hand");
    const droppable = boardGameType === "shogi";
    const iconColor = droppable ? side : side === "white" ? "black" : "white";
    let any = false;
    for (const kind of capturedPieceOrder()) {
      const count = captured[kind] || 0;
      if (count <= 0) continue;
      const path = imagePathFromPiece({ kind, color: iconColor });
      if (!path) continue;
      any = true;
      const node = document.createElement(droppable ? "button" : "span");
      if (droppable) node.type = "button";
      node.className = "shogi_hand_piece";
      node.setAttribute("data-side", side);
      node.setAttribute("data-kind", kind);
      node.title = `${kind} ×${count}`;
      if (
        droppable &&
        selectedDropKind &&
        selectedDropKind.side === side &&
        selectedDropKind.kind === kind
      ) {
        node.classList.add("shogi_hand_piece_selected");
      }
      if (droppable) applyHandHintToNode(node, side, kind);
      const img = document.createElement("img");
      img.src = path;
      img.alt = kind;
      // Shogi only: black hand pieces are rotated (same art). Xiangqi/chess use colour-specific art.
      if (droppable) img.setAttribute("data-color", iconColor);
      img.draggable = false;
      const badge = document.createElement("span");
      badge.className = "shogi_hand_count";
      badge.textContent = String(count);
      node.appendChild(img);
      node.appendChild(badge);
      if (droppable) {
        node.disabled = !isHandSidePlayable(side);
        node.addEventListener("click", (event) => {
          event.preventDefault();
          event.stopPropagation();
          void selectShogiHandPiece(side, kind);
        });
      }
      el.appendChild(node);
    }
    if (!any) {
      el.classList.remove("shogi_hand");
    }
  };

  const emptyCapturedSide = () => {
    const side = {};
    for (const kind of capturedPieceOrder()) side[kind] = 0;
    return side;
  };

  const normalizeCapturedSummary = (summary) => {
    const order = capturedPieceOrder();
    if (!summary || typeof summary !== "object") {
      return { white: emptyCapturedSide(), black: emptyCapturedSide() };
    }
    const normalized = { white: emptyCapturedSide(), black: emptyCapturedSide() };
    for (const side of ["white", "black"]) {
      const source = summary[side];
      if (!source || typeof source !== "object") continue;
      for (const kind of order) {
        const value = Number(source[kind]);
        normalized[side][kind] = Number.isFinite(value) && value > 0 ? value : 0;
      }
    }
    return normalized;
  };

  const clampPercentage = (value) => {
    const n = Number(value);
    if (!Number.isFinite(n)) return 50;
    return Math.max(0, Math.min(100, n));
  };

  const formatPercentage = (value) => `${value.toFixed(1)}%`;

  const winProbLabelColor = (chance, isLightBackground) => {
    if (chance >= 70) return isLightBackground ? "#0f5e2a" : "#8df0a8";
    if (chance <= 30) return isLightBackground ? "#7a1e1e" : "#ff9f9f";
    return isLightBackground ? "#101010" : "#f5f5f5";
  };

  const fromAnalyzerChance = (value) => {
    const n = Number(value);
    if (!Number.isFinite(n)) return null;
    // Analyzer uses 0..1; tolerate 0..100 values too.
    return n <= 1 ? n * 100 : n;
  };

  const renderGameInfo = (capturedSummary, analysis) => {
    if (capturedSummary) cachedCapturedSummary = capturedSummary;
    const effectiveCapturedSummary = capturedSummary || cachedCapturedSummary;
    const effectiveAnalysis = analysis || cachedAnalysis;
    if (analysis) cachedAnalysis = analysis;
    const normalizedCaptured = normalizeCapturedSummary(effectiveCapturedSummary);
    const whiteCaptured = normalizedCaptured.white;
    const blackCaptured = normalizedCaptured.black;
    const analyzerWhite = fromAnalyzerChance(effectiveAnalysis?.win_chance_white);
    const analyzerBlack = fromAnalyzerChance(effectiveAnalysis?.win_chance_black);
    const hasAnalyzerProb = analyzerWhite != null && analyzerBlack != null;
    const whiteProb = clampPercentage(hasAnalyzerProb ? analyzerWhite : 50);
    const blackProb = clampPercentage(hasAnalyzerProb ? analyzerBlack : 50);
    const whiteTiny = whiteProb < 12;
    const blackTiny = blackProb < 12;

    renderCapturedIcons(capturedWhiteValue, "white", whiteCaptured);
    renderCapturedIcons(capturedBlackValue, "black", blackCaptured);
    if (winProbWhiteValue) {
      winProbWhiteValue.textContent = formatPercentage(whiteProb);
      winProbWhiteValue.style.color = winProbLabelColor(whiteProb, true);
      winProbWhiteValue.classList.toggle("game_info_winprob_label_outside_white", whiteTiny);
    }
    if (winProbBlackValue) {
      winProbBlackValue.textContent = formatPercentage(blackProb);
      winProbBlackValue.style.color = winProbLabelColor(blackProb, false);
      winProbBlackValue.classList.toggle("game_info_winprob_label_outside_black", blackTiny);
    }
    if (winProbWhiteBar) winProbWhiteBar.style.width = `${whiteProb}%`;
    if (winProbBlackBar) winProbBlackBar.style.width = `${blackProb}%`;
    if (winProbWhiteBar) winProbWhiteBar.classList.toggle("game_info_winprob_segment_tiny", whiteTiny);
    if (winProbBlackBar) winProbBlackBar.classList.toggle("game_info_winprob_segment_tiny", blackTiny);

    if (gameInfoNotesBox && effectiveAnalysis && !gameOver) {
      const threatSummary = String(effectiveAnalysis?.threat_summary || "").trim();
      // Skip empty / stub lines that looked like Fairy-Stockfish authored the coach note.
      const stubThreat = new Set([
        "",
        "Position evaluated with Fairy-Stockfish.",
        "No analysis summary yet.",
      ]);
      lastThreatSummary = stubThreat.has(threatSummary) ? "" : threatSummary;
      refreshNotesBox();
    }
  };

  const startAnalysisPolling = (targetMoveNumber, capturedSnapshot) => {
    stopAnalysisPolling();
    // Keep composed notes (suggestions + coach Thinking…); analysis will refresh threat/suggestions.
    const target = Number(targetMoveNumber) || 0;
    pendingAnalysisTargetMove = target;
    pendingAnalysisCapturedSnapshot = capturedSnapshot || cachedCapturedSummary;

    if (isSocketConnected()) {
      analysisPollFallbackTimer = window.setTimeout(() => {
        if (!isSocketConnected() && pendingAnalysisTargetMove > 0) {
          startAnalysisPolling(pendingAnalysisTargetMove, pendingAnalysisCapturedSnapshot);
        }
      }, 1500);
      return;
    }

    const pollOnce = async () => {
      try {
        if (!currentGameId) return;
        const response = await fetch(
          `/api/games/${encodeURIComponent(currentGameId)}/analysis/latest`,
          { method: "GET" }
        );
        if (!response.ok) return;
        const payload = await response.json();
        const latestMoveNumber = Number(payload?.latest_move_number || 0);
        const latestAnalysis = payload?.latest?.analysis;
        if (!latestAnalysis) return;
        if (latestMoveNumber < target) return;
        renderGameInfo(pendingAnalysisCapturedSnapshot || capturedSnapshot, latestAnalysis);
        stopAnalysisPolling();
      } catch (_) {
        // ignore polling errors; next poll may recover
      }
    };

    void pollOnce();
    analysisPollTimer = window.setInterval(() => {
      void pollOnce();
    }, 700);
  };

  const movePieceIcon = (side, pieceKind) => {
    const color = String(side || "").toLowerCase() === "black" ? "black" : "white";
    const kind = String(pieceKind || "").toLowerCase();
    const iconMap = {
      white: {
        pawn: "♙", rook: "♖", knight: "♘", bishop: "♗", queen: "♕", king: "♔",
        // Xiangqi API kinds (unicode fallback when not using piece PNGs)
        cannon: "砲", advisor: "仕", elephant: "相",
        lance: "L", silver: "S", gold: "G",
        promoted_pawn: "+P", promoted_lance: "+L", promoted_knight: "+N",
        promoted_silver: "+S", dragon: "D", horse: "H",
      },
      black: {
        pawn: "♟", rook: "♜", knight: "♞", bishop: "♝", queen: "♛", king: "♚",
        cannon: "炮", advisor: "士", elephant: "象",
        lance: "l", silver: "s", gold: "g",
        promoted_pawn: "+p", promoted_lance: "+l", promoted_knight: "+n",
        promoted_silver: "+s", dragon: "d", horse: "h",
      },
    };
    return iconMap[color]?.[kind] || kind.slice(0, 1).toUpperCase() || "?";
  };

  const fillHistoryPieceIcon = (el, side, pieceKind) => {
    el.className = "chess_move_history_piece_icon";
    el.replaceChildren();
    if (boardGameType === "xianqi" || boardGameType === "shogi") {
      const path = imagePathFromPiece({ kind: pieceKind, color: side });
      if (path) {
        const img = document.createElement("img");
        img.src = path;
        img.alt = String(pieceKind || "");
        img.setAttribute("data-color", String(side || "").toLowerCase());
        el.appendChild(img);
        return;
      }
    }
    el.textContent = movePieceIcon(side, pieceKind);
  };

  const opponentSide = (side) =>
    String(side || "").toLowerCase() === "black" ? "white" : "black";

  const destinationFromCommand = (command) => {
    const text = String(command || "").trim().toLowerCase();
    if (!text) return "";
    // Chess a-h/1-8 (+ promo); Xiangqi/Shogi a-i and ranks to 10 (+ optional '+')
    const match = text.match(/([a-i]\d{1,2})(?:[qrbn]|\+)?$/i);
    return match ? match[1] : text;
  };

  const appendHistoryMove = (listEl, side, pieceKind, toSquare, fallbackText, isCapture, capturedPieceKind) => {
    const item = document.createElement("li");
    const iconSpan = document.createElement("span");
    fillHistoryPieceIcon(iconSpan, side, pieceKind);
    const textSpan = document.createElement("span");
    textSpan.className = "chess_move_history_move_text";
    const moveText = toSquare || fallbackText || "";
    if (isCapture) {
      textSpan.textContent = `${moveText} x `;
      if (capturedPieceKind) {
        const capturedIcon = document.createElement("span");
        fillHistoryPieceIcon(capturedIcon, opponentSide(side), capturedPieceKind);
        textSpan.appendChild(capturedIcon);
      }
    } else {
      textSpan.textContent = moveText;
    }
    item.appendChild(iconSpan);
    item.appendChild(document.createTextNode(" "));
    item.appendChild(textSpan);
    listEl.appendChild(item);
  };

  const clearHistoryPlaceholder = (listEl) => {
    const placeholder = listEl.querySelector(".chess_move_history_placeholder");
    if (placeholder) placeholder.remove();
  };

  // update move history from backend source of truth
  const renderMoveHistory = (history, historyDetailed) => {
    moveHistoryWhiteList.innerHTML = "";
    moveHistoryBlackList.innerHTML = "";
    if ((!Array.isArray(history) || history.length === 0) && (!Array.isArray(historyDetailed) || historyDetailed.length === 0)) {
      const whitePlaceholder = document.createElement("li");
      whitePlaceholder.className = "chess_move_history_placeholder";
      whitePlaceholder.textContent = "No moves yet.";
      moveHistoryWhiteList.appendChild(whitePlaceholder);

      const blackPlaceholder = document.createElement("li");
      blackPlaceholder.className = "chess_move_history_placeholder";
      blackPlaceholder.textContent = "No moves yet.";
      moveHistoryBlackList.appendChild(blackPlaceholder);
      return;
    }

    if (Array.isArray(historyDetailed) && historyDetailed.length > 0) {
      for (const move of historyDetailed) {
        const side = String(move?.side || "white");
        const toSquare = String(move?.to || "");
        const pieceKind = String(move?.pieceKind || "pawn");
        const fallbackText = destinationFromCommand(move?.command);
        const isCapture = Boolean(move?.isCapture);
        const capturedPieceKind = String(move?.capturedPieceKind || "");
        if (side.toLowerCase() === "black") {
          appendHistoryMove(moveHistoryBlackList, side, pieceKind, toSquare, fallbackText, isCapture, capturedPieceKind);
        } else {
          appendHistoryMove(moveHistoryWhiteList, side, pieceKind, toSquare, fallbackText, isCapture, capturedPieceKind);
        }
      }
    } else if (Array.isArray(history)) {
      for (const move of history) {
        if (move.startsWith("White:")) {
          const commandText = move.replace(/^White:\s*/, "");
          appendHistoryMove(moveHistoryWhiteList, "white", "pawn", destinationFromCommand(commandText), commandText, false, "");
        } else if (move.startsWith("Black:")) {
          const commandText = move.replace(/^Black:\s*/, "");
          appendHistoryMove(moveHistoryBlackList, "black", "pawn", destinationFromCommand(commandText), commandText, false, "");
        } else {
          const commandText = String(move || "");
          appendHistoryMove(moveHistoryWhiteList, "white", "pawn", destinationFromCommand(commandText), commandText, false, "");
        }
      }
    }

    if (!moveHistoryWhiteList.children.length) {
      const whitePlaceholder = document.createElement("li");
      whitePlaceholder.className = "chess_move_history_placeholder";
      whitePlaceholder.textContent = "No moves yet.";
      moveHistoryWhiteList.appendChild(whitePlaceholder);
    }
    if (!moveHistoryBlackList.children.length) {
      const blackPlaceholder = document.createElement("li");
      blackPlaceholder.className = "chess_move_history_placeholder";
      blackPlaceholder.textContent = "No moves yet.";
      moveHistoryBlackList.appendChild(blackPlaceholder);
    }

    moveHistoryWhiteList.scrollTop = moveHistoryWhiteList.scrollHeight;
    moveHistoryBlackList.scrollTop = moveHistoryBlackList.scrollHeight;
  };

  const maxSequence = () => boardFiles * boardMaxRank - 1;

  const sequenceToSquare = (sequence) => {
    const seq = Number(sequence);
    if (Number.isNaN(seq) || seq < 0 || seq > maxSequence()) return "";
    const fileChar = String.fromCharCode("a".charCodeAt(0) + (seq % boardFiles));
    const rankNum = boardMaxRank - Math.floor(seq / boardFiles);
    return `${fileChar}${rankNum}`;
  };

  const moveCommandFromSequence = (fromSequence, toSequence) => {
    const fromSquare = sequenceToSquare(fromSequence);
    const toSquare = sequenceToSquare(toSequence);
    if (!fromSquare || !toSquare) return "";
    return `${fromSquare}${toSquare}`;
  };

  const rankFromSequence = (sequence) => {
    const seq = Number(sequence);
    if (Number.isNaN(seq) || seq < 0 || seq > maxSequence()) return NaN;
    return boardMaxRank - Math.floor(seq / boardFiles);
  };

  const fileRankFromSequence = (sequence) => {
    const seq = Number(sequence);
    if (Number.isNaN(seq) || seq < 0 || seq > maxSequence()) return null;
    return {
      file: (seq % boardFiles) + 1,
      rank: boardMaxRank - Math.floor(seq / boardFiles),
    };
  };

  // Helper to apply a UCI move string (e.g. "e2e4", "h3h10") to the visual board DOM.
  const applyUciMoveToBoard = (uci) => {
    if (!uci || uci.length < 4) return;
    const match = String(uci).match(/^([a-i])(\d{1,2})([a-i])(\d{1,2})/i);
    if (!match) return;
    const fromFile = match[1].toLowerCase().charCodeAt(0) - "a".charCodeAt(0) + 1;
    const fromRank = parseInt(match[2], 10);
    const toFile = match[3].toLowerCase().charCodeAt(0) - "a".charCodeAt(0) + 1;
    const toRank = parseInt(match[4], 10);
    if (
      fromFile < 1 || fromFile > boardFiles || fromRank < 1 || fromRank > boardMaxRank ||
      toFile < 1 || toFile > boardFiles || toRank < 1 || toRank > boardMaxRank
    ) {
      return;
    }
    const fromSeq = sequenceByFileRank(fromFile, fromRank);
    const toSeq = sequenceByFileRank(toFile, toRank);

    const fromEl = boardElement.querySelector(`.chess_board_square[data-sequence="${fromSeq}"]`);
    const toEl = boardElement.querySelector(`.chess_board_square[data-sequence="${toSeq}"]`);

    if (!fromEl || !toEl) return;

    const piece = fromEl.querySelector(".piece_img");
    if (!piece) return;

    const captured = toEl.querySelector(".piece_img");
    if (captured) captured.remove();

    toEl.appendChild(piece);
  };

  const getSquareElement = (target) =>
    target instanceof Element ? target.closest(".chess_board_square[data-sequence]") : null;

  const getSquareSequence = (square) => {
    if (!square) return NaN;
    return Number(square.getAttribute("data-sequence"));
  };

  const getPieceOnSquare = (square) => square?.querySelector(".piece_img") || null;

  const isCurrentTurnPiece = (square) => {
    const piece = getPieceOnSquare(square);
    if (!piece) return false;
    const pieceColor = String(piece.getAttribute("data-color") || "").toLowerCase();

    // In Human vs AI mode, only allow the human to move their chosen color
    // (use the stored humanColor from the game config, not the select box)
    const mode = String(gameModeSelect?.value || "");
    if (mode === "human_vs_ai") {
      if (pieceColor !== humanColor) {
        return false;
      }
    }

    return pieceColor === currentTurn;
  };

  const legalMoveAt = (toSequence) => {
    const target = fileRankFromSequence(toSequence);
    if (!target) return null;
    return (
      selectedLegalMoves.find(
        (move) => Number(move?.file) === target.file && Number(move?.rank) === target.rank
      ) || null
    );
  };

  const requiresPromotion = (toSequence) => {
    if (boardGameType !== "chess") return false;
    return Boolean(legalMoveAt(toSequence)?.requiresPromotion);
  };

  const shogiPromotionFlags = (toSequence) => {
    const move = legalMoveAt(toSequence);
    if (!move) return { must: false, can: false };
    return {
      must: Boolean(move.requiresPromotion),
      can: Boolean(move.canPromote),
    };
  };

  const closePromotionPicker = () => {
    if (!promotionPicker) return;
    promotionPicker.classList.remove("promotion_picker_visible");
    promotionPicker.classList.add("promotion_picker_hidden");
    promotionPicker.setAttribute("aria-hidden", "true");
  };

  const openPromotionPicker = () => {
    if (!promotionPicker) return;
    promotionPicker.classList.remove("promotion_picker_hidden");
    promotionPicker.classList.add("promotion_picker_visible");
    promotionPicker.setAttribute("aria-hidden", "false");
  };

  const configurePromotionPicker = (mode) => {
    if (!promotionPicker) return;
    const title = promotionPicker.querySelector("#promotion_picker_title");
    const choices = promotionPicker.querySelector(".promotion_picker_choices");
    if (!title || !choices) return;
    if (mode === "shogi") {
      title.textContent = "Promote this piece?";
      choices.innerHTML =
        `<button type="button" class="promotion_choice_btn" data-promotion="+">Promote</button>` +
        `<button type="button" class="promotion_choice_btn" data-promotion="-">Do not promote</button>`;
      return;
    }
    title.textContent = "Choose promotion piece";
    choices.innerHTML =
      `<button type="button" class="promotion_choice_btn" data-promotion="q">Queen</button>` +
      `<button type="button" class="promotion_choice_btn" data-promotion="r">Rook</button>` +
      `<button type="button" class="promotion_choice_btn" data-promotion="b">Bishop</button>` +
      `<button type="button" class="promotion_choice_btn" data-promotion="n">Knight</button>`;
  };

  // mode: "chess" | "shogi". Cancel → ""; chess piece letter; shogi "+" or "-".
  const requestPromotionChoice = (mode = "chess") =>
    new Promise((resolve) => {
      if (!promotionPicker) {
        resolve(mode === "shogi" ? "+" : "q");
        return;
      }
      configurePromotionPicker(mode);
      pendingPromotionResolve = resolve;
      openPromotionPicker();
    });

  const resolvePromotionChoice = (pieceCode) => {
    if (!pendingPromotionResolve) return;
    const resolver = pendingPromotionResolve;
    pendingPromotionResolve = null;
    closePromotionPicker();
    resolver(pieceCode);
  };

  const clearSelectedSquare = () => {
    selectedSquareSequence = null;
    selectedDropKind = null;
    selectedLegalMoves = [];
    legalMovesRequestVersion += 1;
    boardElement
      .querySelectorAll(`.piece_img.${SELECTED_PIECE_CLASS}`)
      .forEach((piece) => piece.classList.remove(SELECTED_PIECE_CLASS));
    boardElement
      .querySelectorAll(`.${LEGAL_DESTINATION_CLASS}, .${LEGAL_PROMOTION_DESTINATION_CLASS}, .${LEGAL_CAPTURE_DESTINATION_CLASS}`)
      .forEach((square) => {
        square.classList.remove(LEGAL_DESTINATION_CLASS);
        square.classList.remove(LEGAL_PROMOTION_DESTINATION_CLASS);
        square.classList.remove(LEGAL_CAPTURE_DESTINATION_CLASS);
      });
    document.querySelectorAll(".shogi_hand_piece_selected").forEach((el) => {
      el.classList.remove("shogi_hand_piece_selected");
    });
    // Do not clear FS suggestion highlights here; they are independent of piece selection.
  };

  const selectShogiHandPiece = async (side, kind) => {
    if (boardGameType !== "shogi" || gameOver || isSubmitting) return;
    if (!isHandSidePlayable(side)) return;
    if (
      selectedDropKind &&
      selectedDropKind.side === side &&
      selectedDropKind.kind === kind
    ) {
      clearSelectedSquare();
      return;
    }
    clearSelectedSquare();
    selectedDropKind = { side, kind };
    document
      .querySelectorAll(`.shogi_hand_piece[data-side="${side}"][data-kind="${kind}"]`)
      .forEach((el) => el.classList.add("shogi_hand_piece_selected"));
    const requestVersion = ++legalMovesRequestVersion;
    try {
      if (!currentGameId) return;
      const response = await fetch(
        `/api/games/${encodeURIComponent(currentGameId)}/legal-moves?dropKind=${encodeURIComponent(kind)}`
      );
      if (!response.ok) {
        if (requestVersion === legalMovesRequestVersion) highlightLegalDestinations([]);
        return;
      }
      const result = await response.json();
      if (requestVersion !== legalMovesRequestVersion) return;
      if (!selectedDropKind || selectedDropKind.kind !== kind) return;
      const moves = Array.isArray(result?.legalMoves) ? result.legalMoves : [];
      selectedLegalMoves = moves;
      highlightLegalDestinations(moves);
    } catch (_error) {
      if (requestVersion === legalMovesRequestVersion) highlightLegalDestinations([]);
    }
  };

  const submitShogiDrop = async (toSequence) => {
    if (!selectedDropKind) return false;
    const target = fileRankFromSequence(toSequence);
    if (!target) return false;
    const legal = selectedLegalMoves.some(
      (move) => Number(move?.file) === target.file && Number(move?.rank) === target.rank
    );
    if (!legal) return false;
    const ch = SHOGI_DROP_CHAR[selectedDropKind.kind];
    if (!ch) return false;
    const fileLetter = String.fromCharCode("a".charCodeAt(0) + target.file - 1);
    return submitCommand(`${ch}*${fileLetter}${target.rank}`);
  };

  const highlightLegalDestinations = (moves) => {
    boardElement
      .querySelectorAll(`.${LEGAL_DESTINATION_CLASS}, .${LEGAL_PROMOTION_DESTINATION_CLASS}, .${LEGAL_CAPTURE_DESTINATION_CLASS}`)
      .forEach((square) => {
        square.classList.remove(LEGAL_DESTINATION_CLASS);
        square.classList.remove(LEGAL_PROMOTION_DESTINATION_CLASS);
        square.classList.remove(LEGAL_CAPTURE_DESTINATION_CLASS);
      });
    if (!Array.isArray(moves)) return;
    const selectedSource = fileRankFromSequence(selectedSquareSequence);
    const selectedSquare = selectedSource
      ? boardElement.querySelector(
          `.chess_board_square[data-sequence="${sequenceByFileRank(selectedSource.file, selectedSource.rank)}"]`
        )
      : null;
    const selectedPiece = getPieceOnSquare(selectedSquare);
    const selectedPieceKind = String(selectedPiece?.getAttribute("data-kind") || "").toLowerCase();
    for (const move of moves) {
      const fileNum = Number(move?.file);
      const rankNum = Number(move?.rank);
      if (Number.isNaN(fileNum) || Number.isNaN(rankNum)) continue;
      const sequence = sequenceByFileRank(fileNum, rankNum);
      const destinationSquare = boardElement.querySelector(
        `.chess_board_square[data-sequence="${sequence}"]`
      );
      if (!destinationSquare) continue;
      const isCapture = Boolean(move?.isCapture);
      if (isCapture) {
        let markerSquare = destinationSquare;
        // En passant: destination is empty, captured pawn is on source rank.
        const destinationPiece = getPieceOnSquare(destinationSquare);
        if (
          !destinationPiece &&
          selectedSource &&
          selectedPieceKind === "pawn" &&
          selectedSource.file !== fileNum
        ) {
          const capturedSequence = sequenceByFileRank(fileNum, selectedSource.rank);
          const capturedSquare = boardElement.querySelector(
            `.chess_board_square[data-sequence="${capturedSequence}"]`
          );
          if (capturedSquare) markerSquare = capturedSquare;
        }
        markerSquare.classList.add(LEGAL_CAPTURE_DESTINATION_CLASS);
      } else {
        destinationSquare.classList.add(LEGAL_DESTINATION_CLASS);
      }
      if (Boolean(move?.requiresPromotion) || Boolean(move?.canPromote)) {
        destinationSquare.classList.add(LEGAL_PROMOTION_DESTINATION_CLASS);
      }
    }
  };

  const loadLegalDestinationsForSelection = async (sequence) => {
    const source = fileRankFromSequence(sequence);
    if (!source) {
      highlightLegalDestinations([]);
      return;
    }
    const requestVersion = ++legalMovesRequestVersion;
    try {
      if (!currentGameId) return;
      const response = await fetch(
        `/api/games/${encodeURIComponent(currentGameId)}/legal-moves?file=${source.file}&rank=${source.rank}`
      );
      if (!response.ok) {
        if (requestVersion === legalMovesRequestVersion) {
          highlightLegalDestinations([]);
        }
        return;
      }
      const result = await response.json();
      if (requestVersion !== legalMovesRequestVersion) return;
      if (selectedSquareSequence !== Number(sequence)) return;
      const moves = Array.isArray(result?.legalMoves) ? result.legalMoves : [];
      selectedLegalMoves = moves;
      highlightLegalDestinations(moves);
    } catch (_error) {
      if (requestVersion === legalMovesRequestVersion) {
        highlightLegalDestinations([]);
      }
    }
  };

  let selectedSuggestedMoves = [];
  /** @type {{ side: string, kind: string, rank: string }[]} */
  let lastHandHints = [];

  const parseUciMove = (move) => {
    const raw = String(move || "").trim().toLowerCase();
    const board = raw.match(/^([a-i])(\d{1,2})([a-i])(\d{1,2})(\+?)$/);
    if (board) {
      return {
        from: { file: board[1].charCodeAt(0) - 96, rank: Number(board[2]) },
        to: { file: board[3].charCodeAt(0) - 96, rank: Number(board[4]) },
        dropKind: null,
        promote: board[5] === "+",
      };
    }
    const drop = raw.match(/^([plnsgbr])[*@]([a-i])([1-9])$/);
    if (drop) {
      return {
        from: null,
        to: { file: drop[2].charCodeAt(0) - 96, rank: Number(drop[3]) },
        dropKind: drop[1].toUpperCase(),
        promote: false,
      };
    }
    return null;
  };

  const squareAt = (file, rank) => {
    if (file < 1 || file > boardFiles || rank < 1 || rank > boardMaxRank) return null;
    const sequence = sequenceByFileRank(file, rank);
    return boardElement.querySelector(`.chess_board_square[data-sequence="${sequence}"]`);
  };

  const pieceKindAt = (file, rank) => {
    const sq = squareAt(file, rank);
    const kind = sq?.querySelector(".piece_img")?.getAttribute("data-kind");
    return kind ? String(kind) : "piece";
  };

  const formatHintLine = (rank, parsed, scoreCp) => {
    const sc =
      typeof scoreCp === "number" ? ` (${scoreCp > 0 ? "+" : ""}${scoreCp})` : "";
    if (!parsed) return `${rank}. ????${sc}`;
    const toLab = `${String.fromCharCode(96 + parsed.to.file)}${parsed.to.rank}`;
    if (parsed.dropKind) {
      const kind =
        SHOGI_DROP_KIND_FROM_CHAR[parsed.dropKind.toLowerCase()] || parsed.dropKind;
      return `${rank}. drop ${kind} from hand → ${toLab}${sc}`;
    }
    const fromLab = `${String.fromCharCode(96 + parsed.from.file)}${parsed.from.rank}`;
    const kind = pieceKindAt(parsed.from.file, parsed.from.rank);
    const promo = parsed.promote ? " (promote)" : "";
    return `${rank}. ${kind} ${fromLab} → ${toLab}${promo}${sc}`;
  };

  const clearSuggestedHighlights = () => {
    boardElement
      .querySelectorAll(`.${SUGGESTED_MOVE_CLASS}, .${SUGGESTED_FROM_CLASS}, .${SUGGESTED_DROP_CLASS}`)
      .forEach((square) => {
        square.classList.remove(SUGGESTED_MOVE_CLASS, SUGGESTED_FROM_CLASS, SUGGESTED_DROP_CLASS);
        square.removeAttribute("data-hint-rank");
      });
    document.querySelectorAll(`.${HAND_HINT_CLASS}`).forEach((el) => {
      el.classList.remove(HAND_HINT_CLASS);
      el.removeAttribute("data-hint-rank");
    });
    lastHandHints = [];
  };

  const appendHintRank = (el, rankLabel) => {
    if (!el) return;
    const label = String(rankLabel || "").trim();
    if (!label) return;
    const prev = el.getAttribute("data-hint-rank");
    if (!prev) {
      el.setAttribute("data-hint-rank", label);
      return;
    }
    const parts = prev.split(/[·,/]/).map((s) => s.trim()).filter(Boolean);
    if (!parts.includes(label)) parts.push(label);
    parts.sort((a, b) => Number(a) - Number(b));
    el.setAttribute("data-hint-rank", parts.join("·"));
  };

  const mergeHandHintRank = (side, kind, rankLabel) => {
    const hit = lastHandHints.find((h) => h.side === side && h.kind === kind);
    if (!hit) {
      lastHandHints.push({ side, kind, rank: String(rankLabel) });
      return;
    }
    const parts = String(hit.rank).split(/[·,/]/).map((s) => s.trim()).filter(Boolean);
    if (!parts.includes(String(rankLabel))) parts.push(String(rankLabel));
    parts.sort((a, b) => Number(a) - Number(b));
    hit.rank = parts.join("·");
  };

  const applyHandHintToNode = (node, side, kind) => {
    const hit = lastHandHints.find((h) => h.side === side && h.kind === kind);
    if (!hit) return;
    node.classList.add(HAND_HINT_CLASS);
    node.setAttribute("data-hint-rank", hit.rank);
  };

  const refreshSuggestedMoves = async (retry = true) => {
    if (isSimulationPlayback) return;
    if (!currentGameId || gameOver) {
      return;
    }
    try {
      const profile = String(aiStrengthSelect?.value || "intermediate");
      const url = `/api/games/${encodeURIComponent(currentGameId)}/top-moves?profile=${encodeURIComponent(profile)}&k=3`;
      const resp = await fetch(url);
      if (!resp.ok) {
        // Transient 503 (engine starting) or 404/500 — retry once; keep previous suggestions (no clear/flicker).
        if (retry && !gameOver) {
          window.setTimeout(() => {
            void refreshSuggestedMoves(false);
          }, 1200);
        }
        return;
      }
      if (gameOver) return;
      const data = await resp.json();
      if (gameOver) return;
      const suggestions = Array.isArray(data?.suggestions) ? data.suggestions : [];
      if (suggestions.length) {
        highlightSuggestedMoves(suggestions);
      }
      // Empty payload: keep lastSuggestionsText until a non-empty update arrives.
    } catch (_) {
      if (retry && !gameOver) {
        window.setTimeout(() => {
          void refreshSuggestedMoves(false);
        }, 1200);
      }
    }
  };

  const highlightSuggestedMoves = (suggestions) => {
    clearSuggestedHighlights();

    selectedSuggestedMoves = [];
    const top = Array.isArray(suggestions) ? suggestions.slice(0, 3) : [];

    if (!top.length) {
      // Keep prior notes text; clearing here made Suggested moves jump/blank between plies.
      refreshNotesBox();
      return;
    }

    // Board move: amber origin + blue dest. Drop: highlight hand chip + dashed dest.
    top.forEach((sug, idx) => {
      const parsed = parseUciMove(sug?.move || "");
      if (!parsed) return;
      const move = sug.move || "";
      const rankLabel = String(idx + 1);

      if (parsed.dropKind) {
        const kind = SHOGI_DROP_KIND_FROM_CHAR[parsed.dropKind.toLowerCase()];
        const side = String(currentTurn || "white").toLowerCase();
        if (kind) {
          mergeHandHintRank(side, kind, rankLabel);
          document
            .querySelectorAll(`.shogi_hand_piece[data-side="${side}"][data-kind="${kind}"]`)
            .forEach((el) => applyHandHintToNode(el, side, kind));
        }
      } else if (parsed.from) {
        const fromSq = squareAt(parsed.from.file, parsed.from.rank);
        if (fromSq) {
          fromSq.classList.add(SUGGESTED_FROM_CLASS);
          appendHintRank(fromSq, rankLabel);
        }
      }

      const toSq = squareAt(parsed.to.file, parsed.to.rank);
      if (toSq) {
        toSq.classList.add(SUGGESTED_MOVE_CLASS);
        if (parsed.dropKind) toSq.classList.add(SUGGESTED_DROP_CLASS);
        appendHintRank(toSq, rankLabel);
        selectedSuggestedMoves.push({
          sequence: Number(toSq.getAttribute("data-sequence")),
          move,
        });
      }
    });

    const gt = String(boardGameType || gameTypeSelect?.value || "chess").toLowerCase();
    const header =
      gt === "shogi"
        ? "Suggested moves (including drops from hand):\n"
        : "Suggested moves:\n";
    let text = header;
    top.forEach((sug, idx) => {
      text += `${formatHintLine(idx + 1, parseUciMove(sug?.move || ""), sug.score_cp)}\n`;
    });
    lastSuggestionsText = text.trim();
    refreshNotesBox();
  };

  const loadSuggestedMovesForSelection = async (sequence) => {
    if (isSimulationPlayback || gameOver) return; // Suppress during simulation / after end
    if (!currentGameId) {
      highlightSuggestedMoves([]);
      return;
    }
    const source = fileRankFromSequence(sequence);
    if (!source) {
      highlightSuggestedMoves([]);
      return;
    }
    try {
      const profile = String(aiStrengthSelect?.value || "intermediate");
      const url = `/api/games/${encodeURIComponent(currentGameId)}/top-moves?profile=${encodeURIComponent(profile)}&k=3`;
      const resp = await fetch(url);
      if (!resp.ok) {
        highlightSuggestedMoves([]);
        return;
      }
      const data = await resp.json();
      const suggestions = Array.isArray(data?.suggestions) ? data.suggestions : [];
      highlightSuggestedMoves(suggestions);
      if (suggestions.length) {
        console.log("[hints] showing", suggestions.length, "FS suggestions");
      }
    } catch (_) {
      highlightSuggestedMoves([]);
    }
  };

  const setSelectedSquare = (sequence) => {
    // clearSelectedSquare also clears drop selection
    clearSelectedSquare();
    selectedSquareSequence = Number(sequence);
    if (Number.isNaN(selectedSquareSequence)) {
      selectedSquareSequence = null;
      return;
    }
    const selectedSquare = boardElement.querySelector(
      `.chess_board_square[data-sequence="${selectedSquareSequence}"]`
    );
    const selectedPiece = getPieceOnSquare(selectedSquare);
    if (selectedPiece) {
      selectedPiece.classList.add(SELECTED_PIECE_CLASS);
      void loadLegalDestinationsForSelection(selectedSquareSequence);
      // Do NOT refresh FS suggestions on piece selection.
      // Suggestions are only updated after a real move (human or AI) or on New Game.
    }
  };

  const sequenceByFileRank = (fileNum, rankNum) =>
    (boardMaxRank - rankNum) * boardFiles + (fileNum - 1);

  // API kinds → xianqi_pic filenames (bear = elephant; unused: dragon_*, empress_*).
  const XIANQI_KIND_FILE = {
    king: "general",
    advisor: "advisor",
    elephant: "bear",
    knight: "horse",
    rook: "chariot",
    cannon: "cannon",
    pawn: "soldier",
  };

  // API kinds → shogi_pic/*.svg (filenames match kinds; black via CSS rotate).
  const SHOGI_KINDS = new Set([
    "pawn", "lance", "knight", "silver", "gold", "bishop", "rook", "king",
    "promoted_pawn", "promoted_lance", "promoted_knight", "promoted_silver",
    "horse", "dragon",
  ]);

  const imagePathFromPiece = (piece) => {
    const kind = String(piece?.kind || "").toLowerCase();
    const color = String(piece?.color || "").toLowerCase();
    if (!kind || !color) return "";
    if (boardGameType === "xianqi") {
      const file = XIANQI_KIND_FILE[kind];
      if (!file) return "";
      const side = color === "black" ? "black" : "white";
      return `/pic/xianqi_pic/${file}_${side}.png`;
    }
    if (boardGameType === "shogi") {
      if (!SHOGI_KINDS.has(kind)) return "";
      return `/pic/shogi_pic/${kind}.svg`;
    }
    const tone = color === "black" ? "dark" : "light";
    return `/pic/chess_pic/${kind}_${tone}.png`;
  };

  // Full board sync from backend state (handles en passant, castling, promotion)
  const renderBoardFromState = (state, typeHint) => {
    if (!Array.isArray(state)) return false;
    ensureBoardGeometry(typeHint || gameTypeSelect?.value || boardGameType);

    const boardSquares = boardElement
      ? boardElement.querySelectorAll(".chess_board_square[data-sequence]")
      : document.querySelectorAll(".chess_board_square[data-sequence]");
    boardSquares.forEach((square) => {
      square.querySelectorAll(".piece_img").forEach((el) => el.remove());
    });

    for (const piece of state) {
      if (!piece || !piece.file || !piece.rank) continue;
      const sequence = sequenceByFileRank(piece.file, piece.rank);
      const square = boardElement
        ? boardElement.querySelector(`.chess_board_square[data-sequence="${sequence}"]`)
        : document.querySelector(`.chess_board_square[data-sequence="${sequence}"]`);
      if (!square) continue;
      const imagePath = imagePathFromPiece(piece);
      if (!imagePath) continue;

      const img = document.createElement("img");
      img.className = "piece_img";
      img.src = imagePath;
      img.alt = `piece_${piece.file}_${piece.rank}`;
      img.setAttribute("draggable", "true");
      if (piece.color) img.setAttribute("data-color", String(piece.color).toLowerCase());
      if (piece.kind) img.setAttribute("data-kind", String(piece.kind).toLowerCase());
      square.appendChild(img);
    }

    return true;
  };

  // send the movement command to backend
  const submitCommand = async (commandText = "") => {
    if (isSubmitting) return false;
    if (simulationRequestInFlight || isSimulationPlayback) {
      setStatus("Simulation is in progress. Please wait for it to finish.", "error");
      return false;
    }
    if (gameOver) {
      setStatus("Game has ended. Refresh to start a new game.", "error");
      return false;
    }

    const command = String(commandText || input.value).trim();
    if (!command) {
      setStatus("Please enter a chess movement command.", "error");
      return false;
    }
    isSubmitting = true;
    try {
      if (!currentGameId) {
        setStatus("Missing game session. Start a new game first.", "error");
        return false;
      }
      const body = new URLSearchParams({ command });
      const response = await fetch(`/api/games/${encodeURIComponent(currentGameId)}/move`, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: body.toString(),
      });

      if (!response.ok) {
        const errorMessage = await readErrorMessage(response, "Invalid command format");
        setStatus(errorMessage || "Invalid command format", "error");
        input.focus();

        return false;
      }

      const result = await response.json();
      syncGameIdFromResult(result);
      if (!result?.from || !result?.to) {
        setStatus("Invalid move response from server", "error");
        input.focus();
        return false;
      }

      input.value = input.value.trim() === command ? "" : input.value;
      const usedStateRender = renderBoardFromState(result.state);
      if (!usedStateRender) {
        setStatus("Missing board state in server response.", "error");
        return false;
      }
      const historyArray = Array.isArray(result.history) ? result.history : [];
      const detailedArray = Array.isArray(result.historyDetailed) ? result.historyDetailed : [];
      renderMoveHistory(historyArray, detailedArray);
      renderCurrentTurn(result.currentTurn);
      renderCheckState(result.checkedSide || result?.game?.outcome?.checkedSide);
      renderGameOutcome(result.game);
      renderClocks(result.game);
      renderGameConfig(result.game);
      renderGameInfo(result.captured, result.analysis);
      clearSelectedSquare();
      if (result.analysis) {
        stopAnalysisPolling();
      } else {
        const targetMoveNumber = Math.max(historyArray.length, detailedArray.length);
        startAnalysisPolling(targetMoveNumber, result.captured);
      }
      void refreshSuggestedMoves();
      input.focus();
      return true;
    } catch (error) {
      setCatchStatus(error);
      input.focus();
      return false;
    } finally {
      isSubmitting = false;
    }
  };

  const submitBoardMove = async (fromSequence, toSequence) => {
    let command = moveCommandFromSequence(fromSequence, toSequence);
    if (!command) return false;
    if (boardGameType === "shogi") {
      // Policy: must-promote → auto "+"; optional zone → Promote / Do not promote.
      const { must, can } = shogiPromotionFlags(toSequence);
      if (must) {
        command += "+";
      } else if (can) {
        const choice = await requestPromotionChoice("shogi");
        if (!choice) return false; // cancelled
        if (choice === "+") command += "+";
      }
      return submitCommand(command);
    }
    if (requiresPromotion(toSequence)) {
      const promotionChoice = await requestPromotionChoice("chess");
      if (!promotionChoice) return false;
      command += promotionChoice;
    }
    return submitCommand(command);
  };

  const createSessionOnLoad = async () => {
    const body = setupConfigBody();

    try {
      setStatus("Creating game session...", "success");
      const response = await fetch("/api/games", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: body.toString(),
      });
      if (!response.ok) {
        const errorMessage = await readErrorMessage(response, "Failed to create game session.");
        setStatus(errorMessage, "error");
        return;
      }
      const result = await response.json();
      syncGameIdFromResult(result);
      renderGameConfig(result.game, { syncClockSetup: true });
      renderBoardFromState(result.state, result.game?.type);
      renderMoveHistory(result.history, result.historyDetailed);
      renderCurrentTurn(result.currentTurn);
      renderCheckState(result.checkedSide || result?.game?.outcome?.checkedSide);
      renderGameOutcome(result.game);
      renderClocks(result.game);
      cachedAnalysis = null;
      clearCoachNotesState();
      renderGameInfo(result.captured, result.analysis);
      stopAnalysisPolling();
      clearSelectedSquare();
      void refreshSuggestedMoves();
      input.disabled = false;
      button.disabled = false;
      if (flagButton) flagButton.disabled = false;
      gameOver = false;
      setStatus("Game session ready.", "success");
      input.focus();
    } catch (error) {
      setCatchStatus(error);
    }
  };

  const initMouseMoveControls = () => {
    boardElement.addEventListener("click", async (event) => {
      if (gameOver || isSubmitting || pendingPromotionResolve) return;
      const targetSquare = getSquareElement(event.target);
      if (!targetSquare) return;

      const targetSequence = getSquareSequence(targetSquare);
      if (Number.isNaN(targetSequence)) return;

      if (selectedDropKind) {
        const dropped = await submitShogiDrop(targetSequence);
        if (dropped) clearSelectedSquare();
        return;
      }

      const targetHasCurrentTurnPiece = isCurrentTurnPiece(targetSquare);

      if (selectedSquareSequence == null) {
        if (targetHasCurrentTurnPiece) {
          setSelectedSquare(targetSequence);
        }
        return;
      }

      if (targetSequence === selectedSquareSequence) {
        clearSelectedSquare();
        return;
      }

      if (targetHasCurrentTurnPiece) {
        // Transfer selection to another same-side piece.
        setSelectedSquare(targetSequence);
        return;
      }

      const moved = await submitBoardMove(selectedSquareSequence, targetSequence);
      if (moved) clearSelectedSquare();
    });

    boardElement.addEventListener("dragstart", (event) => {
      if (gameOver || isSubmitting || pendingPromotionResolve) {
        event.preventDefault();
        return;
      }
      const piece = event.target instanceof Element ? event.target.closest(".piece_img") : null;
      if (!piece) return;
      const sourceSquare = getSquareElement(piece);
      if (!sourceSquare || !isCurrentTurnPiece(sourceSquare)) {
        event.preventDefault();
        return;
      }
      const sourceSequence = getSquareSequence(sourceSquare);
      if (Number.isNaN(sourceSequence)) {
        event.preventDefault();
        return;
      }
      dragSourceSequence = sourceSequence;
      setSelectedSquare(sourceSequence);
      piece.classList.add("piece_img_dragging");
      event.dataTransfer?.setData("text/plain", String(sourceSequence));
      if (event.dataTransfer) event.dataTransfer.effectAllowed = "move";
      // Shogi black uses CSS rotate(180deg); native drag ghost often drops that transform.
      // Use a pixel-sized canvas — cloning the <img> into body makes % width explode.
      if (
        event.dataTransfer &&
        piece instanceof HTMLImageElement &&
        String(boardWrapper?.dataset?.gameType || "").toLowerCase() === "shogi" &&
        String(piece.getAttribute("data-color") || "").toLowerCase() === "black"
      ) {
        const rect = piece.getBoundingClientRect();
        const w = Math.max(1, Math.round(rect.width));
        const h = Math.max(1, Math.round(rect.height));
        const canvas = document.createElement("canvas");
        canvas.width = w;
        canvas.height = h;
        const ctx = canvas.getContext("2d");
        if (ctx) {
          ctx.translate(w / 2, h / 2);
          ctx.rotate(Math.PI);
          ctx.drawImage(piece, -w / 2, -h / 2, w, h);
          event.dataTransfer.setDragImage(canvas, w / 2, h / 2);
        }
      }
    });

    boardElement.addEventListener("dragover", (event) => {
      const targetSquare = getSquareElement(event.target);
      if (!targetSquare) return;
      event.preventDefault();
      if (event.dataTransfer) event.dataTransfer.dropEffect = "move";
    });

    boardElement.addEventListener("drop", async (event) => {
      if (gameOver || isSubmitting || pendingPromotionResolve) return;
      const targetSquare = getSquareElement(event.target);
      if (!targetSquare) return;
      event.preventDefault();

      let sourceSequence = dragSourceSequence;
      if (sourceSequence == null) {
        const payload = Number(event.dataTransfer?.getData("text/plain"));
        if (!Number.isNaN(payload)) sourceSequence = payload;
      }
      const targetSequence = getSquareSequence(targetSquare);
      if (sourceSequence == null || Number.isNaN(targetSequence) || sourceSequence === targetSequence) {
        return;
      }

      const moved = await submitBoardMove(sourceSequence, targetSequence);
      if (moved) clearSelectedSquare();
    });

    boardElement.addEventListener("dragend", (event) => {
      const piece = event.target instanceof Element ? event.target.closest(".piece_img") : null;
      if (piece) piece.classList.remove("piece_img_dragging");
      dragSourceSequence = null;
    });
  };

  const initPromotionPicker = () => {
    if (!promotionPicker) return;
    closePromotionPicker();
    // Delegate so chess/shogi button sets can be swapped per open.
    promotionPicker.addEventListener("click", (event) => {
      const buttonEl =
        event.target instanceof Element
          ? event.target.closest(".promotion_choice_btn[data-promotion]")
          : null;
      if (buttonEl) {
        const choice = String(buttonEl.getAttribute("data-promotion") || "");
        if (!choice) return;
        resolvePromotionChoice(choice);
        return;
      }
      if (event.target === promotionPicker && pendingPromotionResolve) {
        resolvePromotionChoice("");
      }
    });
    document.addEventListener("keydown", (event) => {
      if (event.key === "Escape" && pendingPromotionResolve) {
        resolvePromotionChoice("");
      }
    });
  };

  const setupConfigBody = () => {
    const mode = String(gameModeSelect?.value || "human_vs_human");
    const fen = String(fenInput?.value || "").trim();
    const aiCount = fen ? "1" : String(aiGameCountInput?.value || "1");
    const params = new URLSearchParams({
      type: String(gameTypeSelect?.value || "chess"),
      mode,
      humanColor: String(humanSideSelect?.value || "white"),
      aiGameCount: aiCount,
      aiProfile: String(aiStrengthSelect?.value || "intermediate"),
      skillLevel: String(coachLevelSelect?.value || "intermediate"),
      fen,
    });
    return appendClockFields(params);
  };

  // Quiet config POST so the next /explain sees AI strength / coach level without Apply.
  const syncSetupToSession = async () => {
    if (!currentGameId || simulationRequestInFlight || isSimulationPlayback) return;
    try {
      const response = await fetch(`/api/games/${encodeURIComponent(currentGameId)}/config`, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: setupConfigBody().toString(),
      });
      if (!response.ok) return;
      const result = await response.json();
      if (result?.game) renderGameConfig(result.game);
    } catch (_) {
      // non-blocking
    }
  };

  button.addEventListener("click", submitCommand);
  if (gameModeSelect) gameModeSelect.addEventListener("change", updateSetupControlState);
  if (fenInput) fenInput.addEventListener("input", updateSetupControlState);
  if (clockEnabledInput) {
    clockEnabledInput.addEventListener("change", updateSetupControlState);
  }
  if (clockPresetSelect) {
    clockPresetSelect.addEventListener("change", () => {
      applyClockPresetToInputs();
    });
  }
  const markClockPresetCustom = () => {
    if (clockPresetSelect) clockPresetSelect.value = "custom";
  };
  if (clockBaseSecInput) clockBaseSecInput.addEventListener("input", markClockPresetCustom);
  if (clockIncrementSecInput) clockIncrementSecInput.addEventListener("input", markClockPresetCustom);
  if (clockHumanBaseSecInput) clockHumanBaseSecInput.addEventListener("input", markClockPresetCustom);
  if (clockAiBaseSecInput) clockAiBaseSecInput.addEventListener("input", markClockPresetCustom);
  if (aiStrengthSelect) {
    aiStrengthSelect.addEventListener("change", () => {
      updateSetupControlState();
      void syncSetupToSession();
    });
  }
  if (coachLevelSelect) {
    coachLevelSelect.addEventListener("change", () => {
      void syncSetupToSession();
    });
  }
  if (gameTypeSelect) {
    gameTypeSelect.addEventListener("change", () => {
      previewBoardForGameType(gameTypeSelect.value);
      updateSetupControlState();
    });
  }

  // --- Top-3 move hints (Shift + hover) ---
  let hintsVisible = false;
  const showTopMoves = async () => {
    if (!currentGameId) return;
    try {
      const res = await fetch(`/api/games/${encodeURIComponent(currentGameId)}/top-moves?k=3`);
      if (!res.ok) return;
      const data = await res.json();
      if (data.suggestions && data.suggestions.length > 0) {
        // Simple visual: log + optional future arrow rendering
        console.log("[Top moves]", data.suggestions);
        // You can extend this to draw arrows on the board
      }
    } catch (_) {}
  };

  // Show hints when Shift is held
  document.addEventListener("keydown", (e) => {
    if (e.key === "Shift" && !hintsVisible) {
      hintsVisible = true;
      showTopMoves();
    }
  });
  document.addEventListener("keyup", (e) => {
    if (e.key === "Shift") hintsVisible = false;
  });

  if (configApplyButton) {
    configApplyButton.addEventListener("click", async () => {
      if (simulationRequestInFlight || isSimulationPlayback) {
        setStatus("Simulation is in progress. Please wait for it to finish.", "error");
        return;
      }
      try {
        const mode = String(gameModeSelect?.value || "human_vs_human");
        const fen = String(fenInput?.value || "").trim();
        const aiCount = fen ? "1" : String(aiGameCountInput?.value || "1");
        if (!currentGameId) {
          setStatus("Missing game session. Start a new game first.", "error");
          return;
        }
        const body = setupConfigBody();
        const response = await fetch(`/api/games/${encodeURIComponent(currentGameId)}/config`, {
          method: "POST",
          headers: { "Content-Type": "application/x-www-form-urlencoded" },
          body: body.toString(),
        });
        if (!response.ok) {
          const errorMessage = await readErrorMessage(response, "Failed to apply game setup.");
          setStatus(errorMessage || "Failed to apply game setup.", "error");
          return;
        }
        const result = await response.json();
        syncGameIdFromResult(result);
        renderGameConfig(result.game, { syncClockSetup: true });
        renderClocks(result.game);
        previewBoardForGameType(result.game?.type || gameTypeSelect?.value);

        // Immediately store the human color from the applied config
        if (result.game?.config?.humanColor) {
          humanColor = String(result.game.config.humanColor).toLowerCase();
        }

        setStatus("Game setup applied. Click New Game to start.", "success");
      } catch (error) {
        setCatchStatus(error);
      }
    });
  }

  if (aiGameCountInput && configApplyButton) {
    simRunBtn = document.createElement("button");
    simRunBtn.id = "run_simulation_btn";
    simRunBtn.type = "button";
    simRunBtn.textContent = "Run AI Simulation";
    simRunBtn.className = "run-simulation-btn";

    // Insert right after the Apply Setup button for better alignment
    configApplyButton.insertAdjacentElement("afterend", simRunBtn);
    updateSetupControlState();

    simRunBtn.addEventListener("click", async () => {
      const parsed = readSimulationCount();
      if (!parsed.ok) {
        setStatus(parsed.message, "error");
        return;
      }
      if (simRunBtn.disabled) return;

      const n = parsed.count;
      const profile = String(aiStrengthSelect?.value || "intermediate");
      clearSelectedSquare();
      highlightSuggestedMoves([]);
      setNotesText("Simulation running...");
      setStatus("Running AI simulation...", "success");

      simRunBtn.disabled = true;
      isSimulationPlayback = true;
      simulationRequestInFlight = true;
      simulationData = null;
      currentSimGameIdx = -1;
      currentSimMoveIdx = 0;
      clearSimulationSummary();
      updateSetupControlState();

      try {
        const resp = await fetch("/api/simulate?details=true", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            games: n,
            profile,
            game: String(gameTypeSelect?.value || boardGameType || "chess"),
          }),
        });
        simulationRequestInFlight = false;
        updateSetupControlState();

        if (!resp.ok) {
          const errorMessage = await readErrorMessage(resp, "Simulation request failed.");
          if (resp.status === 409) {
            setStatus(`Simulation already running on server. ${errorMessage}`, "error");
          } else {
            setStatus(`Simulation failed: ${errorMessage}`, "error");
          }
          cleanupSimulationControls();
          return;
        }

        const payload = await resp.json();
        if (!Array.isArray(payload?.results)) {
          setStatus("Simulation failed: missing results payload.", "error");
          cleanupSimulationControls();
          return;
        }

        simulationData = payload;
        renderSimulationSummary(simulationData);
        if (simRunBtn) simRunBtn.style.display = "none";
        ensureSimulationControls();
        startNextSimulationGame();
        setStatus(`Simulation loaded (${n} game${n > 1 ? "s" : ""}).`, "success");
      } catch (error) {
        simulationRequestInFlight = false;
        updateSetupControlState();
        setCatchStatus(error, "Network error while loading simulation.");
        cleanupSimulationControls();
      }
    });
  }

  if (flagButton) {
    flagButton.addEventListener("click", async () => {
      if (simulationRequestInFlight || isSimulationPlayback) {
        setStatus("Simulation is in progress. Please wait for it to finish.", "error");
        return;
      }
      if (gameOver) {
        setStatus("Game has ended. Start a new game.", "error");
        return;
      }
      try {
        if (!currentGameId) {
          setStatus("Missing game session. Start a new game first.", "error");
          return;
        }
        const response = await fetch(`/api/games/${encodeURIComponent(currentGameId)}/flag`, {
          method: "POST",
        });
        if (!response.ok) {
          const errorMessage = await readErrorMessage(response, "Failed to flag game.");
          setStatus(errorMessage || "Failed to flag game.", "error");
          return;
        }
        const result = await response.json();
        syncGameIdFromResult(result);
        renderMoveHistory(result.history, result.historyDetailed);
        renderCurrentTurn(result.currentTurn);
        renderCheckState(result.checkedSide || result?.game?.outcome?.checkedSide);
        renderGameOutcome(result.game);
        renderClocks(result.game);
        renderGameConfig(result.game);

        // Store the human color from the game config for Human vs AI mode
        if (result.game?.config?.humanColor) {
          humanColor = String(result.game.config.humanColor).toLowerCase();
        }

        cachedAnalysis = null;
        renderGameInfo(result.captured, null);
        stopAnalysisPolling();
        resolvePromotionChoice("");
        clearSelectedSquare();
        if (gameOver) {
          showGameEndedNotes(result?.game?.outcome?.message || "Game has ended (flag / resign).");
        }
      } catch (error) {
        setCatchStatus(error);
      }
    });
  }
  if (newGameButton) {
    newGameButton.addEventListener("click", async () => {
      if (simulationRequestInFlight || isSimulationPlayback) {
        setStatus("Simulation is in progress. Please wait for it to finish.", "error");
        return;
      }
      try {
        if (!currentGameId) {
          setStatus("Missing game session. Start a new game first.", "error");
          return;
        }
        // Send current dropdown values so the new game respects type/mode/side/profile/clock
        const body = setupConfigBody();

        const response = await fetch(`/api/games/${encodeURIComponent(currentGameId)}/new`, {
          method: "POST",
          headers: { "Content-Type": "application/x-www-form-urlencoded" },
          body: body.toString(),
        });
        if (!response.ok) {
          const errorMessage = await readErrorMessage(response, "Failed to start a new game.");
          setStatus(errorMessage || "Failed to start a new game.", "error");
          return;
        }
        const result = await response.json();
        syncGameIdFromResult(result);
        // Config first so geometry / data-game-type match server type before placing pieces.
        renderGameConfig(result.game, { syncClockSetup: true });
        renderBoardFromState(result.state, result.game?.type);
        renderMoveHistory(result.history, result.historyDetailed);
        renderCurrentTurn(result.currentTurn);
        renderCheckState(result.checkedSide || result?.game?.outcome?.checkedSide);
        renderGameOutcome(result.game);
        renderClocks(result.game);

        // Store the human color from the game config for Human vs AI mode
        if (result.game?.config?.humanColor) {
          humanColor = String(result.game.config.humanColor).toLowerCase();
        }

        cachedAnalysis = null;
        cachedCapturedSummary = null;
        clearCoachNotesState();

        renderGameInfo(result.captured, null);
        stopAnalysisPolling();

        // Force reset win probability to 50/50 AFTER renderGameInfo, 
        // in case any async update tries to restore old analysis values.
        if (winProbWhiteValue) winProbWhiteValue.textContent = "50.0%";
        if (winProbBlackValue) winProbBlackValue.textContent = "50.0%";
        if (winProbWhiteBar) winProbWhiteBar.style.width = "50%";
        if (winProbBlackBar) winProbBlackBar.style.width = "50%";

        input.value = "";
        input.disabled = false;
        button.disabled = false;
        if (flagButton) flagButton.disabled = false;
        gameOver = false;
        resolvePromotionChoice("");
        clearSelectedSquare();
        cleanupSimulationControls();
        setStatus("New game started.", "success");
        input.focus();
      } catch (error) {
        setCatchStatus(error);
      }
    });
  }

  // Review playback: full UCI list + current ply; Back/Forward reload a prefix via load-moves.
  let reviewPlaybackMoves = null;
  let reviewPlaybackPly = 0;
  let reviewPlaybackBusy = false;

  const uciListFromSnapshot = (result) => {
    const detailed = Array.isArray(result?.historyDetailed) ? result.historyDetailed : [];
    const fromDetailed = detailed
      .map((entry) => String(entry?.command || "").trim().toLowerCase())
      .filter(Boolean);
    if (fromDetailed.length) return fromDetailed;
    const history = Array.isArray(result?.history) ? result.history : [];
    return history
      .map((line) => {
        const text = String(line || "");
        const idx = text.indexOf(":");
        return (idx >= 0 ? text.slice(idx + 1) : text).trim().toLowerCase();
      })
      .filter(Boolean);
  };

  const updateReviewPlaybackControls = () => {
    const total = Array.isArray(reviewPlaybackMoves) ? reviewPlaybackMoves.length : 0;
    const ply = total ? reviewPlaybackPly : 0;
    if (reviewMovesPlyLabel) {
      reviewMovesPlyLabel.textContent = total ? `Ply ${ply} / ${total}` : "Ply 0 / 0";
    }
    if (reviewMovesPrev) {
      reviewMovesPrev.disabled = !total || ply <= 0 || reviewPlaybackBusy;
    }
    if (reviewMovesNext) {
      reviewMovesNext.disabled = !total || ply >= total || reviewPlaybackBusy;
    }
  };

  const applyLoadedGameSnapshot = (result) => {
    syncGameIdFromResult(result);
    renderGameConfig(result.game, { syncClockSetup: true });
    renderBoardFromState(result.state, result.game?.type);
    renderMoveHistory(result.history, result.historyDetailed);
    renderCurrentTurn(result.currentTurn);
    renderCheckState(result.checkedSide || result?.game?.outcome?.checkedSide);
    renderGameOutcome(result.game);
    renderClocks(result.game);
    if (result.game?.config?.humanColor) {
      humanColor = String(result.game.config.humanColor).toLowerCase();
    }
    cachedAnalysis = null;
    cachedCapturedSummary = null;
    stopAnalysisPolling();
    input.value = "";
    resolvePromotionChoice("");
    clearSelectedSquare();
    cleanupSimulationControls();
    renderGameInfo(result.captured, null);
    updateReviewPlaybackControls();

    if (gameOver) {
      // renderGameOutcome already set end notes / disabled inputs.
      return;
    }

    clearCoachNotesState();
    const historyArray = Array.isArray(result.history) ? result.history : [];
    const detailedArray = Array.isArray(result.historyDetailed) ? result.historyDetailed : [];
    const targetMoveNumber = Math.max(historyArray.length, detailedArray.length);
    if (targetMoveNumber > 0) {
      lastExplanationText = "[coach] Thinking…";
      refreshNotesBox();
      void refreshSuggestedMoves();
      startAnalysisPolling(targetMoveNumber, result.captured);
    } else {
      clearCoachNotesState();
      refreshNotesBox();
      void refreshSuggestedMoves();
    }
  };

  const postLoadMovesRaw = async (raw) => {
    const response = await fetch(
      `/api/games/${encodeURIComponent(currentGameId)}/load-moves`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ raw }),
      }
    );
    if (!response.ok) {
      const errorMessage = await readErrorMessage(response, "Failed to load moves.");
      throw new Error(errorMessage || "Failed to load moves.");
    }
    return response.json();
  };

  const seekReviewPlayback = async (targetPly) => {
    if (!Array.isArray(reviewPlaybackMoves) || !reviewPlaybackMoves.length) {
      setStatus("Load moves first to enable playback.", "error");
      return;
    }
    if (!currentGameId) {
      setStatus("Missing game session. Start a new game first.", "error");
      return;
    }
    const total = reviewPlaybackMoves.length;
    const ply = Math.max(0, Math.min(total, Number(targetPly) || 0));
    if (reviewPlaybackBusy) return;
    reviewPlaybackBusy = true;
    updateReviewPlaybackControls();
    try {
      const raw = ply <= 0 ? "" : reviewPlaybackMoves.slice(0, ply).join(" ");
      setStatus(ply <= 0 ? "Review: start position…" : `Review: ply ${ply} / ${total}…`, "success");
      const result = await postLoadMovesRaw(raw);
      reviewPlaybackPly = ply;
      applyLoadedGameSnapshot(result);
      setStatus(ply <= 0 ? "Review at start position." : `Review at ply ${ply} / ${total}.`, "success");
    } catch (error) {
      setStatus(error?.message || "Review seek failed.", "error");
    } finally {
      reviewPlaybackBusy = false;
      updateReviewPlaybackControls();
    }
  };

  if (reviewMovesFile && reviewMovesInput) {
    reviewMovesFile.addEventListener("change", async () => {
      const file = reviewMovesFile.files && reviewMovesFile.files[0];
      if (!file) return;
      try {
        reviewMovesInput.value = await file.text();
        setStatus(`Loaded file ${file.name} into review box.`, "success");
      } catch (error) {
        setCatchStatus(error);
      }
    });
  }

  if (reviewMovesLoad) {
    reviewMovesLoad.addEventListener("click", async () => {
      if (simulationRequestInFlight || isSimulationPlayback) {
        setStatus("Simulation is in progress. Please wait for it to finish.", "error");
        return;
      }
      const raw = String(reviewMovesInput?.value || "").trim();
      if (!raw) {
        setStatus("Paste UCI moves or a game JSON first.", "error");
        return;
      }
      if (!currentGameId) {
        setStatus("Missing game session. Start a new game first.", "error");
        return;
      }
      if (reviewPlaybackBusy) return;
      reviewPlaybackBusy = true;
      updateReviewPlaybackControls();
      try {
        setStatus("Loading moves…", "success");
        const result = await postLoadMovesRaw(raw);
        const moves = uciListFromSnapshot(result);
        if (!moves.length) {
          setStatus("Load succeeded but no moves were found in the response.", "error");
          return;
        }
        reviewPlaybackMoves = moves;
        reviewPlaybackPly = moves.length;
        applyLoadedGameSnapshot(result);
        setStatus(`Loaded ${moves.length} move(s) for review. Use Back / Forward to step.`, "success");
        input.focus();
      } catch (error) {
        setStatus(error?.message || "Failed to load moves.", "error");
      } finally {
        reviewPlaybackBusy = false;
        updateReviewPlaybackControls();
      }
    });
  }

  if (reviewMovesPrev) {
    reviewMovesPrev.addEventListener("click", () => {
      if (simulationRequestInFlight || isSimulationPlayback) {
        setStatus("Simulation is in progress. Please wait for it to finish.", "error");
        return;
      }
      void seekReviewPlayback(reviewPlaybackPly - 1);
    });
  }

  if (reviewMovesNext) {
    reviewMovesNext.addEventListener("click", () => {
      if (simulationRequestInFlight || isSimulationPlayback) {
        setStatus("Simulation is in progress. Please wait for it to finish.", "error");
        return;
      }
      void seekReviewPlayback(reviewPlaybackPly + 1);
    });
  }

  updateReviewPlaybackControls();

  // --- Simulation Manual Playback Helpers ---
  function initialChessState() {
    const order = ["rook", "knight", "bishop", "queen", "king", "bishop", "knight", "rook"];
    const state = [];
    for (let file = 1; file <= 8; file++) {
      state.push({ file, rank: 1, kind: order[file - 1], color: "white" });
      state.push({ file, rank: 2, kind: "pawn", color: "white" });
      state.push({ file, rank: 7, kind: "pawn", color: "black" });
      state.push({ file, rank: 8, kind: order[file - 1], color: "black" });
    }
    return state;
  }

  function resetSimulationHistoryPanels() {
    if (moveHistoryWhiteList) moveHistoryWhiteList.innerHTML = '<li class="chess_move_history_placeholder">No moves yet.</li>';
    if (moveHistoryBlackList) moveHistoryBlackList.innerHTML = '<li class="chess_move_history_placeholder">No moves yet.</li>';
  }

  function initialXiangqiState() {
    const state = [];
    const back = ["rook", "knight", "elephant", "advisor", "king", "advisor", "elephant", "knight", "rook"];
    for (let file = 1; file <= 9; file++) {
      state.push({ file, rank: 1, kind: back[file - 1], color: "white" });
      state.push({ file, rank: 10, kind: back[file - 1], color: "black" });
    }
    for (const file of [2, 8]) {
      state.push({ file, rank: 3, kind: "cannon", color: "white" });
      state.push({ file, rank: 8, kind: "cannon", color: "black" });
    }
    for (const file of [1, 3, 5, 7, 9]) {
      state.push({ file, rank: 4, kind: "pawn", color: "white" });
      state.push({ file, rank: 7, kind: "pawn", color: "black" });
    }
    return state;
  }

  // Matches DefaultShogiStartFEN board (empty hands).
  function initialShogiState() {
    const state = [];
    const back = ["lance", "knight", "silver", "gold", "king", "gold", "silver", "knight", "lance"];
    for (let file = 1; file <= 9; file++) {
      state.push({ file, rank: 1, kind: back[file - 1], color: "white" });
      state.push({ file, rank: 9, kind: back[file - 1], color: "black" });
      state.push({ file, rank: 3, kind: "pawn", color: "white" });
      state.push({ file, rank: 7, kind: "pawn", color: "black" });
    }
    state.push({ file: 2, rank: 2, kind: "bishop", color: "white" });
    state.push({ file: 8, rank: 2, kind: "rook", color: "white" });
    state.push({ file: 2, rank: 8, kind: "rook", color: "black" });
    state.push({ file: 8, rank: 8, kind: "bishop", color: "black" });
    return state;
  }

  function resetBoardToInitialState() {
    previewBoardForGameType(gameTypeSelect?.value || boardGameType);
  }

  function clearResultLabelClasses() {
    if (resultWhiteValue) resultWhiteValue.classList.remove("game_info_result_win", "game_info_result_loss", "game_info_result_draw");
    if (resultBlackValue) resultBlackValue.classList.remove("game_info_result_win", "game_info_result_loss", "game_info_result_draw");
  }

  function setPlayingResultLabels() {
    clearResultLabelClasses();
    if (resultWhiteValue) resultWhiteValue.textContent = "Result: PLAYING";
    if (resultBlackValue) resultBlackValue.textContent = "Result: PLAYING";
  }

  function applySimulationResultLabels(gameResult) {
    clearResultLabelClasses();
    const resultText = String(gameResult?.result || "").toLowerCase();
    if (resultText === "white_win") {
      if (resultWhiteValue) {
        resultWhiteValue.textContent = "Result: WIN";
        resultWhiteValue.classList.add("game_info_result_win");
      }
      if (resultBlackValue) {
        resultBlackValue.textContent = "Result: LOSS";
        resultBlackValue.classList.add("game_info_result_loss");
      }
      return;
    }
    if (resultText === "black_win") {
      if (resultWhiteValue) {
        resultWhiteValue.textContent = "Result: LOSS";
        resultWhiteValue.classList.add("game_info_result_loss");
      }
      if (resultBlackValue) {
        resultBlackValue.textContent = "Result: WIN";
        resultBlackValue.classList.add("game_info_result_win");
      }
      return;
    }
    if (resultWhiteValue) {
      resultWhiteValue.textContent = "Result: DRAW";
      resultWhiteValue.classList.add("game_info_result_draw");
    }
    if (resultBlackValue) {
      resultBlackValue.textContent = "Result: DRAW";
      resultBlackValue.classList.add("game_info_result_draw");
    }
  }

  function ensureSimulationControls() {
    if (!configApplyButton || !configApplyButton.parentNode) return;

    if (!simNextMoveBtn) {
      simNextMoveBtn = document.createElement("button");
      simNextMoveBtn.id = "sim_next_move_btn";
      simNextMoveBtn.type = "button";
      simNextMoveBtn.textContent = "Next Move";
      simNextMoveBtn.className = "run-simulation-btn";
      simNextMoveBtn.addEventListener("click", playNextSimulationMove);
      configApplyButton.parentNode.appendChild(simNextMoveBtn);
    }

    if (!simNextGameBtn) {
      simNextGameBtn = document.createElement("button");
      simNextGameBtn.id = "sim_next_game_btn";
      simNextGameBtn.type = "button";
      simNextGameBtn.textContent = "Next Game";
      simNextGameBtn.className = "run-simulation-btn";
      simNextGameBtn.style.display = "none";
      simNextGameBtn.addEventListener("click", startNextSimulationGame);
      configApplyButton.parentNode.appendChild(simNextGameBtn);
    }
  }

  function finishCurrentSimulationGame() {
    const gameResult = simulationData?.results?.[currentSimGameIdx] || null;
    if (!gameResult) return;
    applySimulationResultLabels(gameResult);
    if (simNextMoveBtn) simNextMoveBtn.style.display = "none";
    if (simNextGameBtn) simNextGameBtn.style.display = "inline-block";
    const totalGames = Array.isArray(simulationData?.results) ? simulationData.results.length : 0;
    const isLastGame = currentSimGameIdx >= totalGames - 1;
    if (isLastGame) {
      setStatus(`Game ${currentSimGameIdx + 1} finished. All simulation games completed.`, "success");
      cleanupSimulationControls();
    } else {
      setStatus(`Game ${currentSimGameIdx + 1} finished. Click Next Game.`, "success");
    }
  }

  function startNextSimulationGame() {
    if (!simulationData || !Array.isArray(simulationData.results)) return;

    currentSimGameIdx++;
    currentSimMoveIdx = 0;

    if (currentSimGameIdx >= simulationData.results.length) {
      setStatus("All simulation games completed.", "success");
      cleanupSimulationControls();
      return;
    }

    if (simNextGameBtn) {
      simNextGameBtn.textContent = "Next Game";
      simNextGameBtn.disabled = false;
      simNextGameBtn.style.display = "none";
    }
    if (simNextMoveBtn) simNextMoveBtn.style.display = "inline-block";

    resetBoardToInitialState();
    resetSimulationHistoryPanels();
    setPlayingResultLabels();
    highlightSuggestedMoves([]);

    const totalGames = simulationData.results.length;
    setNotesText(`Simulation playback: Game ${currentSimGameIdx + 1}/${totalGames}`);
    setStatus(`Game ${currentSimGameIdx + 1} ready. Click Next Move.`, "success");
  }

  function playNextSimulationMove() {
    const gameResult = simulationData?.results?.[currentSimGameIdx];
    if (!gameResult) return;
    const moves = Array.isArray(gameResult.history_detailed) ? gameResult.history_detailed : [];

    if (moves.length === 0) {
      setStatus(`Game ${currentSimGameIdx + 1} has no move history in response.`, "error");
      finishCurrentSimulationGame();
      return;
    }
    if (currentSimMoveIdx >= moves.length) {
      finishCurrentSimulationGame();
      return;
    }

    const moveEntry = moves[currentSimMoveIdx] || {};
    const uciMove = String(moveEntry.command || "").trim();
    if (uciMove) {
      applyUciMoveToBoard(uciMove);
      playMoveSound(Boolean(moveEntry.isCapture));
      const side = String(moveEntry.side || (currentSimMoveIdx % 2 === 0 ? "white" : "black")).toLowerCase();
      const listEl = side === "black" ? moveHistoryBlackList : moveHistoryWhiteList;
      if (listEl) {
        clearHistoryPlaceholder(listEl);
        appendHistoryMove(
          listEl,
          side,
          String(moveEntry.pieceKind || "pawn"),
          String(moveEntry.to || ""),
          destinationFromCommand(uciMove),
          Boolean(moveEntry.isCapture),
          String(moveEntry.capturedPieceKind || "")
        );
        listEl.scrollTop = listEl.scrollHeight;
      }
      const line = `#${currentSimMoveIdx + 1} ${uciMove}`;
      appendNotesLine(line);
    }
    currentSimMoveIdx++;
  }

  function cleanupSimulationControls() {
    if (simNextMoveBtn) {
      simNextMoveBtn.remove();
      simNextMoveBtn = null;
    }
    if (simNextGameBtn) {
      simNextGameBtn.remove();
      simNextGameBtn = null;
    }
    if (simRunBtn) {
      simRunBtn.textContent = "Run AI Simulation";
      simRunBtn.style.display = "inline-block";
      simRunBtn.disabled = false;
      if (!isAIVsAIModeSelected()) {
        simRunBtn.style.display = "none";
      }
    }
    currentSimGameIdx = 0;
    currentSimMoveIdx = 0;
    simulationRequestInFlight = false;
    isSimulationPlayback = false;
    updateSetupControlState();
  }

  if (simulationDownloadJsonBtn) {
    simulationDownloadJsonBtn.addEventListener("click", () => {
      const json = buildSimulationJSON();
      if (!json) {
        setStatus("Run a simulation first to download results.", "error");
        return;
      }
      downloadTextFile(simulationDownloadFilename("json"), "application/json", json);
    });
  }

  if (simulationDownloadCsvBtn) {
    simulationDownloadCsvBtn.addEventListener("click", () => {
      const csv = buildSimulationCSV();
      if (!csv) {
        setStatus("Run a simulation first to download results.", "error");
        return;
      }
      downloadTextFile(simulationDownloadFilename("csv"), "text/csv", csv);
    });
  }
  // --- End Simulation Helpers ---

  input.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      submitCommand();
    }
  });
  initPromotionPicker();
  initMouseMoveControls();
  if (typeof ResizeObserver !== "undefined" && boardElement) {
    const xqGutterRo = new ResizeObserver(() => syncXiangqiCoordGutters());
    xqGutterRo.observe(boardElement);
  }
  window.addEventListener("beforeunload", () => closeGameSocket(false));

  renderGameInfo(null, null);
  clearSimulationSummary();
  renderCheckState("");
  renderGameOutcome({ status: "in_progress", result: "in_progress" });
  renderClocks(null);
  applyClockPresetToInputs();
  renderGameConfig({
    type: "chess",
    mode: "human_vs_human",
    config: { humanColor: "white", aiGameCount: 1, startFen: "" },
  });
  updateSetupControlState();
  void createSessionOnLoad();

  // Apply AI move when the backend returns it together with the human move
  window.applyAIMoveFromResult = (result) => {
    if (!result || !result.aiMove) return false;
    const match = String(result.aiMove).match(/^([a-i])(\d{1,2})([a-i])(\d{1,2})/i);
    if (!match) return false;
    const fromFile = match[1].toLowerCase().charCodeAt(0) - 97 + 1;
    const fromRank = parseInt(match[2], 10);
    const toFile = match[3].toLowerCase().charCodeAt(0) - 97 + 1;
    const toRank = parseInt(match[4], 10);
    if (
      fromFile < 1 || fromFile > boardFiles || fromRank < 1 || fromRank > boardMaxRank ||
      toFile < 1 || toFile > boardFiles || toRank < 1 || toRank > boardMaxRank
    ) {
      return false;
    }
    applyMoveOnBoard(fromFile, fromRank, toFile, toRank);
    return true;
  };
})();

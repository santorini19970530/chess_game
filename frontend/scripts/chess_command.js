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
  const gameTypeSelect = document.getElementById("game_type");
  const gameModeSelect = document.getElementById("game_mode");
  const humanSideSelect = document.getElementById("human_side");
  const aiGameCountInput = document.getElementById("ai_game_count");
  const fenInput = document.getElementById("fen_input");
  const aiStrengthSelect = document.getElementById("ai_strength");
  const configApplyButton = document.getElementById("game_config_apply");
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
  let gameOver = false;
  let currentTurn = "white";
  let humanColor = "white";           // human's chosen color in Human vs AI mode
  let selectedSquareSequence = null;
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

  // Future: pass isCapture / isCheckmate from snapshot or move result
  // Example: playMoveSound(result.wasCapture, result.wasCheckmate);
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
      renderGameInfo(result.captured, result.analysis);
      clearSelectedSquare();
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
      // Update board immediately — the CSS transition on .piece_img (400ms) gives the slide animation
      void refreshGameSnapshotFromAPI(gameId);
      // Also refresh FS suggestions directly (in case snapshot path is delayed)
      void refreshSuggestedMoves();
      // Play sound at the same time the piece starts moving
      playMoveSound(false);
      return;
    }
    if (event === "turn_changed") {
      renderCurrentTurn(data?.current_turn);
      renderCheckState(data?.checked_side);
      void refreshSuggestedMoves();
      return;
    }
    if (event === "game_outcome") {
      renderGameOutcome({
        result: data?.result,
        outcome: data?.outcome || {},
      });
      return;
    }
    if (event === "analysis_status_update") {
      const statusText = String(data?.status || "").toLowerCase();
      if (statusText === "pending") {
        setNotesText("Analyzing...");
        return;
      }
      if (statusText === "ready" && data?.analysis) {
        renderGameInfo(pendingAnalysisCapturedSnapshot || cachedCapturedSummary, data.analysis);
        stopAnalysisPolling();
        return;
      }
      if (statusText === "error") {
        const safeMessage = String(data?.last_error || "").trim();
        if (safeMessage) setNotesText(safeMessage);
      }
    }
    if (event === "explanation_ready" || event === "explanationReady") {
      if (!gameInfoNotesBox) return;
      const expl = String(data?.explanation || data?.analysis_explanation || "").trim();
      if (!expl) return;
      const prefix = data?.source === "heuristic_fallback" ? "(heuristic) " : "";
      lastExplanationText = prefix + expl;
      const current = String(gameInfoNotesBox.value || "").trim();
      if (current && current !== "Analyzing...") {
        if (!current.includes(lastExplanationText)) {
          setNotesText(current + "\n\n" + lastExplanationText);
        }
      } else {
        setNotesText(lastExplanationText);
      }
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
      highlightSuggestedMoves([]);
      return;
    }

    if (statusValue === "stalemate") {
      setStatus("Draw by stalemate.", "success");
      input.disabled = true;
      button.disabled = true;
      if (flagButton) flagButton.disabled = true;
      gameOver = true;
      highlightSuggestedMoves([]);
      return;
    }
    if (statusValue.startsWith("draw_")) {
      setStatus(outcome?.message || "Game drawn.", "success");
      input.disabled = true;
      button.disabled = true;
      if (flagButton) flagButton.disabled = true;
      gameOver = true;
      return;
    }

    if (statusValue === "resigned") {
      setStatus(outcome?.message || "Game ended by flag.", "error");
      input.disabled = true;
      button.disabled = true;
      if (flagButton) flagButton.disabled = true;
      gameOver = true;
      highlightSuggestedMoves([]);
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

  const updateSetupControlState = () => {
    const mode = String(gameModeSelect?.value || "human_vs_human");
    const isAIVsAI = mode === "ai_vs_ai";
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
      filesEl.replaceChildren(
        ...Array.from({ length: boardFiles }, (_, i) => {
          const span = document.createElement("span");
          span.className = "board_label";
          span.textContent = String.fromCharCode("a".charCodeAt(0) + i);
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

  const rebuildBoardGrid = () => {
    if (!boardElement || !boardWrapper) return;
    boardWrapper.dataset.gameType = boardGameType;
    boardWrapper.style.setProperty("--board-files", String(boardFiles));
    boardWrapper.style.setProperty("--board-ranks", String(boardMaxRank));
    rebuildBoardLabels();
    if (boardGameType === "xianqi") rebuildXiangqiBoard();
    else rebuildSquareGridBoard();
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

  const renderGameConfig = (game) => {
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
    humanColor = String(cfg.humanColor || "white").toLowerCase();
    updateSetupControlState();
  };

  const CHESS_PIECE_ORDER = ["queen", "rook", "bishop", "knight", "pawn", "king"];
  const XIANQI_PIECE_ORDER = ["cannon", "rook", "knight", "elephant", "advisor", "pawn", "king"];
  const PIECE_SYMBOL = {
    queen: "♛",
    rook: "♜",
    bishop: "♝",
    knight: "♞",
    pawn: "♟",
    king: "♚",
    cannon: "砲",
    elephant: "相",
    advisor: "仕",
  };

  const capturedPieceOrder = () =>
    boardGameType === "xianqi" ? XIANQI_PIECE_ORDER : CHESS_PIECE_ORDER;

  const capturedMapToText = (captured) => {
    const parts = [];
    for (const kind of capturedPieceOrder()) {
      const count = captured[kind] || 0;
      if (count <= 0) continue;
      parts.push(`${PIECE_SYMBOL[kind] || kind}×${count}`);
    }
    return parts.length ? parts.join("  ") : "";
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

    if (capturedWhiteValue) capturedWhiteValue.textContent = capturedMapToText(whiteCaptured);
    if (capturedBlackValue) capturedBlackValue.textContent = capturedMapToText(blackCaptured);
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

    if (gameInfoNotesBox && effectiveAnalysis) {
      const threatSummary = String(effectiveAnalysis?.threat_summary || "").trim();
      let notesText = threatSummary || "No analysis summary yet.";
      if (lastExplanationText) {
        notesText += `\n\n${lastExplanationText}`;
      }
      setNotesText(notesText);
    }
  };

  const startAnalysisPolling = (targetMoveNumber, capturedSnapshot) => {
    stopAnalysisPolling();
    if (boardGameType === "xianqi" || boardGameType === "shogi") {
      setNotesText("Coach analysis is Chess-only for now. Play is unaffected.");
      return;
    }
    setNotesText("Analyzing...");
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

  const requiresPromotion = (toSequence) => {
    if (boardGameType !== "chess") return false;
    const target = fileRankFromSequence(toSequence);
    if (!target) return false;
    return selectedLegalMoves.some(
      (move) =>
        Number(move?.file) === target.file &&
        Number(move?.rank) === target.rank &&
        Boolean(move?.requiresPromotion)
    );
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

  const requestPromotionChoice = () =>
    new Promise((resolve) => {
      if (!promotionPicker) {
        resolve("q");
        return;
      }
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
    // Do not clear FS suggestion highlights here; they are independent of piece selection.
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
      if (Boolean(move?.requiresPromotion)) {
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

  const refreshSuggestedMoves = async (retry = true) => {
    if (isSimulationPlayback) return;
    if (boardGameType === "xianqi" || boardGameType === "shogi") {
      highlightSuggestedMoves([]);
      return;
    }
    if (!currentGameId || gameOver) {
      highlightSuggestedMoves([]);
      return;
    }
    try {
      const profile = String(aiStrengthSelect?.value || "intermediate");
      const url = `/api/games/${encodeURIComponent(currentGameId)}/top-moves?profile=${encodeURIComponent(profile)}&k=3`;
      const resp = await fetch(url);
      if (!resp.ok) {
        // Transient 503 (engine starting) or 404/500 — retry once after a short delay
        if (retry) {
          window.setTimeout(() => {
            void refreshSuggestedMoves(false);
          }, 1200);
        } else {
          highlightSuggestedMoves([]);
        }
        return;
      }
      const data = await resp.json();
      const suggestions = Array.isArray(data?.suggestions) ? data.suggestions : [];
      highlightSuggestedMoves(suggestions);
    } catch (_) {
      if (retry) {
        window.setTimeout(() => {
          void refreshSuggestedMoves(false);
        }, 1200);
      } else {
        highlightSuggestedMoves([]);
      }
    }
  };

  const highlightSuggestedMoves = (suggestions) => {
    // Clear previous suggestion highlights
    boardElement
      .querySelectorAll(`.${SUGGESTED_MOVE_CLASS}`)
      .forEach((square) => square.classList.remove(SUGGESTED_MOVE_CLASS));

    selectedSuggestedMoves = [];
    const top = Array.isArray(suggestions) ? suggestions.slice(0, 3) : [];

    if (!top.length) {
      if (gameInfoNotesBox) {
        // Only clear if we previously wrote FS suggestions here.
        // Preserve any LLM explanation that may still be relevant.
        if (gameInfoNotesBox.dataset.fsSuggestions === "1") {
          if (lastExplanationText) {
            setNotesText(lastExplanationText);
          } else {
            setNotesText("");
          }
          delete gameInfoNotesBox.dataset.fsSuggestions;
        }
      }
      return;
    }

    // Subtle cyan ring on destination squares (secondary visual cue)
    for (const sug of top) {
      const toSq = String(sug?.to || sug?.move?.slice(2, 4) || "").toLowerCase();
      if (!toSq || toSq.length !== 2) continue;
      const file = toSq.charCodeAt(0) - 96;
      const rank = Number(toSq[1]);
      if (Number.isNaN(file) || Number.isNaN(rank)) continue;
      const sequence = sequenceByFileRank(file, rank);
      const square = boardElement.querySelector(
        `.chess_board_square[data-sequence="${sequence}"]`
      );
      if (square) {
        square.classList.add(SUGGESTED_MOVE_CLASS);
        selectedSuggestedMoves.push({ sequence, move: sug.move || `${sug.from}${sug.to}` });
      }
    }

    // Write suggestions into the existing notes textarea (reusing the reserved box)
    if (gameInfoNotesBox) {
      // Note: full SAN notation requires a backend enhancement (current data is UCI).
      // UCI is kept for now because it is unambiguous and matches engine output.
      let text = "FS suggestions:\n";
      top.forEach((sug, idx) => {
        const mv = sug.move || "????";
        const sc = typeof sug.score_cp === "number" ? ` (${sug.score_cp > 0 ? "+" : ""}${sug.score_cp})` : "";
        text += `${idx + 1}. ${mv}${sc}\n`;
      });
      if (lastExplanationText) {
        text += "\n" + lastExplanationText;
      }
      setNotesText(text.trim());
      gameInfoNotesBox.dataset.fsSuggestions = "1";
    }
  };

  const loadSuggestedMovesForSelection = async (sequence) => {
    if (isSimulationPlayback) return; // Suppress during simulation
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
    } catch (_error) {
      setStatus("Network error. Please try again.", "error");
      input.focus();
      return false;
    } finally {
      isSubmitting = false;
    }
  };

  const submitBoardMove = async (fromSequence, toSequence) => {
    let command = moveCommandFromSequence(fromSequence, toSequence);
    if (!command) return false;
    if (requiresPromotion(toSequence)) {
      const promotionChoice = await requestPromotionChoice();
      if (!promotionChoice) return false;
      command += promotionChoice;
    }
    return submitCommand(command);
  };

  const createSessionOnLoad = async () => {
    const mode = String(gameModeSelect?.value || "human_vs_human");
    const fen = String(fenInput?.value || "").trim();
    const aiCount = fen ? "1" : String(aiGameCountInput?.value || "1");
    const body = new URLSearchParams({
      type: String(gameTypeSelect?.value || "chess"),
      mode,
      humanColor: String(humanSideSelect?.value || "white"),
      aiGameCount: aiCount,
      aiProfile: String(aiStrengthSelect?.value || "intermediate"),
      fen,
    });

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
      renderGameConfig(result.game);
      renderBoardFromState(result.state, result.game?.type);
      renderMoveHistory(result.history, result.historyDetailed);
      renderCurrentTurn(result.currentTurn);
      renderCheckState(result.checkedSide || result?.game?.outcome?.checkedSide);
      renderGameOutcome(result.game);
      cachedAnalysis = null;
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
    } catch (_) {
      setStatus("Network error. Please try again.", "error");
    }
  };

  const initMouseMoveControls = () => {
    boardElement.addEventListener("click", async (event) => {
      if (gameOver || isSubmitting || pendingPromotionResolve) return;
      const targetSquare = getSquareElement(event.target);
      if (!targetSquare) return;

      const targetSequence = getSquareSequence(targetSquare);
      if (Number.isNaN(targetSequence)) return;
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
    promotionPicker
      .querySelectorAll(".promotion_choice_btn[data-promotion]")
      .forEach((buttonEl) => {
        buttonEl.addEventListener("click", () => {
          const choice = String(buttonEl.getAttribute("data-promotion") || "").toLowerCase();
          if (!choice) return;
          resolvePromotionChoice(choice);
        });
      });
    promotionPicker.addEventListener("click", (event) => {
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

  button.addEventListener("click", submitCommand);
  if (gameModeSelect) gameModeSelect.addEventListener("change", updateSetupControlState);
  if (fenInput) fenInput.addEventListener("input", updateSetupControlState);
  if (aiStrengthSelect) aiStrengthSelect.addEventListener("change", updateSetupControlState);
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
        const body = new URLSearchParams({
          type: String(gameTypeSelect?.value || "chess"),
          mode,
          humanColor: String(humanSideSelect?.value || "white"),
          aiGameCount: aiCount,
          aiProfile: String(aiStrengthSelect?.value || "intermediate"),
          fen,
        });
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
        renderGameConfig(result.game);
        previewBoardForGameType(result.game?.type || gameTypeSelect?.value);

        // Immediately store the human color from the applied config
        if (result.game?.config?.humanColor) {
          humanColor = String(result.game.config.humanColor).toLowerCase();
        }

        setStatus("Game setup applied. Click New Game to start.", "success");
      } catch (_error) {
        setStatus("Network error. Please try again.", "error");
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
      } catch (_e) {
        simulationRequestInFlight = false;
        updateSetupControlState();
        setStatus("Network error while loading simulation.", "error");
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
        renderGameConfig(result.game);

        // Store the human color from the game config for Human vs AI mode
        if (result.game?.config?.humanColor) {
          humanColor = String(result.game.config.humanColor).toLowerCase();
        }

        cachedAnalysis = null;
        renderGameInfo(result.captured, result.analysis);
        stopAnalysisPolling();
        resolvePromotionChoice("");
        clearSelectedSquare();
      } catch (_error) {
        setStatus("Network error. Please try again.", "error");
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
        // Send current dropdown values so the new game respects type/mode/side/profile
        const mode = String(gameModeSelect?.value || "human_vs_human");
        const humanColor = String(humanSideSelect?.value || "white");
        const body = new URLSearchParams({
          type: String(gameTypeSelect?.value || "chess"),
          mode,
          humanColor,
          aiProfile: String(aiStrengthSelect?.value || "intermediate"),
        });

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
        renderGameConfig(result.game);
        renderBoardFromState(result.state, result.game?.type);
        renderMoveHistory(result.history, result.historyDetailed);
        renderCurrentTurn(result.currentTurn);
        renderCheckState(result.checkedSide || result?.game?.outcome?.checkedSide);
        renderGameOutcome(result.game);

        // Store the human color from the game config for Human vs AI mode
        if (result.game?.config?.humanColor) {
          humanColor = String(result.game.config.humanColor).toLowerCase();
        }

        cachedAnalysis = null;
        cachedCapturedSummary = null;

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
      } catch (_error) {
        setStatus("Network error. Please try again.", "error");
      }
    });
  }
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
  window.addEventListener("beforeunload", () => closeGameSocket(false));

  renderGameInfo(null, null);
  clearSimulationSummary();
  renderCheckState("");
  renderGameOutcome({ status: "in_progress", result: "in_progress" });
  renderGameConfig({
    type: "chess",
    mode: "human_vs_human",
    config: { humanColor: "white", aiGameCount: 1, startFen: "" },
  });
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

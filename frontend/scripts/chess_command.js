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
  const promotionPicker = document.getElementById("promotion_picker");
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
      renderBoardFromState(result.state);
      renderMoveHistory(result.history, result.historyDetailed);
      renderCurrentTurn(result.currentTurn);
      renderCheckState(result.checkedSide || result?.game?.outcome?.checkedSide);
      renderGameOutcome(result.game);
      renderGameConfig(result.game);
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
    if (!event || !gameId || gameId !== currentGameId) return;
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
        if (gameInfoNotesBox) gameInfoNotesBox.value = "Analyzing...";
        return;
      }
      if (statusText === "ready" && data?.analysis) {
        renderGameInfo(pendingAnalysisCapturedSnapshot || cachedCapturedSummary, data.analysis);
        stopAnalysisPolling();
        return;
      }
      if (statusText === "error") {
        const safeMessage = String(data?.last_error || "").trim();
        if (safeMessage && gameInfoNotesBox) gameInfoNotesBox.value = safeMessage;
      }
    }
    if (event === "explanation_ready" || event === "explanationReady") {
      if (!gameInfoNotesBox) return;
      const expl = String(data?.explanation || data?.analysis_explanation || "").trim();
      if (!expl) return;
      const prefix = data?.source === "heuristic_fallback" ? "(heuristic) " : "";
      lastExplanationText = prefix + expl;
      const current = gameInfoNotesBox.value.trim();
      if (current && current !== "Analyzing...") {
        if (!current.includes(lastExplanationText)) {
          gameInfoNotesBox.value = current + "\n\n" + lastExplanationText;
        }
      } else {
        gameInfoNotesBox.value = lastExplanationText;
      }
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
    const fenProvided = Boolean(String(fenInput?.value || "").trim());
    if (humanSideSelect) humanSideSelect.disabled = mode === "ai_vs_ai";
    if (aiGameCountInput) {
      aiGameCountInput.disabled = mode !== "ai_vs_ai";
      if (fenProvided) aiGameCountInput.value = "1";
    }
    if (aiStrengthSelect) {
      // Show strength selector only for modes that involve AI
      aiStrengthSelect.disabled = !(mode === "human_vs_ai" || mode === "ai_vs_ai");
    }
  };

  const renderGameConfig = (game) => {
    const cfg = game?.config;
    if (!cfg) return;
    if (gameTypeSelect) gameTypeSelect.value = String(game.type || "chess");
    if (gameModeSelect) gameModeSelect.value = String(game.mode || "human_vs_human");
    if (humanSideSelect) humanSideSelect.value = String(cfg.humanColor || "white");
    if (aiGameCountInput) aiGameCountInput.value = String(cfg.aiGameCount || 1);
    if (fenInput) fenInput.value = String(cfg.startFen || "");
    if (aiStrengthSelect) aiStrengthSelect.value = String(cfg.aiProfile || cfg.aiStrength || "intermediate");
    // Update the internal humanColor variable used for move validation
    humanColor = String(cfg.humanColor || "white").toLowerCase();
    updateSetupControlState();
  };

  const PIECE_ORDER = ["queen", "rook", "bishop", "knight", "pawn", "king"];
  const PIECE_SYMBOL = {
    queen: "♛",
    rook: "♜",
    bishop: "♝",
    knight: "♞",
    pawn: "♟",
    king: "♚",
  };

  const capturedMapToText = (captured) => {
    const parts = [];
    for (const kind of PIECE_ORDER) {
      const count = captured[kind] || 0;
      if (count <= 0) continue;
      parts.push(`${PIECE_SYMBOL[kind]}×${count}`);
    }
    return parts.length ? parts.join("  ") : "";
  };

  const normalizeCapturedSummary = (summary) => {
    if (!summary || typeof summary !== "object")
      return {
        white: { pawn: 0, rook: 0, knight: 0, bishop: 0, queen: 0, king: 0 },
        black: { pawn: 0, rook: 0, knight: 0, bishop: 0, queen: 0, king: 0 },
      };
    const normalized = {
      white: { pawn: 0, rook: 0, knight: 0, bishop: 0, queen: 0, king: 0 },
      black: { pawn: 0, rook: 0, knight: 0, bishop: 0, queen: 0, king: 0 },
    };
    for (const side of ["white", "black"]) {
      const source = summary[side];
      if (!source || typeof source !== "object") continue;
      for (const kind of PIECE_ORDER) {
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
      gameInfoNotesBox.value = notesText;
    }
  };

  const startAnalysisPolling = (targetMoveNumber, capturedSnapshot) => {
    stopAnalysisPolling();
    if (gameInfoNotesBox) gameInfoNotesBox.value = "Analyzing...";
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
      white: { pawn: "♙", rook: "♖", knight: "♘", bishop: "♗", queen: "♕", king: "♔" },
      black: { pawn: "♟", rook: "♜", knight: "♞", bishop: "♝", queen: "♛", king: "♚" },
    };
    return iconMap[color]?.[kind] || (color === "black" ? "♟" : "♙");
  };

  const opponentSide = (side) =>
    String(side || "").toLowerCase() === "black" ? "white" : "black";

  const destinationFromCommand = (command) => {
    const text = String(command || "").trim().toLowerCase();
    if (!text) return "";
    const match = text.match(/([a-h][1-8])[qrbn]?$/);
    return match ? match[1] : text;
  };

  const appendHistoryMove = (listEl, side, pieceKind, toSquare, fallbackText, isCapture, capturedPieceKind) => {
    const item = document.createElement("li");
    const iconSpan = document.createElement("span");
    iconSpan.className = "chess_move_history_piece_icon";
    iconSpan.textContent = movePieceIcon(side, pieceKind);
    const textSpan = document.createElement("span");
    textSpan.className = "chess_move_history_move_text";
    const moveText = toSquare || fallbackText || "";
    if (isCapture) {
      textSpan.textContent = `${moveText} x `;
      if (capturedPieceKind) {
        const capturedIcon = document.createElement("span");
        capturedIcon.className = "chess_move_history_piece_icon";
        capturedIcon.textContent = movePieceIcon(opponentSide(side), capturedPieceKind);
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

  const sequenceToSquare = (sequence) => {
    const seq = Number(sequence);
    if (Number.isNaN(seq) || seq < 0 || seq > 63) return "";
    const fileChar = String.fromCharCode("a".charCodeAt(0) + (seq % 8));
    const rankNum = 8 - Math.floor(seq / 8);
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
    if (Number.isNaN(seq) || seq < 0 || seq > 63) return NaN;
    return 8 - Math.floor(seq / 8);
  };

  const fileRankFromSequence = (sequence) => {
    const seq = Number(sequence);
    if (Number.isNaN(seq) || seq < 0 || seq > 63) return null;
    return {
      file: (seq % 8) + 1,
      rank: 8 - Math.floor(seq / 8),
    };
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
            gameInfoNotesBox.value = lastExplanationText;
          } else {
            gameInfoNotesBox.value = "";
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
      gameInfoNotesBox.value = text.trim();
      gameInfoNotesBox.dataset.fsSuggestions = "1";
    }
  };

  const loadSuggestedMovesForSelection = async (sequence) => {
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
    (8 - rankNum) * 8 + (fileNum - 1);

  const imagePathFromPiece = (piece) => {
    const kind = String(piece?.kind || "").toLowerCase();
    const color = String(piece?.color || "").toLowerCase();
    if (!kind || !color) return "";
    const tone = color === "black" ? "dark" : "light";
    return `/pic/chess_pic/${kind}_${tone}.png`;
  };

  // Full board sync from backend state (handles en passant, castling, promotion)
  const renderBoardFromState = (state) => {
    if (!Array.isArray(state)) return false;

    const boardSquares = document.querySelectorAll(".chess_board_square[data-sequence]");
    boardSquares.forEach((square) => {
      square.querySelectorAll(".piece_img").forEach((el) => el.remove());
    });

    for (const piece of state) {
      if (!piece || !piece.file || !piece.rank) continue;
      const sequence = sequenceByFileRank(piece.file, piece.rank);
      const square = document.querySelector(
        `.chess_board_square[data-sequence="${sequence}"]`
      );
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
      renderBoardFromState(result.state);
      renderMoveHistory(result.history, result.historyDetailed);
      renderCurrentTurn(result.currentTurn);
      renderCheckState(result.checkedSide || result?.game?.outcome?.checkedSide);
      renderGameOutcome(result.game);
      renderGameConfig(result.game);
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
  if (flagButton) {
    flagButton.addEventListener("click", async () => {
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
      try {
        if (!currentGameId) {
          setStatus("Missing game session. Start a new game first.", "error");
          return;
        }
        // Send current dropdown values so the new game respects the selected mode/side/profile
        const mode = String(gameModeSelect?.value || "human_vs_human");
        const humanColor = String(humanSideSelect?.value || "white");
        const body = new URLSearchParams({
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
        renderBoardFromState(result.state);
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
        input.value = "";
        input.disabled = false;
        button.disabled = false;
        if (flagButton) flagButton.disabled = false;
        gameOver = false;
        resolvePromotionChoice("");
        clearSelectedSquare();
        setStatus("New game started.", "success");
        input.focus();
      } catch (_error) {
        setStatus("Network error. Please try again.", "error");
      }
    });
  }
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
  renderCheckState("");
  renderGameOutcome({ status: "in_progress", result: "in_progress" });
  renderGameConfig({
    type: "chess",
    mode: "human_vs_human",
    config: { humanColor: "white", aiGameCount: 1, startFen: "" },
  });
  void createSessionOnLoad();

  // Helper for step 4 – apply AI move when the backend returns it together with the human move
  window.applyAIMoveFromResult = (result) => {
    if (!result || !result.aiMove) return false;
    const uci = String(result.aiMove).toLowerCase();
    if (uci.length < 4) return false;

    const fromFile = uci.charCodeAt(0) - 97 + 1;
    const fromRank = parseInt(uci[1], 10);
    const toFile = uci.charCodeAt(2) - 97 + 1;
    const toRank = parseInt(uci[3], 10);

    if (fromFile && fromRank && toFile && toRank) {
      applyMoveOnBoard(fromFile, fromRank, toFile, toRank);
      return true;
    }
    return false;
  };
})();

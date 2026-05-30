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
  const configApplyButton = document.getElementById("game_config_apply");
  const boardElement = document.querySelector(".chess_board");
  const promotionPicker = document.getElementById("promotion_picker");
  const moveSound = new Audio("/sounds/chess_movement.wav");
  const CHECK_CLASS = "game_info_col_in_check";
  const SELECTED_PIECE_CLASS = "piece_img_selected";
  const LEGAL_DESTINATION_CLASS = "chess_board_square_legal_destination";
  const LEGAL_PROMOTION_DESTINATION_CLASS = "chess_board_square_legal_promotion";
  const LEGAL_CAPTURE_DESTINATION_CLASS = "chess_board_square_legal_capture";
  let gameOver = false;
  let currentTurn = "white";
  let selectedSquareSequence = null;
  let dragSourceSequence = null;
  let legalMovesRequestVersion = 0;
  let isSubmitting = false;
  let pendingPromotionResolve = null;
  let analysisPollTimer = null;
  let cachedAnalysis = null;
  let currentGameId = "";

  if (!input || !button || !status || !moveHistoryWhiteList || !moveHistoryBlackList || !boardElement) return;

  const gameIdInput = document.getElementById("active_game_id");
  if (gameIdInput?.value) currentGameId = String(gameIdInput.value).trim();

  input.focus();

  // set current status
  const setStatus = (message, type) => {
    status.textContent = message;
    status.className = `command_status ${type}`;
  };

  const syncGameIdFromResult = (result) => {
    const nextId = String(result?.game?.id || "").trim();
    if (!nextId) return;
    currentGameId = nextId;
    if (gameIdInput) gameIdInput.value = nextId;
  };

  const stopAnalysisPolling = () => {
    if (analysisPollTimer != null) {
      window.clearInterval(analysisPollTimer);
      analysisPollTimer = null;
    }
  };

  const sideLabel = (side) => (String(side || "").toLowerCase() === "black" ? "Black" : "White");

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
      return;
    }

    if (statusValue === "stalemate") {
      setStatus("Draw by stalemate.", "success");
      input.disabled = true;
      button.disabled = true;
      if (flagButton) flagButton.disabled = true;
      gameOver = true;
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
  };

  const renderGameConfig = (game) => {
    const cfg = game?.config;
    if (!cfg) return;
    if (gameTypeSelect) gameTypeSelect.value = String(game.type || "chess");
    if (gameModeSelect) gameModeSelect.value = String(game.mode || "human_vs_human");
    if (humanSideSelect) humanSideSelect.value = String(cfg.humanColor || "white");
    if (aiGameCountInput) aiGameCountInput.value = String(cfg.aiGameCount || 1);
    if (fenInput) fenInput.value = String(cfg.startFen || "");
    updateSetupControlState();
  };

  const INITIAL_COUNTS = {
    white: { pawn: 8, rook: 2, knight: 2, bishop: 2, queen: 1, king: 1 },
    black: { pawn: 8, rook: 2, knight: 2, bishop: 2, queen: 1, king: 1 },
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

  const countPiecesByColor = (state) => {
    const counts = {
      white: { pawn: 0, rook: 0, knight: 0, bishop: 0, queen: 0, king: 0 },
      black: { pawn: 0, rook: 0, knight: 0, bishop: 0, queen: 0, king: 0 },
    };
    if (!Array.isArray(state)) return counts;
    for (const piece of state) {
      const side = String(piece?.color || "").toLowerCase();
      const kind = String(piece?.kind || "").toLowerCase();
      if (!counts[side] || !counts[side][kind]) continue;
      counts[side][kind] += 1;
    }
    return counts;
  };

  const capturedMap = (capturerColor, liveCounts) => {
    const opponent = capturerColor === "white" ? "black" : "white";
    const out = { pawn: 0, rook: 0, knight: 0, bishop: 0, queen: 0, king: 0 };
    for (const kind of Object.keys(out)) {
      out[kind] = Math.max(0, INITIAL_COUNTS[opponent][kind] - liveCounts[opponent][kind]);
    }
    return out;
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
    if (!summary || typeof summary !== "object") return null;
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

  const estimateWinProb = (liveCounts) => {
    const pieceValue = { pawn: 1, knight: 3, bishop: 3, rook: 5, queen: 9, king: 0 };
    const material = { white: 0, black: 0 };
    for (const side of ["white", "black"]) {
      for (const kind of Object.keys(pieceValue)) {
        material[side] += (liveCounts[side][kind] || 0) * pieceValue[kind];
      }
    }
    const total = material.white + material.black;
    if (total <= 0) return { white: 50, black: 50 };
    const whiteProb = Math.round((material.white / total) * 100);
    const blackProb = 100 - whiteProb;
    return { white: whiteProb, black: blackProb };
  };

  const extractBoardStateFromDOM = () => {
    const pieces = [];
    const squares = document.querySelectorAll(".chess_board_square[data-sequence]");
    for (const square of squares) {
      const pieceEl = square.querySelector(".piece_img");
      if (!pieceEl) continue;

      const sequence = Number(square.getAttribute("data-sequence"));
      if (Number.isNaN(sequence)) continue;
      const file = (sequence % 8) + 1;
      const rank = 8 - Math.floor(sequence / 8);

      let kind = String(pieceEl.getAttribute("data-kind") || "").toLowerCase();
      let color = String(pieceEl.getAttribute("data-color") || "").toLowerCase();

      // Fallback for old markup where data attributes may be missing.
      if (!kind || !color) {
        const src = pieceEl.getAttribute("src") || "";
        const match = src.match(/(pawn|rook|knight|bishop|queen|king)_(light|dark)\.png/i);
        if (!match) continue;
        kind = match[1].toLowerCase();
        color = match[2].toLowerCase() === "light" ? "white" : "black";
      }

      pieces.push({
        kind,
        color,
        file,
        rank,
      });
    }
    return pieces;
  };

  const renderGameInfo = (state, capturedSummary, analysis) => {
    const effectiveAnalysis = analysis || cachedAnalysis;
    if (analysis) cachedAnalysis = analysis;
    const liveCounts = countPiecesByColor(state);
    const normalizedCaptured = normalizeCapturedSummary(capturedSummary);
    const whiteCaptured = normalizedCaptured
      ? normalizedCaptured.white
      : capturedMap("white", liveCounts);
    const blackCaptured = normalizedCaptured
      ? normalizedCaptured.black
      : capturedMap("black", liveCounts);
    const winProb = estimateWinProb(liveCounts);
    const analyzerWhite = fromAnalyzerChance(effectiveAnalysis?.win_chance_white);
    const analyzerBlack = fromAnalyzerChance(effectiveAnalysis?.win_chance_black);
    const hasAnalyzerProb = analyzerWhite != null && analyzerBlack != null;
    const whiteProb = clampPercentage(hasAnalyzerProb ? analyzerWhite : winProb.white);
    const blackProb = clampPercentage(hasAnalyzerProb ? analyzerBlack : winProb.black);
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
      gameInfoNotesBox.value = threatSummary || "No analysis summary yet.";
    }
  };

  const startAnalysisPolling = (targetMoveNumber, stateSnapshot, capturedSnapshot) => {
    stopAnalysisPolling();
    if (gameInfoNotesBox) gameInfoNotesBox.value = "Analyzing...";
    const target = Number(targetMoveNumber) || 0;

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
        renderGameInfo(stateSnapshot, capturedSnapshot, latestAnalysis);
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

  // highlight the box being mentioned by user
  const squareSelectorByFileRank = (fileChar, rankChar) => {
    const fileIndex = fileChar.charCodeAt(0) - "a".charCodeAt(0) + 1;
    const rankNum = Number(rankChar);
    if (fileIndex < 1 || fileIndex > 8 || rankNum < 1 || rankNum > 8) return "";

    const sequence = (8 - rankNum) * 8 + (fileIndex - 1);
    return `.chess_board_square[data-sequence="${sequence}"]`;
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
    return pieceColor === currentTurn;
  };

  const requiresPromotion = (fromSequence, toSequence) => {
    const fromSquare = boardElement.querySelector(
      `.chess_board_square[data-sequence="${Number(fromSequence)}"]`
    );
    const piece = getPieceOnSquare(fromSquare);
    if (!piece) return false;
    const kind = String(piece.getAttribute("data-kind") || "").toLowerCase();
    if (kind !== "pawn") return false;
    const color = String(piece.getAttribute("data-color") || "").toLowerCase();
    const targetRank = rankFromSequence(toSequence);
    if (Number.isNaN(targetRank)) return false;
    return (color === "white" && targetRank === 8) || (color === "black" && targetRank === 1);
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
    for (const move of moves) {
      const fileNum = Number(move?.file);
      const rankNum = Number(move?.rank);
      if (Number.isNaN(fileNum) || Number.isNaN(rankNum)) continue;
      const sequence = sequenceByFileRank(fileNum, rankNum);
      const square = boardElement.querySelector(
        `.chess_board_square[data-sequence="${sequence}"]`
      );
      if (!square) continue;
      const isCapture = Boolean(move?.isCapture);
      if (isCapture) {
        square.classList.add(LEGAL_CAPTURE_DESTINATION_CLASS);
      } else {
        square.classList.add(LEGAL_DESTINATION_CLASS);
      }
      if (Boolean(move?.requiresPromotion)) {
        square.classList.add(LEGAL_PROMOTION_DESTINATION_CLASS);
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
      highlightLegalDestinations(
        Array.isArray(result?.legalMoves) ? result.legalMoves : []
      );
    } catch (_error) {
      if (requestVersion === legalMovesRequestVersion) {
        highlightLegalDestinations([]);
      }
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
    }
  };

  // move chess piece
  const applyMoveOnBoard = (fromFile, fromRank, toFile, toRank) => {
    const fromSquare = document.querySelector(
      squareSelectorByFileRank(fromFile, fromRank)
    );
    const toSquare = document.querySelector(
      squareSelectorByFileRank(toFile, toRank)
    );
    if (!fromSquare || !toSquare) return; // check if there is such box

    // update the position of the picture of the piece
    const pieceEl = fromSquare.querySelector(".piece_img");
    if (!pieceEl) return;

    const captured = toSquare.querySelector(".piece_img");
    if (captured) captured.remove();

    toSquare.appendChild(pieceEl);
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
    const movingSide = currentTurn;
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
        const errorMessage = (await response.text()).trim();

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
        applyMoveOnBoard(
          result.from.file,
          String(result.from.rank),
          result.to.file,
          String(result.to.rank)
        );
      }
      const historyArray = Array.isArray(result.history) ? result.history : [];
      const detailedArray = Array.isArray(result.historyDetailed) ? result.historyDetailed : [];
      if (historyArray.length === 0 && detailedArray.length === 0) {
        // Fallback: if backend omits history payload but move succeeds, still show the move.
        renderMoveHistory([`${sideLabel(movingSide)}: ${String(result.command || command).toLowerCase()}`], []);
      } else {
        renderMoveHistory(historyArray, detailedArray);
      }
      if (
        moveHistoryWhiteList.children.length === 1 &&
        moveHistoryBlackList.children.length === 1 &&
        moveHistoryWhiteList.querySelector(".chess_move_history_placeholder") &&
        moveHistoryBlackList.querySelector(".chess_move_history_placeholder")
      ) {
        const toSquare = `${String(result?.to?.file || "").toLowerCase()}${String(result?.to?.rank || "")}`;
        const targetList = String(movingSide).toLowerCase() === "black" ? moveHistoryBlackList : moveHistoryWhiteList;
        clearHistoryPlaceholder(targetList);
        appendHistoryMove(targetList, movingSide, "pawn", toSquare, String(result.command || command).toLowerCase(), false, "");
      }
      renderCurrentTurn(result.currentTurn);
      renderCheckState(result.checkedSide || result?.game?.outcome?.checkedSide);
      renderGameOutcome(result.game);
      renderGameConfig(result.game);
      // Always compute from the rendered board so capture info matches what user sees.
      const boardStateForInfo = extractBoardStateFromDOM();
      renderGameInfo(boardStateForInfo, result.captured, result.analysis);
      clearSelectedSquare();
      if (result.analysis) {
        stopAnalysisPolling();
      } else {
        const targetMoveNumber = Math.max(historyArray.length, detailedArray.length);
        startAnalysisPolling(targetMoveNumber, boardStateForInfo, result.captured);
      }
      try {
        moveSound.currentTime = 0;
        await moveSound.play();
      } catch (_) {
        // ignore play errors
      }
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
    if (requiresPromotion(fromSequence, toSequence)) {
      const promotionChoice = await requestPromotionChoice();
      if (!promotionChoice) return false;
      command += promotionChoice;
    }
    return submitCommand(command);
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
          fen,
        });
        const response = await fetch(`/api/games/${encodeURIComponent(currentGameId)}/config`, {
          method: "POST",
          headers: { "Content-Type": "application/x-www-form-urlencoded" },
          body: body.toString(),
        });
        if (!response.ok) {
          const errorMessage = (await response.text()).trim();
          setStatus(errorMessage || "Failed to apply game setup.", "error");
          return;
        }
        const result = await response.json();
        syncGameIdFromResult(result);
        renderGameConfig(result.game);
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
          const errorMessage = (await response.text()).trim();
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
        renderGameInfo(extractBoardStateFromDOM(), result.captured, result.analysis);
        cachedAnalysis = null;
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
        const response = await fetch(`/api/games/${encodeURIComponent(currentGameId)}/new`, {
          method: "POST",
        });
        if (!response.ok) {
          const errorMessage = (await response.text()).trim();
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
        renderGameInfo(extractBoardStateFromDOM(), result.captured, result.analysis);
        cachedAnalysis = null;
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

  renderGameInfo(extractBoardStateFromDOM(), null, null);
  renderCheckState("");
  renderGameOutcome({ status: "in_progress", result: "in_progress" });
  renderGameConfig({
    type: "chess",
    mode: "human_vs_human",
    config: { humanColor: "white", aiGameCount: 1, startFen: "" },
  });
  const activeSide = document.querySelector(".game_info_side.game_info_col_active");
  if (activeSide?.textContent) {
    renderCurrentTurn(activeSide.textContent.trim());
  }
})();

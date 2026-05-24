// CM3070 FP code
// chess_command.js
// records the movement command from user
// this is operating on frontend level

(() => {
  const input = document.getElementById("chess_command");
  const button = document.getElementById("chess_command_submit");
  const status = document.getElementById("chess_command_status");
  const whiteColumnCells = document.querySelectorAll(".game_info_col_white");
  const blackColumnCells = document.querySelectorAll(".game_info_col_black");
  const capturedWhiteValue = document.getElementById("game_info_captured_white");
  const capturedBlackValue = document.getElementById("game_info_captured_black");
  const winProbWhiteValue = document.getElementById("game_info_winprob_white");
  const winProbBlackValue = document.getElementById("game_info_winprob_black");
  const moveHistoryWhiteList = document.getElementById("chess_move_history_white");
  const moveHistoryBlackList = document.getElementById("chess_move_history_black");
  const moveSound = new Audio("/sounds/chess_movement.wav");

  if (!input || !button || !status || !moveHistoryWhiteList || !moveHistoryBlackList) return;

  input.focus();

  // set current status
  const setStatus = (message, type) => {
    status.textContent = message;
    status.className = `command_status ${type}`;
  };

  const renderCurrentTurn = (turnText) => {
    if (!turnText) return;
    const isWhiteTurn = turnText.toLowerCase() === "white";
    whiteColumnCells.forEach((cell) => {
      cell.classList.toggle("game_info_col_active", isWhiteTurn);
    });
    blackColumnCells.forEach((cell) => {
      cell.classList.toggle("game_info_col_active", !isWhiteTurn);
    });
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

  const estimateWinProb = (liveCounts) => {
    const pieceValue = { pawn: 1, knight: 3, bishop: 3, rook: 5, queen: 9, king: 0 };
    const material = { white: 0, black: 0 };
    for (const side of ["white", "black"]) {
      for (const kind of Object.keys(pieceValue)) {
        material[side] += (liveCounts[side][kind] || 0) * pieceValue[kind];
      }
    }
    const total = material.white + material.black;
    if (total <= 0) return { white: "50%", black: "50%" };
    const whiteProb = Math.round((material.white / total) * 100);
    const blackProb = 100 - whiteProb;
    return { white: `${whiteProb}%`, black: `${blackProb}%` };
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

  const renderGameInfo = (state, capturedSummary) => {
    const liveCounts = countPiecesByColor(state);
    const normalizedCaptured = normalizeCapturedSummary(capturedSummary);
    const whiteCaptured = normalizedCaptured
      ? normalizedCaptured.white
      : capturedMap("white", liveCounts);
    const blackCaptured = normalizedCaptured
      ? normalizedCaptured.black
      : capturedMap("black", liveCounts);
    const winProb = estimateWinProb(liveCounts);

    if (capturedWhiteValue) capturedWhiteValue.textContent = capturedMapToText(whiteCaptured);
    if (capturedBlackValue) capturedBlackValue.textContent = capturedMapToText(blackCaptured);
    if (winProbWhiteValue) winProbWhiteValue.textContent = `◎ ${winProb.white}`;
    if (winProbBlackValue) winProbBlackValue.textContent = `◎ ${winProb.black}`;
  };

  // update move history from backend source of truth
  const renderMoveHistory = (history) => {
    moveHistoryWhiteList.innerHTML = "";
    moveHistoryBlackList.innerHTML = "";
    if (!Array.isArray(history) || history.length === 0) {
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

    for (const move of history) {
      const item = document.createElement("li");
      if (move.startsWith("White:")) {
        item.textContent = move.replace(/^White:\s*/, "");
        moveHistoryWhiteList.appendChild(item);
      } else if (move.startsWith("Black:")) {
        item.textContent = move.replace(/^Black:\s*/, "");
        moveHistoryBlackList.appendChild(item);
      } else {
        item.textContent = move;
        moveHistoryWhiteList.appendChild(item);
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

  // Full board sync from backend state (handles en passant, castling, promotion)
  const renderBoardFromState = (state) => {
    if (!Array.isArray(state)) return false;

    const boardSquares = document.querySelectorAll(".chess_board_square[data-sequence]");
    boardSquares.forEach((square) => {
      square.querySelectorAll(".piece_img").forEach((el) => el.remove());
    });

    for (const piece of state) {
      if (!piece || !piece.file || !piece.rank || !piece.imgFile) continue;
      const sequence = sequenceByFileRank(piece.file, piece.rank);
      const square = document.querySelector(
        `.chess_board_square[data-sequence="${sequence}"]`
      );
      if (!square) continue;

      const img = document.createElement("img");
      img.className = "piece_img";
      img.src = piece.imgFile.startsWith("/") ? piece.imgFile : `/${piece.imgFile}`;
      img.alt = `piece_${piece.file}_${piece.rank}`;
      img.setAttribute("draggable", "false");
      if (piece.color) img.setAttribute("data-color", String(piece.color).toLowerCase());
      if (piece.kind) img.setAttribute("data-kind", String(piece.kind).toLowerCase());
      square.appendChild(img);
    }

    return true;
  };

  // send the movement command to backend
  const submitCommand = async () => {
    const command = input.value.trim();
    if (!command) {
      setStatus("Please enter a chess movement command.", "error");
      return;
    }
    try {
      const body = new URLSearchParams({ command });
      const response = await fetch("/command", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: body.toString(),
      });

      if (!response.ok) {
        const errorMessage = (await response.text()).trim();

        setStatus(errorMessage || "Invalid command format", "error");
        input.focus();

        return;
      }

      const result = await response.json();
      if (!result?.from || !result?.to) {
        setStatus("Invalid move response from server", "error");
        input.focus();
        return;
      }

      input.value = "";
      setStatus("Command submitted", "success");
      const usedStateRender = renderBoardFromState(result.state);
      if (!usedStateRender) {
        applyMoveOnBoard(
          result.from.file,
          String(result.from.rank),
          result.to.file,
          String(result.to.rank)
        );
      }
      renderMoveHistory(result.history);
      renderCurrentTurn(result.currentTurn);
      // Always compute from the rendered board so capture info matches what user sees.
      renderGameInfo(extractBoardStateFromDOM(), result.captured);
      try {
        moveSound.currentTime = 0;
        await moveSound.play();
      } catch (_) {
        // ignore play errors
      }
      input.focus();
    } catch (_error) {
      setStatus("Network error. Please try again.", "error");
      input.focus();
    }
  };

  button.addEventListener("click", submitCommand);
  input.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      submitCommand();
    }
  });

  renderGameInfo(extractBoardStateFromDOM());
  const activeSide = document.querySelector(".game_info_side.game_info_col_active");
  if (activeSide?.textContent) {
    renderCurrentTurn(activeSide.textContent.trim());
  }
})();

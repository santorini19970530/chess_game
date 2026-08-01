// CM3070 FP code
// 08_game_info.js - side panel: win%, captures, turn/check, and outcome labels
// GameInfoView - paints game info, captures, and outcome (history is MoveHistoryView)
class GameInfoView {
  constructor(app) {
    this.app = app;
    this.app.CHESS_PIECE_ORDER = ["queen", "rook", "bishop", "knight", "pawn", "king"];
    this.app.XIANQI_PIECE_ORDER = ["cannon", "rook", "knight", "elephant", "advisor", "pawn", "king"];
    // unpromoted hand kinds only — promoted captives demote before entering hand
    this.app.SHOGI_PIECE_ORDER = ["rook", "bishop", "gold", "silver", "knight", "lance", "pawn"];
    this.app.SHOGI_DROP_CHAR = {
      pawn: "P",
      lance: "L",
      knight: "N",
      silver: "S",
      gold: "G",
      bishop: "B",
      rook: "R",
    };
  }

  // renderCurrentTurn - highlights the active side column and refreshes suggestions
  renderCurrentTurn(turnText) {
    if (!turnText) return;
    const isWhiteTurn = turnText.toLowerCase() === "white";
    this.app.state.currentTurn = isWhiteTurn ? "white" : "black";
    this.app.el.whiteColumnCells.forEach((cell) => {
      cell.classList.toggle("game_info_col_active", isWhiteTurn);
    });
    this.app.el.blackColumnCells.forEach((cell) => {
      cell.classList.toggle("game_info_col_active", !isWhiteTurn);
    });
    void this.app.coach.refreshSuggestedMoves();
  }

  // renderCheckState - toggles check styling on white/black info columns
  renderCheckState(checkedSide) {
    const side = String(checkedSide || "").toLowerCase();
    const whiteChecked = side === "white";
    const blackChecked = side === "black";
    this.app.el.whiteColumnCells.forEach((cell) => {
      cell.classList.toggle(this.app.CHECK_CLASS, whiteChecked);
    });
    this.app.el.blackColumnCells.forEach((cell) => {
      cell.classList.toggle(this.app.CHECK_CLASS, blackChecked);
    });
  }

  // capitalize - capitalizes the first letter of a side or status word
  capitalize(text) {
    const value = String(text || "").toLowerCase();
    if (!value) return "";
    return value.charAt(0).toUpperCase() + value.slice(1);
  }

  // drawReasonText - maps a draw status value to a short result label
  drawReasonText(statusValue) {
    switch (statusValue) {
      case "stalemate":
        return "draw by stalemate";
      case "draw_insufficient_material":
        return "draw by insufficient material";
      case "draw_threefold_repetition":
        return "draw by threefold repetition";
      case "draw_fifty_move_rule":
        return "draw by 50-move rule";
      default:
        return "draw";
    }
  }

  // paintResultLabels - paints win/loss/draw/playing labels for both sides
  paintResultLabels(gameResult, statusValue) {
    if (!this.app.el.resultWhiteValue || !this.app.el.resultBlackValue) return;
    const reset = (el) => {
      el.classList.remove("game_info_result_win", "game_info_result_loss", "game_info_result_draw");
    };
    reset(this.app.el.resultWhiteValue);
    reset(this.app.el.resultBlackValue);
    switch (gameResult) {
      case "white_win":
        this.app.el.resultWhiteValue.textContent = "Result: WIN";
        this.app.el.resultBlackValue.textContent = "Result: LOSS";
        this.app.el.resultWhiteValue.classList.add("game_info_result_win");
        this.app.el.resultBlackValue.classList.add("game_info_result_loss");
        break;
      case "black_win":
        this.app.el.resultWhiteValue.textContent = "Result: LOSS";
        this.app.el.resultBlackValue.textContent = "Result: WIN";
        this.app.el.resultWhiteValue.classList.add("game_info_result_loss");
        this.app.el.resultBlackValue.classList.add("game_info_result_win");
        break;
      case "draw": {
        const drawLabel = `Result: ${this.drawReasonText(statusValue)}`;
        this.app.el.resultWhiteValue.textContent = drawLabel;
        this.app.el.resultBlackValue.textContent = drawLabel;
        this.app.el.resultWhiteValue.classList.add("game_info_result_draw");
        this.app.el.resultBlackValue.classList.add("game_info_result_draw");
        break;
      }
      default:
        this.app.el.resultWhiteValue.textContent = "Result: PLAYING";
        this.app.el.resultBlackValue.textContent = "Result: PLAYING";
        break;
    }
  }

  // endGameUi - disables play controls and shows an ended-game status/notes message
  endGameUi(statusMsg, statusType, notesMsg) {
    this.app.util.setStatus(statusMsg, statusType);
    this.app.el.input.disabled = true;
    this.app.el.button.disabled = true;
    if (this.app.el.flagButton) this.app.el.flagButton.disabled = true;
    this.app.state.gameOver = true;
    this.app.clocks.stopClockTick();
    this.app.util.showGameEndedNotes(notesMsg);
    this.app.setup.updateSetupControlState();
  }

  // applyOutcomeStatus - applies terminal or in-progress status from an outcome payload
  applyOutcomeStatus(outcome, statusValue) {
    switch (statusValue) {
      case "checkmate": {
        const winner = this.capitalize(outcome?.winner);
        const loser = this.capitalize(outcome?.loser);
        this.endGameUi(
          `Checkmate! ${winner} wins. ${loser} loses.`,
          "error",
          `Game has ended. Checkmate — ${winner} wins.`
        );
        return;
      }
      case "stalemate":
        this.endGameUi("Draw by stalemate.", "success", "Game has ended. Draw by stalemate.");
        return;
      case "draw_insufficient_material":
      case "draw_threefold_repetition":
      case "draw_fifty_move_rule":
        this.endGameUi(outcome?.message || "Game drawn.", "success", outcome?.message || "Game has ended. Draw.");
        return;
      case "resigned":
        this.endGameUi(
          outcome?.message || "Game ended by flag.",
          "error",
          outcome?.message || "Game has ended (flag / resign)."
        );
        return;
      default:
        if (statusValue.startsWith("draw_")) {
          this.endGameUi(outcome?.message || "Game drawn.", "success", outcome?.message || "Game has ended. Draw.");
          return;
        }
        break;
    }

    this.app.enablePlayInputs();
    if (statusValue === "check") {
      const checked = this.capitalize(outcome?.checkedSide);
      const legalMoves = Number(outcome?.legalMoves || 0);
      this.app.util.setStatus(`${checked} is in check. Legal moves available: ${legalMoves}.`, "error");
      return;
    }
    this.app.util.setStatus("", "success");
  }

  // renderGameOutcome - paints result labels and end-game ui from an outcome payload
  renderGameOutcome(game) {
    const outcome = game?.outcome || game || {};
    const statusValue = String(outcome?.status || "").toLowerCase();
    const gameResult = String(game?.result || "in_progress").toLowerCase();
    this.paintResultLabels(gameResult, statusValue);
    this.applyOutcomeStatus(outcome, statusValue);
  }

  // capturedPieceOrder - returns the capture/hand piece order for the current game type
  capturedPieceOrder() {
    if (this.app.state.boardGameType === "xianqi") return this.app.XIANQI_PIECE_ORDER;
    if (this.app.state.boardGameType === "shogi") return this.app.SHOGI_PIECE_ORDER;
    return this.app.CHESS_PIECE_ORDER;
  }

  // isHandSidePlayable - reports whether a shogi hand side may drop this turn
  isHandSidePlayable(side) {
    const s = String(side || "").toLowerCase();
    if (s !== this.app.state.currentTurn) return false;
    const mode = String(this.app.el.gameModeSelect?.value || "");
    if (mode === "human_vs_ai" && s !== this.app.state.humanColor) return false;
    return true;
  }

  // renderCapturedIcons - renders capture or shogi-hand icons for one side
  renderCapturedIcons(el, side, captured) {
    if (!el) return;
    el.replaceChildren();
    el.classList.add("shogi_hand");
    const droppable = this.app.state.boardGameType === "shogi";
    const iconColor = droppable ? side : side === "white" ? "black" : "white";
    let any = false;
    for (const kind of this.capturedPieceOrder()) {
      const count = captured[kind] || 0;
      if (count <= 0) continue;
      const path = this.app.board.imagePathFromPiece({ kind, color: iconColor });
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
        this.app.state.selectedDropKind &&
        this.app.state.selectedDropKind.side === side &&
        this.app.state.selectedDropKind.kind === kind
      ) {
        node.classList.add("shogi_hand_piece_selected");
      }
      if (droppable) this.app.coach.applyHandHintToNode(node, side, kind);
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
        node.disabled = !this.isHandSidePlayable(side);
        node.addEventListener("click", (event) => {
          event.preventDefault();
          event.stopPropagation();
          void this.app.interaction.selectShogiHandPiece(side, kind);
        });
      }
      el.appendChild(node);
    }
    if (!any) {
      el.classList.remove("shogi_hand");
    }
  }

  // emptyCapturedSide - builds a zeroed captured-piece map for the current game type
  emptyCapturedSide() {
    const side = {};
    for (const kind of this.capturedPieceOrder()) side[kind] = 0;
    return side;
  }

  // normalizeCapturedSummary - normalizes a captured summary into white/black maps
  normalizeCapturedSummary(summary) {
    const order = this.capturedPieceOrder();
    if (!summary || typeof summary !== "object") {
      return { white: this.emptyCapturedSide(), black: this.emptyCapturedSide() };
    }
    const normalized = { white: this.emptyCapturedSide(), black: this.emptyCapturedSide() };
    for (const side of ["white", "black"]) {
      const source = summary[side];
      if (!source || typeof source !== "object") continue;
      for (const kind of order) {
        const value = Number(source[kind]);
        normalized[side][kind] = Number.isFinite(value) && value > 0 ? value : 0;
      }
    }
    return normalized;
  }

  // clampPercentage - clamps a win-probability percentage into 0..100
  clampPercentage(value) {
    const n = Number(value);
    if (!Number.isFinite(n)) return 50;
    return Math.max(0, Math.min(100, n));
  }

  // winProbLabelColor - picks a readable label color for a win-probability value
  winProbLabelColor(chance, isLightBackground) {
    if (chance >= 70) return isLightBackground ? "#0f5e2a" : "#8df0a8";
    if (chance <= 30) return isLightBackground ? "#7a1e1e" : "#ff9f9f";
    return isLightBackground ? "#101010" : "#f5f5f5";
  }

  // fromAnalyzerChance - converts analyzer chance (0..1 or 0..100) into a percentage
  fromAnalyzerChance(value) {
    const n = Number(value);
    if (!Number.isFinite(n)) return null;
    // Analyzer uses 0..1; tolerate 0..100 values too.
    return n <= 1 ? n * 100 : n;
  }

  // renderGameInfo - paints captures, win% bars, and threat notes from analysis
  renderGameInfo(capturedSummary, analysis) {
    if (capturedSummary) this.app.state.cachedCapturedSummary = capturedSummary;
    const effectiveCapturedSummary = capturedSummary || this.app.state.cachedCapturedSummary;
    const effectiveAnalysis = analysis || this.app.state.cachedAnalysis;
    if (analysis) this.app.state.cachedAnalysis = analysis;
    const normalizedCaptured = this.normalizeCapturedSummary(effectiveCapturedSummary);
    const whiteCaptured = normalizedCaptured.white;
    const blackCaptured = normalizedCaptured.black;
    const analyzerWhite = this.fromAnalyzerChance(effectiveAnalysis?.win_chance_white);
    const analyzerBlack = this.fromAnalyzerChance(effectiveAnalysis?.win_chance_black);
    const hasAnalyzerProb = analyzerWhite != null && analyzerBlack != null;
    const whiteProb = this.clampPercentage(hasAnalyzerProb ? analyzerWhite : 50);
    const blackProb = this.clampPercentage(hasAnalyzerProb ? analyzerBlack : 50);
    const whiteTiny = whiteProb < 12;
    const blackTiny = blackProb < 12;

    this.renderCapturedIcons(this.app.el.capturedWhiteValue, "white", whiteCaptured);
    this.renderCapturedIcons(this.app.el.capturedBlackValue, "black", blackCaptured);
    if (this.app.el.winProbWhiteValue) {
      this.app.el.winProbWhiteValue.textContent = this.formatPercentage(whiteProb);
      this.app.el.winProbWhiteValue.style.color = this.winProbLabelColor(whiteProb, true);
      this.app.el.winProbWhiteValue.classList.toggle("game_info_winprob_label_outside_white", whiteTiny);
    }
    if (this.app.el.winProbBlackValue) {
      this.app.el.winProbBlackValue.textContent = this.formatPercentage(blackProb);
      this.app.el.winProbBlackValue.style.color = this.winProbLabelColor(blackProb, false);
      this.app.el.winProbBlackValue.classList.toggle("game_info_winprob_label_outside_black", blackTiny);
    }
    if (this.app.el.winProbWhiteBar) this.app.el.winProbWhiteBar.style.width = `${whiteProb}%`;
    if (this.app.el.winProbBlackBar) this.app.el.winProbBlackBar.style.width = `${blackProb}%`;
    if (this.app.el.winProbWhiteBar)
      this.app.el.winProbWhiteBar.classList.toggle("game_info_winprob_segment_tiny", whiteTiny);
    if (this.app.el.winProbBlackBar)
      this.app.el.winProbBlackBar.classList.toggle("game_info_winprob_segment_tiny", blackTiny);

    if (this.app.el.gameInfoNotesBox && effectiveAnalysis && !this.app.state.gameOver) {
      const threatSummary = String(effectiveAnalysis?.threat_summary || "").trim();
      // Skip empty / stub lines that looked like Fairy-Stockfish authored the coach note.
      const stubThreat = new Set(["", "Position evaluated with Fairy-Stockfish.", "No analysis summary yet."]);
      this.app.state.lastThreatSummary = stubThreat.has(threatSummary) ? "" : threatSummary;
      this.app.util.refreshNotesBox();
    }
  }

  // formatPercentage - formats a percentage with one decimal place
  formatPercentage(value) {
    return `${value.toFixed(1)}%`;
  }
}

window.GameInfoView = GameInfoView;

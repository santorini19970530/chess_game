// CM3070 FP code
// 11_simulation.js - ai simulation run, playback, and download controls

// SimulationPanel - owns run-sim button, playback helpers, and download buttons
class SimulationPanel {
  constructor(app) {
    this.app = app;
    this.bindDownloadButtons();
    this.bindRunButton();
  }

  // bindDownloadButtons - wires simulation json/csv download clicks
  bindDownloadButtons() {
    if (this.app.el.simulationDownloadJsonBtn) {
      this.app.el.simulationDownloadJsonBtn.addEventListener("click", () => {
        const json = this.buildSimulationJSON();
        if (!json) {
          this.app.util.setStatus("Run a simulation first to download results.", "error");
          return;
        }
        this.app.util.downloadTextFile(this.simulationDownloadFilename("json"), "application/json", json);
      });
    }

    if (this.app.el.simulationDownloadCsvBtn) {
      this.app.el.simulationDownloadCsvBtn.addEventListener("click", () => {
        const csv = this.buildSimulationCSV();
        if (!csv) {
          this.app.util.setStatus("Run a simulation first to download results.", "error");
          return;
        }
        this.app.util.downloadTextFile(this.simulationDownloadFilename("csv"), "text/csv", csv);
      });
    }
  }

  // setSimulationDownloadEnabled - enables or disables simulation download buttons
  setSimulationDownloadEnabled(enabled) {
    if (this.app.el.simulationDownloadJsonBtn) this.app.el.simulationDownloadJsonBtn.disabled = !enabled;
    if (this.app.el.simulationDownloadCsvBtn) this.app.el.simulationDownloadCsvBtn.disabled = !enabled;
  }

  // buildSimulationExportPayload - builds the json/csv export object from the last simulation
  buildSimulationExportPayload() {
    if (!this.app.state.simulationData || !Array.isArray(this.app.state.simulationData.results)) return null;
    return {
      exported_at: new Date().toISOString(),
      profile: String(this.app.el.aiStrengthSelect?.value || "intermediate"),
      mode: String(this.app.el.gameModeSelect?.value || "ai_vs_ai"),
      game_type: String(this.app.el.gameTypeSelect?.value || "chess"),
      summary: {
        games: Number(this.app.state.simulationData.games || 0),
        white_wins: Number(this.app.state.simulationData.white_wins || 0),
        black_wins: Number(this.app.state.simulationData.black_wins || 0),
        draws: Number(this.app.state.simulationData.draws || 0),
        avg_moves: Number(this.app.state.simulationData.avg_moves || 0),
      },
      results: this.app.state.simulationData.results,
    };
  }

  // buildSimulationJSON - serialises the simulation export payload as json text
  buildSimulationJSON() {
    const payload = this.buildSimulationExportPayload();
    return payload ? JSON.stringify(payload, null, 2) : "";
  }

  // csvEscape - escapes a csv cell when it contains commas, quotes, or newlines
  csvEscape(value) {
    const text = String(value ?? "");
    if (/[",\n]/.test(text)) return `"${text.replace(/"/g, '""')}"`;
    return text;
  }

  // buildSimulationCSV - serialises the simulation export payload as csv text
  buildSimulationCSV() {
    const payload = this.buildSimulationExportPayload();
    if (!payload) return "";
    const lines = ["game,result,winner,moves"];
    for (let i = 0; i < payload.results.length; i++) {
      const row = payload.results[i] || {};
      lines.push(
        [i + 1, this.csvEscape(row.result || ""), this.csvEscape(row.winner || ""), Number(row.moves || 0)].join(",")
      );
    }
    const summary = payload.summary;
    lines.push(
      `# Summary,${summary.games} games,White ${summary.white_wins},Black ${summary.black_wins},Draws ${summary.draws},Avg ${Number(summary.avg_moves || 0).toFixed(1)}`
    );
    return lines.join("\n");
  }

  // simulationDownloadFilename - builds a timestamped download filename for simulation exports
  simulationDownloadFilename(ext) {
    const profile = String(this.app.el.aiStrengthSelect?.value || "intermediate");
    const stamp = new Date().toISOString().replace(/[:.]/g, "-");
    return `simulation-${profile}-${stamp}.${ext}`;
  }

  // clearSimulationSummary - resets the simulation summary panel to empty hidden state
  clearSimulationSummary() {
    if (this.app.el.simulationSummaryGames) this.app.el.simulationSummaryGames.textContent = "0";
    if (this.app.el.simulationSummaryWhite) this.app.el.simulationSummaryWhite.textContent = "0";
    if (this.app.el.simulationSummaryBlack) this.app.el.simulationSummaryBlack.textContent = "0";
    if (this.app.el.simulationSummaryDraws) this.app.el.simulationSummaryDraws.textContent = "0";
    if (this.app.el.simulationSummaryAvg) this.app.el.simulationSummaryAvg.textContent = "0.0";
    if (this.app.el.simulationResultList) this.app.el.simulationResultList.innerHTML = "";
    if (this.app.el.simulationResultSummaryText) this.app.el.simulationResultSummaryText.textContent = "Per-game results";
    if (this.app.el.simulationResultDetails) this.app.el.simulationResultDetails.open = false;
    if (this.app.el.simulationSummaryPanel) this.app.el.simulationSummaryPanel.classList.add("simulation_summary_hidden");
    this.setSimulationDownloadEnabled(false);
  }

  // renderSimulationSummary - paints summary counts and per-game rows after a simulation run
  renderSimulationSummary(summary) {
    const games = Number(summary?.games || 0);
    const whiteWins = Number(summary?.white_wins || 0);
    const blackWins = Number(summary?.black_wins || 0);
    const draws = Number(summary?.draws || 0);
    const avgMoves = Number(summary?.avg_moves || 0);

    if (this.app.el.simulationSummaryGames) this.app.el.simulationSummaryGames.textContent = String(games);
    if (this.app.el.simulationSummaryWhite) this.app.el.simulationSummaryWhite.textContent = String(whiteWins);
    if (this.app.el.simulationSummaryBlack) this.app.el.simulationSummaryBlack.textContent = String(blackWins);
    if (this.app.el.simulationSummaryDraws) this.app.el.simulationSummaryDraws.textContent = String(draws);
    if (this.app.el.simulationSummaryAvg) {
      this.app.el.simulationSummaryAvg.textContent = Number.isFinite(avgMoves) ? avgMoves.toFixed(1) : "0.0";
    }

    if (this.app.el.simulationResultList) {
      this.app.el.simulationResultList.innerHTML = "";
      const results = Array.isArray(summary?.results) ? summary.results : [];
      if (this.app.el.simulationResultSummaryText) {
        this.app.el.simulationResultSummaryText.textContent = `Per-game results (${results.length})`;
      }
      if (this.app.el.simulationResultDetails) this.app.el.simulationResultDetails.open = false;
      for (let i = 0; i < results.length; i++) {
        const item = document.createElement("li");
        const one = results[i] || {};
        item.textContent = `Game ${i + 1}: ${String(one.result || "unknown")} | winner: ${String(one.winner || "-")} | moves: ${Number(one.moves || 0)}`;
        this.app.el.simulationResultList.appendChild(item);
      }
    }

    if (this.app.el.simulationSummaryPanel) this.app.el.simulationSummaryPanel.classList.remove("simulation_summary_hidden");
    this.setSimulationDownloadEnabled(Array.isArray(summary?.results) && summary.results.length > 0);
  }

  // readSimulationCount - validates the ai game count input for simulation runs
  readSimulationCount() {
    const raw = String(this.app.el.aiGameCountInput?.value || "").trim();
    const n = Number(raw);
    if (!Number.isInteger(n) || n < 1 || n > 1000) {
      return { ok: false, message: "Please enter an integer game count between 1 and 1000." };
    }
    return { ok: true, count: n };
  }

  // bindRunButton - creates and wires the run ai simulation control
  bindRunButton() {
    if (!this.app.el.aiGameCountInput || !this.app.el.configApplyButton) return;

    this.app.state.simRunBtn = document.createElement("button");
    this.app.state.simRunBtn.id = "run_simulation_btn";
    this.app.state.simRunBtn.type = "button";
    this.app.state.simRunBtn.textContent = "Run AI Simulation";
    this.app.state.simRunBtn.className = "run-simulation-btn";
    this.app.el.configApplyButton.insertAdjacentElement("afterend", this.app.state.simRunBtn);
    this.app.setup.updateSetupControlState();
    this.app.state.simRunBtn.addEventListener("click", () => {
      void this.onRunSimulation();
    });
  }

  // onRunSimulation - requests an ai-vs-ai simulation and starts manual playback
  async onRunSimulation() {
    const parsed = this.readSimulationCount();
    if (!parsed.ok) {
      this.app.util.setStatus(parsed.message, "error");
      return;
    }
    if (this.app.state.simRunBtn.disabled) return;

    const n = parsed.count;
    const profile = String(this.app.el.aiStrengthSelect?.value || "intermediate");
    this.app.interaction.clearSelectedSquare();
    this.app.coach.highlightSuggestedMoves([]);
    this.app.util.setNotesText("Simulation running...");
    this.app.util.setStatus("Running AI simulation...", "success");

    this.app.state.simRunBtn.disabled = true;
    this.app.state.isSimulationPlayback = true;
    this.app.state.simulationRequestInFlight = true;
    this.app.state.simulationData = null;
    this.app.state.currentSimGameIdx = -1;
    this.app.state.currentSimMoveIdx = 0;
    this.clearSimulationSummary();
    this.app.setup.updateSetupControlState();

    try {
      const resp = await fetch("/api/simulate?details=true", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          games: n,
          profile,
          game: String(this.app.el.gameTypeSelect?.value || this.app.state.boardGameType || "chess"),
        }),
      });
      this.app.state.simulationRequestInFlight = false;
      this.app.setup.updateSetupControlState();

      if (!resp.ok) {
        const errorMessage = await this.app.util.readErrorMessage(resp, "Simulation request failed.");
        if (resp.status === 409) {
          this.app.util.setStatus(`Simulation already running on server. ${errorMessage}`, "error");
        } else {
          this.app.util.setStatus(`Simulation failed: ${errorMessage}`, "error");
        }
        this.cleanupSimulationControls();
        return;
      }

      const payload = await resp.json();
      if (!Array.isArray(payload?.results)) {
        this.app.util.setStatus("Simulation failed: missing results payload.", "error");
        this.cleanupSimulationControls();
        return;
      }

      this.app.state.simulationData = payload;
      this.renderSimulationSummary(this.app.state.simulationData);
      if (this.app.state.simRunBtn) this.app.state.simRunBtn.style.display = "none";
      this.ensureSimulationControls();
      this.startNextSimulationGame();
      this.app.util.setStatus(`Simulation loaded (${n} game${n > 1 ? "s" : ""}).`, "success");
    } catch (error) {
      this.app.state.simulationRequestInFlight = false;
      this.app.setup.updateSetupControlState();
      this.app.util.setCatchStatus(error, "Network error while loading simulation.");
      this.cleanupSimulationControls();
    }
  }

  // initialChessState - builds the starting piece list for western chess
  initialChessState() {
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

  // resetSimulationHistoryPanels - clears white/black move lists for a new sim game
  resetSimulationHistoryPanels() {
    if (this.app.el.moveHistoryWhiteList) {
      this.app.el.moveHistoryWhiteList.innerHTML = '<li class="chess_move_history_placeholder">No moves yet.</li>';
    }
    if (this.app.el.moveHistoryBlackList) {
      this.app.el.moveHistoryBlackList.innerHTML = '<li class="chess_move_history_placeholder">No moves yet.</li>';
    }
  }

  // resetSimulationCapturedPanel - clears the side-panel captured icons for a new sim game
  resetSimulationCapturedPanel() {
    this.app.gameInfo.renderGameInfo(this.app.gameInfo.normalizeCapturedSummary(null), null);
  }

  // recordSimulationCapture - adds one captured piece to the side panel during next-move playback
  recordSimulationCapture(side, capturedPieceKind) {
    const kind = String(capturedPieceKind || "").toLowerCase();
    if (!kind) return;
    const capturer = String(side || "").toLowerCase() === "black" ? "black" : "white";
    const summary = this.app.gameInfo.normalizeCapturedSummary(this.app.state.cachedCapturedSummary);
    summary[capturer][kind] = (summary[capturer][kind] || 0) + 1;
    this.app.gameInfo.renderGameInfo(summary, null);
  }

  // initialXiangqiState - builds the starting piece list for xiangqi
  initialXiangqiState() {
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

  // initialShogiState - builds the starting piece list for shogi
  initialShogiState() {
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

  // resetBoardToInitialState - restores the board preview for the selected game type
  resetBoardToInitialState() {
    this.app.board.previewBoardForGameType(this.app.el.gameTypeSelect?.value || this.app.state.boardGameType);
  }

  // clearResultLabelClasses - strips win/loss/draw classes from result labels
  clearResultLabelClasses() {
    if (this.app.el.resultWhiteValue) {
      this.app.el.resultWhiteValue.classList.remove(
        "game_info_result_win",
        "game_info_result_loss",
        "game_info_result_draw"
      );
    }
    if (this.app.el.resultBlackValue) {
      this.app.el.resultBlackValue.classList.remove(
        "game_info_result_win",
        "game_info_result_loss",
        "game_info_result_draw"
      );
    }
  }

  // setPlayingResultLabels - shows playing labels while a sim game is in progress
  setPlayingResultLabels() {
    this.clearResultLabelClasses();
    if (this.app.el.resultWhiteValue) this.app.el.resultWhiteValue.textContent = "Result: PLAYING";
    if (this.app.el.resultBlackValue) this.app.el.resultBlackValue.textContent = "Result: PLAYING";
  }

  // applySimulationResultLabels - paints win/loss/draw labels for a finished simulation game
  applySimulationResultLabels(gameResult) {
    this.clearResultLabelClasses();
    const resultText = String(gameResult?.result || "").toLowerCase();
    switch (resultText) {
      case "white_win":
        if (this.app.el.resultWhiteValue) {
          this.app.el.resultWhiteValue.textContent = "Result: WIN";
          this.app.el.resultWhiteValue.classList.add("game_info_result_win");
        }
        if (this.app.el.resultBlackValue) {
          this.app.el.resultBlackValue.textContent = "Result: LOSS";
          this.app.el.resultBlackValue.classList.add("game_info_result_loss");
        }
        break;
      case "black_win":
        if (this.app.el.resultWhiteValue) {
          this.app.el.resultWhiteValue.textContent = "Result: LOSS";
          this.app.el.resultWhiteValue.classList.add("game_info_result_loss");
        }
        if (this.app.el.resultBlackValue) {
          this.app.el.resultBlackValue.textContent = "Result: WIN";
          this.app.el.resultBlackValue.classList.add("game_info_result_win");
        }
        break;
      default:
        if (this.app.el.resultWhiteValue) {
          this.app.el.resultWhiteValue.textContent = "Result: DRAW";
          this.app.el.resultWhiteValue.classList.add("game_info_result_draw");
        }
        if (this.app.el.resultBlackValue) {
          this.app.el.resultBlackValue.textContent = "Result: DRAW";
          this.app.el.resultBlackValue.classList.add("game_info_result_draw");
        }
        break;
    }
  }

  // ensureSimulationControls - creates next-move and next-game buttons when missing
  ensureSimulationControls() {
    if (!this.app.el.configApplyButton || !this.app.el.configApplyButton.parentNode) return;

    if (!this.app.state.simNextMoveBtn) {
      this.app.state.simNextMoveBtn = document.createElement("button");
      this.app.state.simNextMoveBtn.id = "sim_next_move_btn";
      this.app.state.simNextMoveBtn.type = "button";
      this.app.state.simNextMoveBtn.textContent = "Next Move";
      this.app.state.simNextMoveBtn.className = "run-simulation-btn";
      this.app.state.simNextMoveBtn.addEventListener("click", this.playNextSimulationMove.bind(this));
      this.app.el.configApplyButton.parentNode.appendChild(this.app.state.simNextMoveBtn);
    }

    if (!this.app.state.simNextGameBtn) {
      this.app.state.simNextGameBtn = document.createElement("button");
      this.app.state.simNextGameBtn.id = "sim_next_game_btn";
      this.app.state.simNextGameBtn.type = "button";
      this.app.state.simNextGameBtn.textContent = "Next Game";
      this.app.state.simNextGameBtn.className = "run-simulation-btn";
      this.app.state.simNextGameBtn.style.display = "none";
      this.app.state.simNextGameBtn.addEventListener("click", this.startNextSimulationGame.bind(this));
      this.app.el.configApplyButton.parentNode.appendChild(this.app.state.simNextGameBtn);
    }
  }

  // finishCurrentSimulationGame - shows result labels and advances to next-game or cleanup
  finishCurrentSimulationGame() {
    const gameResult = this.app.state.simulationData?.results?.[this.app.state.currentSimGameIdx] || null;
    if (!gameResult) return;
    this.applySimulationResultLabels(gameResult);
    if (this.app.state.simNextMoveBtn) this.app.state.simNextMoveBtn.style.display = "none";
    if (this.app.state.simNextGameBtn) this.app.state.simNextGameBtn.style.display = "inline-block";
    const totalGames = Array.isArray(this.app.state.simulationData?.results)
      ? this.app.state.simulationData.results.length
      : 0;
    const isLastGame = this.app.state.currentSimGameIdx >= totalGames - 1;
    if (isLastGame) {
      this.app.util.setStatus(
        `Game ${this.app.state.currentSimGameIdx + 1} finished. All simulation games completed.`,
        "success"
      );
      this.cleanupSimulationControls();
    } else {
      this.app.util.setStatus(`Game ${this.app.state.currentSimGameIdx + 1} finished. Click Next Game.`, "success");
    }
  }

  // startNextSimulationGame - resets the board and history for the next simulation game
  startNextSimulationGame() {
    if (!this.app.state.simulationData || !Array.isArray(this.app.state.simulationData.results)) return;

    this.app.state.currentSimGameIdx++;
    this.app.state.currentSimMoveIdx = 0;

    if (this.app.state.currentSimGameIdx >= this.app.state.simulationData.results.length) {
      this.app.util.setStatus("All simulation games completed.", "success");
      this.cleanupSimulationControls();
      return;
    }

    if (this.app.state.simNextGameBtn) {
      this.app.state.simNextGameBtn.textContent = "Next Game";
      this.app.state.simNextGameBtn.disabled = false;
      this.app.state.simNextGameBtn.style.display = "none";
    }
    if (this.app.state.simNextMoveBtn) this.app.state.simNextMoveBtn.style.display = "inline-block";

    this.resetBoardToInitialState();
    this.resetSimulationHistoryPanels();
    this.resetSimulationCapturedPanel();
    this.setPlayingResultLabels();
    this.app.coach.highlightSuggestedMoves([]);

    const totalGames = this.app.state.simulationData.results.length;
    this.app.util.setNotesText(`Simulation playback: Game ${this.app.state.currentSimGameIdx + 1}/${totalGames}`);
    this.app.util.setStatus(`Game ${this.app.state.currentSimGameIdx + 1} ready. Click Next Move.`, "success");
  }

  // playNextSimulationMove - applies the next uci move from the current simulation game
  playNextSimulationMove() {
    const gameResult = this.app.state.simulationData?.results?.[this.app.state.currentSimGameIdx];
    if (!gameResult) return;
    const moves = Array.isArray(gameResult.history_detailed) ? gameResult.history_detailed : [];

    if (moves.length === 0) {
      this.app.util.setStatus(`Game ${this.app.state.currentSimGameIdx + 1} has no move history in response.`, "error");
      this.finishCurrentSimulationGame();
      return;
    }
    if (this.app.state.currentSimMoveIdx >= moves.length) {
      this.finishCurrentSimulationGame();
      return;
    }

    const moveEntry = moves[this.app.state.currentSimMoveIdx] || {};
    const uciMove = String(moveEntry.command || "").trim();
    if (uciMove) {
      this.app.board.applyUciMoveToBoard(uciMove);
      const isCapture = Boolean(moveEntry.isCapture);
      this.app.dom.playMoveSound(isCapture);
      const side = String(
        moveEntry.side || (this.app.state.currentSimMoveIdx % 2 === 0 ? "white" : "black")
      ).toLowerCase();
      const capturedPieceKind = String(moveEntry.capturedPieceKind || "");
      if (isCapture) this.recordSimulationCapture(side, capturedPieceKind);
      const listEl = side === "black" ? this.app.el.moveHistoryBlackList : this.app.el.moveHistoryWhiteList;
      if (listEl) {
        this.app.moveHistory.clearHistoryPlaceholder(listEl);
        this.app.moveHistory.appendHistoryMove(
          listEl,
          side,
          String(moveEntry.pieceKind || "pawn"),
          String(moveEntry.to || ""),
          this.app.moveHistory.destinationFromCommand(uciMove),
          isCapture,
          capturedPieceKind
        );
        listEl.scrollTop = listEl.scrollHeight;
      }
      const line = `#${this.app.state.currentSimMoveIdx + 1} ${uciMove}`;
      this.app.util.appendNotesLine(line);
    }
    this.app.state.currentSimMoveIdx++;
  }

  // cleanupSimulationControls - removes playback buttons and restores the run-sim control
  cleanupSimulationControls() {
    if (this.app.state.simNextMoveBtn) {
      this.app.state.simNextMoveBtn.remove();
      this.app.state.simNextMoveBtn = null;
    }
    if (this.app.state.simNextGameBtn) {
      this.app.state.simNextGameBtn.remove();
      this.app.state.simNextGameBtn = null;
    }
    if (this.app.state.simRunBtn) {
      this.app.state.simRunBtn.textContent = "Run AI Simulation";
      this.app.state.simRunBtn.style.display = "inline-block";
      this.app.state.simRunBtn.disabled = false;
      if (!this.app.util.isAIVsAIModeSelected()) {
        this.app.state.simRunBtn.style.display = "none";
      }
    }
    this.app.state.currentSimGameIdx = 0;
    this.app.state.currentSimMoveIdx = 0;
    this.app.state.simulationRequestInFlight = false;
    this.app.state.isSimulationPlayback = false;
    this.app.setup.updateSetupControlState();
  }
}

window.SimulationPanel = SimulationPanel;

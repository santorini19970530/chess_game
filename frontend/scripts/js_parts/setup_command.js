// CM3070 FP code
// setup_command.js - setup form controls and move submit for the puzzle page

// SetupCommand - owns setup controls and move submit
class SetupCommand {
  constructor(app) {
    this.app = app;
    this.bindFormListeners();
  }

  // bindFormListeners - wires setup form inputs to preview, sync, and enable/disable updates
  bindFormListeners() {
    this.app.el.button.addEventListener("click", this.submitCommand.bind(this));
    if (this.app.el.previewButton) {
      this.app.el.previewButton.addEventListener("click", () => this.togglePreviewMode());
    }
    if (this.app.el.previewCloseButton) {
      this.app.el.previewCloseButton.addEventListener("click", () => this.clearMovePreview({ restore: true }));
    }
    if (this.app.el.gameModeSelect) {
      this.app.el.gameModeSelect.addEventListener("change", this.updateSetupControlState.bind(this));
    }
    if (this.app.el.fenInput) {
      this.app.el.fenInput.addEventListener("input", this.updateSetupControlState.bind(this));
    }
    if (this.app.el.clockEnabledInput) {
      this.app.el.clockEnabledInput.addEventListener("change", this.updateSetupControlState.bind(this));
    }
    if (this.app.el.clockPresetSelect) {
      this.app.el.clockPresetSelect.addEventListener("change", () => {
        this.app.clocks.applyClockPresetToInputs();
      });
    }
    if (this.app.el.clockBaseSecInput) {
      this.app.el.clockBaseSecInput.addEventListener("input", this.markClockPresetCustom.bind(this));
    }
    if (this.app.el.clockIncrementSecInput) {
      this.app.el.clockIncrementSecInput.addEventListener("input", this.markClockPresetCustom.bind(this));
    }
    if (this.app.el.clockHumanBaseSecInput) {
      this.app.el.clockHumanBaseSecInput.addEventListener("input", this.markClockPresetCustom.bind(this));
    }
    if (this.app.el.clockAiBaseSecInput) {
      this.app.el.clockAiBaseSecInput.addEventListener("input", this.markClockPresetCustom.bind(this));
    }
    if (this.app.el.aiStrengthSelect) {
      this.app.el.aiStrengthSelect.addEventListener("change", () => {
        this.updateSetupControlState();
        void this.syncSetupToSession();
      });
    }
    if (this.app.el.coachLevelSelect) {
      this.app.el.coachLevelSelect.addEventListener("change", () => {
        void this.syncSetupToSession();
      });
    }
    if (this.app.el.gameTypeSelect) {
      this.app.el.gameTypeSelect.addEventListener("change", () => {
        this.app.board.previewBoardForGameType(this.app.el.gameTypeSelect.value);
        this.updateSetupControlState();
      });
    }
  }

  // syncChessNnueOption - NNUE strength is Chess-only; Xiangqi/Shogi fall back to master
  syncChessNnueOption() {
    const sel = this.app.el.aiStrengthSelect;
    if (!sel) return;
    const chess = String(this.app.el.gameTypeSelect?.value || "chess") === "chess";
    const nnue = sel.querySelector('option[value="nnue"]');
    if (nnue) {
      nnue.hidden = !chess;
      nnue.disabled = !chess;
    }
    if (!chess && sel.value === "nnue") sel.value = "master";
  }

  // updateSetupControlState - enables or disables setup controls for mode and simulation busy state
  updateSetupControlState() {
    const mode = String(this.app.el.gameModeSelect?.value || "human_vs_human");
    const isAIVsAI = mode === "ai_vs_ai";
    const isHvAI = mode === "human_vs_ai";
    const simulationBusy = this.app.state.simulationRequestInFlight || this.app.state.isSimulationPlayback;
    const fenProvided = Boolean(String(this.app.el.fenInput?.value || "").trim());
    if (this.app.el.humanSideSelect) this.app.el.humanSideSelect.disabled = isAIVsAI || simulationBusy;
    if (this.app.el.aiGameCountInput) {
      this.app.el.aiGameCountInput.disabled = !isAIVsAI || simulationBusy;
      if (fenProvided) this.app.el.aiGameCountInput.value = "1";
    }
    if (this.app.el.aiStrengthSelect) {
      this.app.el.aiStrengthSelect.disabled = !(mode === "human_vs_ai" || isAIVsAI) || simulationBusy;
      this.syncChessNnueOption();
    }
    if (this.app.el.gameModeSelect) this.app.el.gameModeSelect.disabled = simulationBusy;
    if (this.app.el.gameTypeSelect) this.app.el.gameTypeSelect.disabled = simulationBusy;
    if (this.app.el.fenInput) this.app.el.fenInput.disabled = simulationBusy;
    if (this.app.el.configApplyButton) this.app.el.configApplyButton.disabled = simulationBusy;
    if (this.app.el.newGameButton) this.app.el.newGameButton.disabled = simulationBusy;
    if (this.app.el.input) this.app.el.input.disabled = simulationBusy || this.app.state.gameOver;
    if (this.app.el.button) {
      this.app.el.button.disabled =
        simulationBusy || this.app.state.gameOver || this.app.state.previewMode;
    }
    if (this.app.el.previewButton) {
      this.app.el.previewButton.disabled =
        simulationBusy || this.app.state.gameOver || this.app.state.isPreviewing;
      this.app.el.previewButton.textContent = this.app.state.previewMode ? "Exit preview mode" : "Preview mode";
    }
    if (this.app.el.previewCloseButton) {
      this.app.el.previewCloseButton.disabled = simulationBusy;
      this.app.el.previewCloseButton.hidden = !this.app.state.previewMode;
      this.app.el.previewCloseButton.textContent = "Resume game";
    }
    if (this.app.el.flagButton) this.app.el.flagButton.disabled = simulationBusy || this.app.state.gameOver;

    const clockOn = Boolean(this.app.el.clockEnabledInput?.checked);
    if (this.app.el.clockEnabledInput) this.app.el.clockEnabledInput.disabled = simulationBusy;
    if (this.app.el.clockPresetSelect) this.app.el.clockPresetSelect.disabled = !clockOn || simulationBusy;
    if (this.app.el.clockIncrementSecInput) this.app.el.clockIncrementSecInput.disabled = !clockOn || simulationBusy;
    if (this.app.el.clockBaseSecInput) this.app.el.clockBaseSecInput.disabled = !clockOn || isHvAI || simulationBusy;
    if (this.app.el.clockHumanBaseSecInput) this.app.el.clockHumanBaseSecInput.disabled = !clockOn || !isHvAI || simulationBusy;
    if (this.app.el.clockAiBaseSecInput) this.app.el.clockAiBaseSecInput.disabled = !clockOn || !isHvAI || simulationBusy;
    if (this.app.el.clockHvAIFields) this.app.el.clockHvAIFields.style.display = isHvAI ? "" : "none";

    if (!isAIVsAI && !simulationBusy && this.app.state.simulationData) {
      this.app.simulation.cleanupSimulationControls();
      this.app.simulation.clearSimulationSummary();
    }
    if (this.app.state.simRunBtn) {
      this.app.state.simRunBtn.style.display = isAIVsAI ? "inline-block" : "none";
      this.app.state.simRunBtn.disabled = simulationBusy;
    }
  }

  // renderGameConfig - syncs setup controls from a game payload
  renderGameConfig(game, opts = {}) {
    if (!game) return;
    if (this.app.el.gameTypeSelect) this.app.el.gameTypeSelect.value = String(game.type || "chess");
    this.app.board.ensureBoardGeometry(game.type || this.app.el.gameTypeSelect?.value || "chess");
    const cfg = game.config;
    if (!cfg) {
      this.updateSetupControlState();
      return;
    }
    if (this.app.el.gameModeSelect) this.app.el.gameModeSelect.value = String(game.mode || "human_vs_human");
    if (this.app.el.humanSideSelect) this.app.el.humanSideSelect.value = String(cfg.humanColor || "white");
    if (this.app.el.aiGameCountInput) this.app.el.aiGameCountInput.value = String(cfg.aiGameCount || 1);
    if (this.app.el.fenInput) this.app.el.fenInput.value = String(cfg.startFen || "");
    if (this.app.el.aiStrengthSelect) {
      this.app.el.aiStrengthSelect.value = String(cfg.aiProfile || cfg.aiStrength || "intermediate");
      this.syncChessNnueOption();
    }
    if (this.app.el.coachLevelSelect) {
      const skill = String(cfg.skillLevel || "").toLowerCase();
      this.app.el.coachLevelSelect.value =
        skill === "beginner" || skill === "intermediate" || skill === "advanced"
          ? skill
          : "intermediate";
    }
    this.app.state.humanColor = String(cfg.humanColor || "white").toLowerCase();
    // syncClockSetup only after create / apply / new game — mid-game get/flag must not overwrite the form
    if (opts.syncClockSetup) this.app.clocks.syncClockControlsFromGame(game);
    this.updateSetupControlState();
  }

  // formatPreviewNotes - builds a short notes block for a what-if preview result
  formatPreviewNotes(command, suggestedMoves, explanation) {
    const lines = [
      "Preview mode — live game unchanged.",
      `What-if candidate: ${command}`,
    ];
    const coach = String(explanation || "").trim();
    if (coach) lines.push(coach);
    const moves = Array.isArray(suggestedMoves) ? suggestedMoves : [];
    const labels = [];
    for (let i = 0; i < moves.length && labels.length < 3; i++) {
      const sm = moves[i];
      const lab = String(sm?.san || sm?.uci || sm?.move || "").trim();
      if (lab) labels.push(lab);
    }
    if (labels.length) lines.push(`Suggested replies after preview: ${labels.join(", ")}`);
    return lines.join("\n");
  }

  // togglePreviewMode - enters or leaves preview mode without playing a live move
  togglePreviewMode() {
    if (this.app.state.previewMode) {
      this.clearMovePreview({ restore: true });
      return;
    }
    if (this.app.state.gameOver || this.app.state.simulationRequestInFlight || this.app.state.isSimulationPlayback) {
      this.app.util.setStatus("Cannot enter preview mode right now.", "error");
      return;
    }
    this.app.state.previewMode = true;
    this.app.state.previewShowing = false;
    this.app.state.previewActive = false;
    this.app.interaction.clearSelectedSquare();
    this.updateSetupControlState();
    this.app.util.setStatus("Preview mode — make a move on the board (not played).", "success");
  }

  // restoreLiveBoardForPreviewPick - paints the live board again so another candidate can be chosen
  restoreLiveBoardForPreviewPick() {
    const snap = this.app.state.previewRestore;
    if (!snap?.boardState || !this.app.state.previewShowing) return false;
    this.app.board.renderBoardFromState(snap.boardState);
    this.app.state.previewShowing = false;
    this.app.state.previewActive = false;
    this.app.state.lastSuggestionsText = snap.suggestionsText || "";
    this.app.state.lastThreatSummary = snap.threatSummary || "";
    this.app.state.lastExplanationText = snap.explanationText || "";
    this.app.gameInfo.renderGameInfo(snap.captured || null, snap.analysis || null);
    this.app.util.refreshNotesBox();
    this.app.gameInfo.setWinProbCaption(false);
    this.app.util.setStatus("Preview mode — pick another candidate move.", "success");
    this.updateSetupControlState();
    return true;
  }

  // clearMovePreview - ends preview mode; optionally restores committed analysis and live board
  clearMovePreview(opts = {}) {
    const restore = opts.restore !== false;
    const snap = this.app.state.previewRestore;
    const wasMode = this.app.state.previewMode || this.app.state.previewActive || Boolean(snap);
    this.app.state.previewMode = false;
    this.app.state.previewActive = false;
    this.app.state.previewShowing = false;
    this.app.state.previewRestore = null;
    this.app.state.isPreviewing = false;
    if (this.app.el.previewCloseButton) this.app.el.previewCloseButton.hidden = true;
    this.app.gameInfo.setWinProbCaption(false);
    this.updateSetupControlState();
    if (!wasMode) return;
    if (!restore) return;
    if (snap?.boardState) this.app.board.renderBoardFromState(snap.boardState);
    if (snap) {
      this.app.state.lastSuggestionsText = snap.suggestionsText || "";
      this.app.state.lastThreatSummary = snap.threatSummary || "";
      this.app.state.lastExplanationText = snap.explanationText || "";
      this.app.gameInfo.renderGameInfo(snap.captured || null, snap.analysis || null);
      this.app.util.refreshNotesBox();
    } else {
      this.app.gameInfo.renderGameInfo(null, this.app.state.cachedAnalysis || null);
      this.app.util.refreshNotesBox();
    }
    this.app.interaction.clearSelectedSquare();
    this.app.util.setStatus("Resumed live game.", "success");
  }

  // previewCommand - posts a legal command to preview-move and paints child board + estimated win%
  async previewCommand(commandText = "") {
    if (this.app.state.isSubmitting || this.app.state.isPreviewing) return false;
    if (this.app.state.simulationRequestInFlight || this.app.state.isSimulationPlayback) {
      this.app.util.setStatus("Simulation is in progress. Please wait for it to finish.", "error");
      return false;
    }
    if (this.app.state.gameOver) {
      this.app.util.setStatus("Game has ended. Refresh to start a new game.", "error");
      return false;
    }
    if (!this.app.state.previewMode) {
      this.togglePreviewMode();
    }
    const command = String(commandText || this.app.el.input.value).trim();
    if (!command) {
      this.app.util.setStatus("Preview mode — make a move on the board.", "error");
      return false;
    }
    if (!this.app.state.currentGameId) {
      this.app.util.setStatus("Missing game session. Start a new game first.", "error");
      return false;
    }

    this.app.state.isPreviewing = true;
    this.updateSetupControlState();
    try {
      if (this.app.state.previewShowing) this.restoreLiveBoardForPreviewPick();
      if (!this.app.state.previewRestore) {
        const liveSnap = await this.captureLiveBoardSnapshot();
        this.app.state.previewRestore = {
          analysis: this.app.state.cachedAnalysis,
          suggestionsText: this.app.state.lastSuggestionsText || "",
          threatSummary: this.app.state.lastThreatSummary || "",
          explanationText: this.app.state.lastExplanationText || "",
          boardState: liveSnap.state,
          captured: liveSnap.captured,
        };
      }
      const response = await fetch(
        `/api/games/${encodeURIComponent(this.app.state.currentGameId)}/preview-move`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ command }),
        }
      );
      if (!response.ok) {
        const errorMessage = await this.app.util.readErrorMessage(response, "Preview failed");
        this.app.util.setStatus(errorMessage || "Preview failed", "error");
        return false;
      }
      const result = await response.json();
      const previewAnalysis = {
        win_chance_white: result.win_chance_white,
        win_chance_black: result.win_chance_black,
        eval_cp_white: result.eval_cp_white,
        evaluation_source: result.evaluation_source,
        suggested_moves: result.suggested_moves,
        threat_summary: "",
      };
      this.app.state.previewMode = true;
      this.app.state.previewActive = true;
      this.app.state.previewShowing = true;
      if (Array.isArray(result.state) && result.state.length) {
        this.app.board.renderBoardFromState(result.state);
      }
      this.app.gameInfo.setWinProbCaption(true);
      this.app.gameInfo.renderGameInfo(result.captured || null, previewAnalysis, { previewOnly: true });
      const explanation = String(result.explanation || "").trim();
      this.app.state.lastExplanationText = explanation;
      this.app.state.lastSuggestionsText = this.formatPreviewNotes(
        result.command || command,
        result.suggested_moves,
        explanation
      );
      this.app.state.lastThreatSummary = "";
      this.app.util.refreshNotesBox();
      if (this.app.el.previewCloseButton) this.app.el.previewCloseButton.hidden = false;
      this.app.util.setStatus(
        `Preview of ${result.command || command} — Resume game to return.`,
        "success"
      );
      this.app.interaction.clearSelectedSquare();
      return true;
    } catch (error) {
      this.app.util.setCatchStatus(error);
      return false;
    } finally {
      this.app.state.isPreviewing = false;
      this.updateSetupControlState();
    }
  }

  // captureLiveBoardSnapshot - reads current live board pieces from the DOM state cache / server
  async captureLiveBoardSnapshot() {
    if (!this.app.state.currentGameId) {
      return { state: [], captured: this.app.state.cachedCapturedSummary };
    }
    try {
      const response = await fetch(`/api/games/${encodeURIComponent(this.app.state.currentGameId)}`);
      if (!response.ok) {
        return { state: [], captured: this.app.state.cachedCapturedSummary };
      }
      const result = await response.json();
      return {
        state: Array.isArray(result.state) ? result.state : [],
        captured: result.captured || this.app.state.cachedCapturedSummary,
      };
    } catch (_error) {
      return { state: [], captured: this.app.state.cachedCapturedSummary };
    }
  }

  // submitCommand - posts a uci command to the move endpoint and refreshes the ui
  async submitCommand(commandText = "") {
    if (this.app.state.isSubmitting) return false;
    if (this.app.state.simulationRequestInFlight || this.app.state.isSimulationPlayback) {
      this.app.util.setStatus("Simulation is in progress. Please wait for it to finish.", "error");
      return false;
    }
    if (this.app.state.gameOver) {
      this.app.util.setStatus("Game has ended. Refresh to start a new game.", "error");
      return false;
    }

    const command = String(commandText || this.app.el.input.value).trim();
    if (!command) {
      this.app.util.setStatus("Please enter a chess movement command.", "error");
      return false;
    }
    if (this.app.state.previewMode) {
      return this.previewCommand(command);
    }
    this.clearMovePreview({ restore: false });
    this.app.state.isSubmitting = true;
    try {
      if (!this.app.state.currentGameId) {
        this.app.util.setStatus("Missing game session. Start a new game first.", "error");
        return false;
      }
      const body = new URLSearchParams({ command });
      const response = await fetch(`/api/games/${encodeURIComponent(this.app.state.currentGameId)}/move`, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: body.toString(),
      });

      if (!response.ok) {
        const errorMessage = await this.app.util.readErrorMessage(response, "Invalid command format");
        this.app.util.setStatus(errorMessage || "Invalid command format", "error");
        this.app.el.input.focus();
        return false;
      }

      const result = await response.json();
      this.app.socket.syncGameIdFromResult(result);
      if (!result?.from || !result?.to) {
        this.app.util.setStatus("Invalid move response from server", "error");
        this.app.el.input.focus();
        return false;
      }

      this.app.el.input.value = this.app.el.input.value.trim() === command ? "" : this.app.el.input.value;
      if (!this.app.board.renderBoardFromState(result.state)) {
        this.app.util.setStatus("Missing board state in server response.", "error");
        return false;
      }
      this.app.applyGameSnapshot(result, { analysis: result.analysis });
      this.app.syncAnalysisAfterSnapshot(result);
      void this.app.coach.refreshSuggestedMoves();
      this.app.el.input.focus();
      return true;
    } catch (error) {
      this.app.util.setCatchStatus(error);
      this.app.el.input.focus();
      return false;
    } finally {
      this.app.state.isSubmitting = false;
    }
  }

  // submitBoardMove - builds a uci command from board sequences, including promotion choice
  async submitBoardMove(fromSequence, toSequence) {
    let command = this.app.board.moveCommandFromSequence(fromSequence, toSequence);
    if (!command) return false;
    if (this.app.state.boardGameType === "shogi") {
      // must-promote → auto "+"; optional zone → promote / do not promote picker
      const { must, can } = this.app.interaction.shogiPromotionFlags(toSequence);
      if (must) {
        command += "+";
      } else if (can) {
        const choice = await this.app.promotion.requestPromotionChoice("shogi");
        if (!choice) return false;
        if (choice === "+") command += "+";
      }
      return this.submitCommand(command);
    }
    if (this.app.interaction.requiresPromotion(toSequence)) {
      const promotionChoice = await this.app.promotion.requestPromotionChoice("chess");
      if (!promotionChoice) return false;
      command += promotionChoice;
    }
    return this.submitCommand(command);
  }

  // setupConfigBody - builds urlencoded setup fields from the form, including clock fields
  setupConfigBody() {
    const mode = String(this.app.el.gameModeSelect?.value || "human_vs_human");
    const fen = String(this.app.el.fenInput?.value || "").trim();
    const aiCount = fen ? "1" : String(this.app.el.aiGameCountInput?.value || "1");
    const params = new URLSearchParams({
      type: String(this.app.el.gameTypeSelect?.value || "chess"),
      mode,
      humanColor: String(this.app.el.humanSideSelect?.value || "white"),
      aiGameCount: aiCount,
      aiProfile: String(this.app.el.aiStrengthSelect?.value || "intermediate"),
      skillLevel: String(this.app.el.coachLevelSelect?.value || "intermediate"),
      fen,
    });
    return this.app.clocks.appendClockFields(params);
  }

  // syncSetupToSession - quietly posts setup so the next explain sees strength/coach without apply
  async syncSetupToSession() {
    if (!this.app.state.currentGameId || this.app.state.simulationRequestInFlight || this.app.state.isSimulationPlayback) return;
    try {
      const response = await fetch(`/api/games/${encodeURIComponent(this.app.state.currentGameId)}/config`, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: this.setupConfigBody().toString(),
      });
      if (!response.ok) return;
      const result = await response.json();
      if (result?.game) this.renderGameConfig(result.game);
    } catch (_) {
      // non-blocking
    }
  }

  // markClockPresetCustom - marks the clock preset dropdown as custom after manual edits
  markClockPresetCustom() {
    if (this.app.el.clockPresetSelect) this.app.el.clockPresetSelect.value = "custom";
  }
}

if (typeof window !== "undefined") {
  window.SetupCommand = SetupCommand;
} else {
  // self-check: preview notes name the command and list reply labels without claiming the move was played
  const setup = new SetupCommand({
    el: { button: { addEventListener() {} } },
    state: {},
  });
  const notes = setup.formatPreviewNotes("e2e4", [
    { san: "e5", uci: "e7e5" },
    { uci: "c7c5" },
  ], "Preview mode: if white played e2e4…");
  if (
    !notes.includes("Preview mode — live game unchanged") ||
    !notes.includes("What-if candidate: e2e4") ||
    !notes.includes("e5") ||
    !notes.includes("Preview mode: if white played e2e4")
  ) {
    throw new Error("formatPreviewNotes self-check failed");
  }
  console.log("setup command preview self-check ok");
}

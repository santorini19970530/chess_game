// CM3070 FP code
// 09_setup_command.js - setup form, command submit, new game, and apply handlers

// SetupCommand - owns setup controls, move submit, and session create/apply
class SetupCommand {
  constructor(app) {
    this.app = app;
    this.bindFormListeners();
    this.bindSessionButtons();
  }

  // bindFormListeners - wires setup form inputs to preview, sync, and enable/disable updates
  bindFormListeners() {
    this.app.el.button.addEventListener("click", this.submitCommand.bind(this));
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

  // bindSessionButtons - wires apply setup, flag, and new game clicks
  bindSessionButtons() {
    if (this.app.el.configApplyButton) {
      this.app.el.configApplyButton.addEventListener("click", () => {
        void this.onApplySetup();
      });
    }
    if (this.app.el.flagButton) {
      this.app.el.flagButton.addEventListener("click", () => {
        void this.onFlag();
      });
    }
    if (this.app.el.newGameButton) {
      this.app.el.newGameButton.addEventListener("click", () => {
        void this.onNewGame();
      });
    }
  }

  // onApplySetup - posts current setup form values to the session config endpoint
  async onApplySetup() {
    if (this.app.state.simulationRequestInFlight || this.app.state.isSimulationPlayback) {
      this.app.util.setStatus("Simulation is in progress. Please wait for it to finish.", "error");
      return;
    }
    try {
      if (!this.app.state.currentGameId) {
        this.app.util.setStatus("Missing game session. Start a new game first.", "error");
        return;
      }
      const body = this.setupConfigBody();
      const response = await fetch(`/api/games/${encodeURIComponent(this.app.state.currentGameId)}/config`, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: body.toString(),
      });
      if (!response.ok) {
        const errorMessage = await this.app.util.readErrorMessage(response, "Failed to apply game setup.");
        this.app.util.setStatus(errorMessage || "Failed to apply game setup.", "error");
        return;
      }
      const result = await response.json();
      this.app.socket.syncGameIdFromResult(result);
      this.renderGameConfig(result.game, { syncClockSetup: true });
      this.app.clocks.renderClocks(result.game);
      this.app.board.previewBoardForGameType(result.game?.type || this.app.el.gameTypeSelect?.value);
      if (result.game?.config?.humanColor) {
        this.app.state.humanColor = String(result.game.config.humanColor).toLowerCase();
      }
      this.app.util.setStatus("Game setup applied. Click New Game to start.", "success");
    } catch (error) {
      this.app.util.setCatchStatus(error);
    }
  }

  // onFlag - resigns/flags the current game via the flag endpoint
  async onFlag() {
    if (this.app.state.simulationRequestInFlight || this.app.state.isSimulationPlayback) {
      this.app.util.setStatus("Simulation is in progress. Please wait for it to finish.", "error");
      return;
    }
    if (this.app.state.gameOver) {
      this.app.util.setStatus("Game has ended. Start a new game.", "error");
      return;
    }
    try {
      if (!this.app.state.currentGameId) {
        this.app.util.setStatus("Missing game session. Start a new game first.", "error");
        return;
      }
      const response = await fetch(`/api/games/${encodeURIComponent(this.app.state.currentGameId)}/flag`, {
        method: "POST",
      });
      if (!response.ok) {
        const errorMessage = await this.app.util.readErrorMessage(response, "Failed to flag game.");
        this.app.util.setStatus(errorMessage || "Failed to flag game.", "error");
        return;
      }
      const result = await response.json();
      this.app.applyGameSnapshot(result, {
        analysis: null,
        clearAnalysisCache: true,
        stopAnalysis: true,
        resolvePromotion: true,
      });
      if (this.app.state.gameOver) {
        this.app.util.showGameEndedNotes(result?.game?.outcome?.message || "Game has ended (flag / resign).");
      }
    } catch (error) {
      this.app.util.setCatchStatus(error);
    }
  }

  // onNewGame - starts a new game with the current setup form values
  async onNewGame() {
    if (this.app.state.simulationRequestInFlight || this.app.state.isSimulationPlayback) {
      this.app.util.setStatus("Simulation is in progress. Please wait for it to finish.", "error");
      return;
    }
    try {
      if (!this.app.state.currentGameId) {
        this.app.util.setStatus("Missing game session. Start a new game first.", "error");
        return;
      }
      const body = this.setupConfigBody();
      const response = await fetch(`/api/games/${encodeURIComponent(this.app.state.currentGameId)}/new`, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: body.toString(),
      });
      if (!response.ok) {
        const errorMessage = await this.app.util.readErrorMessage(response, "Failed to start a new game.");
        this.app.util.setStatus(errorMessage || "Failed to start a new game.", "error");
        return;
      }
      const result = await response.json();
      // config first so geometry matches server type before placing pieces
      this.app.applyGameSnapshot(result, {
        board: true,
        syncClockSetup: true,
        analysis: null,
        clearAnalysisCache: true,
        clearCapturedCache: true,
        clearCoach: true,
        stopAnalysis: true,
        resolvePromotion: true,
        cleanupSimulation: true,
      });
      // reset win% after renderGameInfo so stale analysis cannot repaint old bars
      this.app.resetWinProbBars();
      this.app.el.input.value = "";
      this.app.enablePlayInputs();
      this.app.util.setStatus("New game started.", "success");
      this.app.el.input.focus();
    } catch (error) {
      this.app.util.setCatchStatus(error);
    }
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
    }
    if (this.app.el.gameModeSelect) this.app.el.gameModeSelect.disabled = simulationBusy;
    if (this.app.el.gameTypeSelect) this.app.el.gameTypeSelect.disabled = simulationBusy;
    if (this.app.el.fenInput) this.app.el.fenInput.disabled = simulationBusy;
    if (this.app.el.configApplyButton) this.app.el.configApplyButton.disabled = simulationBusy;
    if (this.app.el.newGameButton) this.app.el.newGameButton.disabled = simulationBusy;
    if (this.app.el.input) this.app.el.input.disabled = simulationBusy || this.app.state.gameOver;
    if (this.app.el.button) this.app.el.button.disabled = simulationBusy || this.app.state.gameOver;
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
        const choice = await this.app.interaction.requestPromotionChoice("shogi");
        if (!choice) return false;
        if (choice === "+") command += "+";
      }
      return this.submitCommand(command);
    }
    if (this.app.interaction.requiresPromotion(toSequence)) {
      const promotionChoice = await this.app.interaction.requestPromotionChoice("chess");
      if (!promotionChoice) return false;
      command += promotionChoice;
    }
    return this.submitCommand(command);
  }

  // createSessionOnLoad - creates the initial game session when the page loads
  async createSessionOnLoad() {
    const body = this.setupConfigBody();
    try {
      this.app.util.setStatus("Creating game session...", "success");
      const response = await fetch("/api/games", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: body.toString(),
      });
      if (!response.ok) {
        const errorMessage = await this.app.util.readErrorMessage(response, "Failed to create game session.");
        this.app.util.setStatus(errorMessage, "error");
        return;
      }
      const result = await response.json();
      this.app.applyGameSnapshot(result, {
        board: true,
        syncClockSetup: true,
        clearAnalysisCache: true,
        clearCoach: true,
        stopAnalysis: true,
      });
      void this.app.coach.refreshSuggestedMoves();
      this.app.enablePlayInputs();
      this.app.util.setStatus("Game session ready.", "success");
      this.app.el.input.focus();
    } catch (error) {
      this.app.util.setCatchStatus(error);
    }
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

window.SetupCommand = SetupCommand;

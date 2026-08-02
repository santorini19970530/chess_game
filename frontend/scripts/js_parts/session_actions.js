// CM3070 FP code
// session_actions.js - apply setup, new game, flag, and session create for the puzzle page
// SessionActions - owns session lifecycle buttons and create-on-load
class SessionActions {
  constructor(app) {
    this.app = app;
    this.bindSessionButtons();
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
      const body = this.app.setup.setupConfigBody();
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
      this.app.setup.renderGameConfig(result.game, { syncClockSetup: true });
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
      const body = this.app.setup.setupConfigBody();
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

  // createSessionOnLoad - creates the initial game session when the page loads
  async createSessionOnLoad() {
    const body = this.app.setup.setupConfigBody();
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
}

window.SessionActions = SessionActions;

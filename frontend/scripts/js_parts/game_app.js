// CM3070 FP code
// game_app.js - owns the puzzle page controllers and shared application state

// GameApp - owns puzzle page controllers, shared el/state, and snapshot paint
class GameApp {
  constructor() {
    this.el = {};
    this.state = {};
    this.ready = true;
    this.util = null;
    this.socket = null;
    this.clocks = null;
    this.board = null;
    this.interaction = null;
    this.coach = null;
    this.gameInfo = null;
    this.moveHistory = null;
    this.setup = null;
    this.review = null;
    this.diagram = null;
    this.simulation = null;
  }

  // applyGameSnapshot - paints setup/board/info/clocks from an api game result
  applyGameSnapshot(result, opts = {}) {
    this.socket.syncGameIdFromResult(result);
    if (opts.config !== false) {
      this.setup.renderGameConfig(result.game, { syncClockSetup: Boolean(opts.syncClockSetup) });
    }
    if (opts.board) {
      const ok = this.board.renderBoardFromState(result.state, result.game?.type);
      if (opts.requireBoard && !ok) return false;
    }
    this.moveHistory.renderMoveHistory(result.history, result.historyDetailed);
    this.gameInfo.renderCurrentTurn(result.currentTurn);
    this.gameInfo.renderCheckState(result.checkedSide || result?.game?.outcome?.checkedSide);
    this.gameInfo.renderGameOutcome(result.game);
    this.clocks.renderClocks(result.game);
    if (opts.syncHumanColor !== false && result.game?.config?.humanColor) {
      this.state.humanColor = String(result.game.config.humanColor).toLowerCase();
    }
    if (opts.clearAnalysisCache) this.state.cachedAnalysis = null;
    if (opts.clearCapturedCache) this.state.cachedCapturedSummary = null;
    if (opts.clearCoach) this.util.clearCoachNotesState();
    if (opts.gameInfo !== false) {
      const analysis = Object.prototype.hasOwnProperty.call(opts, "analysis")
        ? opts.analysis
        : this.state.gameOver
          ? null
          : result.analysis;
      this.gameInfo.renderGameInfo(result.captured, analysis);
    }
    if (opts.stopAnalysis) this.socket.stopAnalysisPolling();
    if (opts.resolvePromotion) this.promotion.resolvePromotionChoice("");
    if (opts.clearSelection !== false) this.interaction.clearSelectedSquare();
    if (opts.cleanupSimulation) this.simulation.cleanupSimulationControls();
    return true;
  }

  // syncAnalysisAfterSnapshot - stops or starts analysis polling from a snapshot result
  syncAnalysisAfterSnapshot(result) {
    if (this.state.gameOver) {
      this.socket.stopAnalysisPolling();
      return;
    }
    const historyArray = Array.isArray(result.history) ? result.history : [];
    const detailedArray = Array.isArray(result.historyDetailed) ? result.historyDetailed : [];
    if (result.analysis) {
      this.socket.stopAnalysisPolling();
      return;
    }
    const targetMoveNumber = Math.max(historyArray.length, detailedArray.length);
    if (targetMoveNumber > 0) this.coach.startAnalysisPolling(targetMoveNumber, result.captured);
  }

  // resetWinProbBars - resets win% labels and bars to a neutral 50/50
  resetWinProbBars() {
    if (this.el.winProbWhiteValue) this.el.winProbWhiteValue.textContent = "50.0%";
    if (this.el.winProbBlackValue) this.el.winProbBlackValue.textContent = "50.0%";
    if (this.el.winProbWhiteBar) this.el.winProbWhiteBar.style.width = "50%";
    if (this.el.winProbBlackBar) this.el.winProbBlackBar.style.width = "50%";
  }

  // enablePlayInputs - re-enables command and flag controls for a live game
  enablePlayInputs() {
    this.el.input.disabled = false;
    this.el.button.disabled = false;
    if (this.el.flagButton) this.el.flagButton.disabled = false;
    this.state.gameOver = false;
  }
}

window.GameApp = GameApp;

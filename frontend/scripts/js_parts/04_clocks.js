// CM3070 FP code
// 04_clocks.js - shared clock display, ticking, and presets for the puzzle page
// ClockController - owns clock paint, tick, flag timeout, and presets
class ClockController {
  constructor(app) {
    this.app = app;
    this.app.CLOCK_PRESETS = {
      "5|0": { baseSec: 300, incrementSec: 0 },
      "10|0": { baseSec: 600, incrementSec: 0 },
      "15|10": { baseSec: 900, incrementSec: 10 },
      "5|30": { baseSec: 300, incrementSec: 30 },
    };
  }

  // formatClockMs - formats a millisecond clock value as m:ss
  formatClockMs(ms) {
    const totalSec = Math.max(0, Math.floor(Number(ms) / 1000) || 0);
    const minutes = Math.floor(totalSec / 60);
    const seconds = totalSec % 60;
    return `${minutes}:${String(seconds).padStart(2, "0")}`;
  }

  // stopClockTick - clears the local clock tick interval
  stopClockTick() {
    if (this.app.state.clockTickTimer != null) {
      window.clearInterval(this.app.state.clockTickTimer);
      this.app.state.clockTickTimer = null;
    }
    this.app.state.clockLastTickAt = 0;
  }

  // paintClockLabels - paints white/black clock labels from local state
  paintClockLabels() {
    if (!this.app.el.timeWhiteValue || !this.app.el.timeBlackValue) return;
    if (!this.app.state.clockEnabledLocal) {
      this.app.el.timeWhiteValue.textContent = "⏱ --:--";
      this.app.el.timeBlackValue.textContent = "⏱ --:--";
      return;
    }
    this.app.el.timeWhiteValue.textContent = `⏱ ${this.formatClockMs(this.app.state.clockWhiteMs)}`;
    this.app.el.timeBlackValue.textContent = `⏱ ${this.formatClockMs(this.app.state.clockBlackMs)}`;
  }

  // startClockTick - starts the local countdown tick while clocks are active
  startClockTick() {
    if (
      !this.app.state.clockEnabledLocal ||
      this.app.state.gameOver ||
      this.app.state.simulationRequestInFlight ||
      this.app.state.isSimulationPlayback
    ) {
      this.stopClockTick();
      return;
    }
    if (this.app.state.clockTickTimer != null) return;
    this.app.state.clockLastTickAt = Date.now();
    this.app.state.clockTickTimer = window.setInterval(() => {
      if (!this.app.state.clockEnabledLocal || this.app.state.gameOver) {
        this.stopClockTick();
        return;
      }
      const now = Date.now();
      const elapsed = now - this.app.state.clockLastTickAt;
      this.app.state.clockLastTickAt = now;
      if (elapsed > 0) {
        if (this.app.state.clockActiveSide === "black") {
          this.app.state.clockBlackMs = Math.max(0, this.app.state.clockBlackMs - elapsed);
        } else {
          this.app.state.clockWhiteMs = Math.max(0, this.app.state.clockWhiteMs - elapsed);
        }
      }
      this.paintClockLabels();
      const remaining =
        this.app.state.clockActiveSide === "black" ? this.app.state.clockBlackMs : this.app.state.clockWhiteMs;
      if (remaining <= 0) {
        this.stopClockTick();
        void this.flagOnLocalTimeout();
      }
    }, 250);
  }

  // applyServerClock - applies server clock payload into local ticking state
  applyServerClock(clk, remaining) {
    if (!clk || !clk.enabled) {
      this.app.state.clockEnabledLocal = false;
      this.app.state.clockFlagInFlight = false;
      this.stopClockTick();
      this.paintClockLabels();
      return;
    }
    this.app.state.clockEnabledLocal = true;
    if (remaining && remaining.white != null && remaining.black != null) {
      this.app.state.clockWhiteMs = Math.max(0, Number(remaining.white) || 0);
      this.app.state.clockBlackMs = Math.max(0, Number(remaining.black) || 0);
    } else {
      this.app.state.clockWhiteMs = Math.max(0, Number(clk.whiteRemainingMs) || 0);
      this.app.state.clockBlackMs = Math.max(0, Number(clk.blackRemainingMs) || 0);
    }
    const active = String(clk.active || this.app.state.currentTurn || "white").toLowerCase();
    this.app.state.clockActiveSide = active === "black" ? "black" : "white";
    this.app.state.clockFlagInFlight = false;
    this.app.state.clockLastTickAt = Date.now();
    this.paintClockLabels();
    if (!this.app.state.gameOver) this.startClockTick();
    else this.stopClockTick();
  }

  // renderClocks - renders clocks from a game payload
  renderClocks(game) {
    this.applyServerClock(game?.clock, null);
  }

  // flagOnLocalTimeout - flags the game when local remaining time hits zero
  async flagOnLocalTimeout() {
    if (this.app.state.clockFlagInFlight || this.app.state.gameOver || !this.app.state.currentGameId) return;
    if (this.app.state.simulationRequestInFlight || this.app.state.isSimulationPlayback) return;
    this.app.state.clockFlagInFlight = true;
    try {
      const response = await fetch(`/api/games/${encodeURIComponent(this.app.state.currentGameId)}/flag`, {
        method: "POST",
      });
      if (!response.ok) {
        this.app.state.clockFlagInFlight = false;
        void this.app.socket.refreshGameSnapshotFromAPI(this.app.state.currentGameId);
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
    } catch (_) {
      this.app.state.clockFlagInFlight = false;
      void this.app.socket.refreshGameSnapshotFromAPI(this.app.state.currentGameId);
    }
  }

  // applyClockPresetToInputs - copies a named clock preset into the setup inputs
  applyClockPresetToInputs() {
    const key = String(this.app.el.clockPresetSelect?.value || "");
    const preset = this.app.CLOCK_PRESETS[key];
    if (!preset) return;
    if (this.app.el.clockBaseSecInput) this.app.el.clockBaseSecInput.value = String(preset.baseSec);
    if (this.app.el.clockIncrementSecInput) this.app.el.clockIncrementSecInput.value = String(preset.incrementSec);
    if (this.app.el.clockHumanBaseSecInput) this.app.el.clockHumanBaseSecInput.value = String(preset.baseSec);
  }

  // appendClockFields - appends clock form fields onto a urlencoded params object
  appendClockFields(params) {
    const enabled = Boolean(this.app.el.clockEnabledInput?.checked);
    params.set("clockEnabled", enabled ? "true" : "false");
    if (!enabled) return params;
    const mode = String(this.app.el.gameModeSelect?.value || "human_vs_human");
    const incMs = String(Math.max(0, Math.round(Number(this.app.el.clockIncrementSecInput?.value || 0) * 1000)));
    params.set("incrementMs", incMs);
    if (mode === "human_vs_ai") {
      params.set(
        "humanInitialMs",
        String(Math.max(0, Math.round(Number(this.app.el.clockHumanBaseSecInput?.value || 0) * 1000)))
      );
      params.set(
        "aiInitialMs",
        String(Math.max(0, Math.round(Number(this.app.el.clockAiBaseSecInput?.value || 0) * 1000)))
      );
      return params;
    }
    const baseMs = String(Math.max(0, Math.round(Number(this.app.el.clockBaseSecInput?.value || 0) * 1000)));
    params.set("whiteInitialMs", baseMs);
    params.set("blackInitialMs", baseMs);
    return params;
  }

  // syncClockControlsFromGame - syncs clock setup inputs from a game payload
  syncClockControlsFromGame(game) {
    const clk = game?.clock;
    if (!this.app.el.clockEnabledInput || !clk) return;
    this.app.el.clockEnabledInput.checked = Boolean(clk.enabled);
    if (!clk.enabled) return;
    const whiteSec = Math.max(0, Math.round(Number(clk.whiteInitialMs || 0) / 1000));
    const blackSec = Math.max(0, Math.round(Number(clk.blackInitialMs || 0) / 1000));
    const incSec = Math.max(0, Math.round(Number(clk.incrementMs || 0) / 1000));
    if (this.app.el.clockIncrementSecInput) this.app.el.clockIncrementSecInput.value = String(incSec);
    if (this.app.el.clockBaseSecInput) this.app.el.clockBaseSecInput.value = String(whiteSec);
    const side = String(game?.config?.humanColor || this.app.state.humanColor || "white").toLowerCase();
    if (side === "black") {
      if (this.app.el.clockHumanBaseSecInput) this.app.el.clockHumanBaseSecInput.value = String(blackSec);
      if (this.app.el.clockAiBaseSecInput) this.app.el.clockAiBaseSecInput.value = String(whiteSec);
    } else {
      if (this.app.el.clockHumanBaseSecInput) this.app.el.clockHumanBaseSecInput.value = String(whiteSec);
      if (this.app.el.clockAiBaseSecInput) this.app.el.clockAiBaseSecInput.value = String(blackSec);
    }
    const matched = Object.entries(this.app.CLOCK_PRESETS).find(
      ([, preset]) => preset.baseSec === whiteSec && preset.incrementSec === incSec
    );
    if (this.app.el.clockPresetSelect) this.app.el.clockPresetSelect.value = matched ? matched[0] : "custom";
  }
}

window.ClockController = ClockController;

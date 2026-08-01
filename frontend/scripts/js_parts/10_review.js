// CM3070 FP code
// 10_review.js - load-moves review playback for the puzzle page
// ReviewPlayback - owns review load, seek, and playback controls
class ReviewPlayback {
  constructor(app) {
    this.app = app;
    this.app.state.reviewPlaybackMoves = null;
    this.app.state.reviewPlaybackPly = 0;
    this.app.state.reviewPlaybackBusy = false;

    if (this.app.el.reviewMovesFile && this.app.el.reviewMovesInput) {
      this.app.el.reviewMovesFile.addEventListener("change", async () => {
        const file = this.app.el.reviewMovesFile.files && this.app.el.reviewMovesFile.files[0];
        if (!file) return;
        try {
          this.app.el.reviewMovesInput.value = await file.text();
          this.app.util.setStatus(`Loaded file ${file.name} into review box.`, "success");
        } catch (error) {
          this.app.util.setCatchStatus(error);
        }
      });
    }

    if (this.app.el.reviewMovesLoad) {
      this.app.el.reviewMovesLoad.addEventListener("click", async () => {
        if (this.app.state.simulationRequestInFlight || this.app.state.isSimulationPlayback) {
          this.app.util.setStatus("Simulation is in progress. Please wait for it to finish.", "error");
          return;
        }
        const raw = String(this.app.el.reviewMovesInput?.value || "").trim();
        if (!raw) {
          this.app.util.setStatus("Paste UCI moves or a game JSON first.", "error");
          return;
        }
        if (!this.app.state.currentGameId) {
          this.app.util.setStatus("Missing game session. Start a new game first.", "error");
          return;
        }
        if (this.app.state.reviewPlaybackBusy) return;
        this.app.state.reviewPlaybackBusy = true;
        this.updateReviewPlaybackControls();
        try {
          this.app.util.setStatus("Loading moves…", "success");
          const result = await this.postLoadMovesRaw(raw);
          const moves = this.uciListFromSnapshot(result);
          if (!moves.length) {
            this.app.util.setStatus("Load succeeded but no moves were found in the response.", "error");
            return;
          }
          this.app.state.reviewPlaybackMoves = moves;
          this.app.state.reviewPlaybackPly = moves.length;
          this.applyLoadedGameSnapshot(result);
          this.app.util.setStatus(`Loaded ${moves.length} move(s) for review. Use Back / Forward to step.`, "success");
          this.app.el.input.focus();
        } catch (error) {
          this.app.util.setStatus(error?.message || "Failed to load moves.", "error");
        } finally {
          this.app.state.reviewPlaybackBusy = false;
          this.updateReviewPlaybackControls();
        }
      });
    }

    if (this.app.el.reviewMovesPrev) {
      this.app.el.reviewMovesPrev.addEventListener("click", () => {
        if (this.app.state.simulationRequestInFlight || this.app.state.isSimulationPlayback) {
          this.app.util.setStatus("Simulation is in progress. Please wait for it to finish.", "error");
          return;
        }
        void this.seekReviewPlayback(this.app.state.reviewPlaybackPly - 1);
      });
    }

    if (this.app.el.reviewMovesNext) {
      this.app.el.reviewMovesNext.addEventListener("click", () => {
        if (this.app.state.simulationRequestInFlight || this.app.state.isSimulationPlayback) {
          this.app.util.setStatus("Simulation is in progress. Please wait for it to finish.", "error");
          return;
        }
        void this.seekReviewPlayback(this.app.state.reviewPlaybackPly + 1);
      });
    }

    this.updateReviewPlaybackControls();
  }

  // uciListFromSnapshot - extracts a uci move list from a loaded game snapshot
  uciListFromSnapshot(result) {
    const detailed = Array.isArray(result?.historyDetailed) ? result.historyDetailed : [];
    const fromDetailed = detailed
      .map((entry) =>
        String(entry?.command || "")
          .trim()
          .toLowerCase()
      )
      .filter(Boolean);
    if (fromDetailed.length) return fromDetailed;
    const history = Array.isArray(result?.history) ? result.history : [];
    return history
      .map((line) => {
        const text = String(line || "");
        const idx = text.indexOf(":");
        return (idx >= 0 ? text.slice(idx + 1) : text).trim().toLowerCase();
      })
      .filter(Boolean);
  }

  // updateReviewPlaybackControls - enables or disables review ply controls and labels
  updateReviewPlaybackControls() {
    const total = Array.isArray(this.app.state.reviewPlaybackMoves) ? this.app.state.reviewPlaybackMoves.length : 0;
    const ply = total ? this.app.state.reviewPlaybackPly : 0;
    if (this.app.el.reviewMovesPlyLabel) {
      this.app.el.reviewMovesPlyLabel.textContent = total ? `Ply ${ply} / ${total}` : "Ply 0 / 0";
    }
    if (this.app.el.reviewMovesPrev) {
      this.app.el.reviewMovesPrev.disabled = !total || ply <= 0 || this.app.state.reviewPlaybackBusy;
    }
    if (this.app.el.reviewMovesNext) {
      this.app.el.reviewMovesNext.disabled = !total || ply >= total || this.app.state.reviewPlaybackBusy;
    }
  }

  // applyLoadedGameSnapshot - applies a load-moves snapshot onto board, clocks, and coach state
  applyLoadedGameSnapshot(result) {
    this.app.applyGameSnapshot(result, {
      board: true,
      syncClockSetup: true,
      analysis: null,
      clearAnalysisCache: true,
      clearCapturedCache: true,
      stopAnalysis: true,
      resolvePromotion: true,
      cleanupSimulation: true,
    });
    this.app.el.input.value = "";
    this.updateReviewPlaybackControls();

    if (this.app.state.gameOver) {
      // renderGameOutcome already set end notes / disabled inputs.
      return;
    }

    this.app.util.clearCoachNotesState();
    const historyArray = Array.isArray(result.history) ? result.history : [];
    const detailedArray = Array.isArray(result.historyDetailed) ? result.historyDetailed : [];
    const targetMoveNumber = Math.max(historyArray.length, detailedArray.length);
    if (targetMoveNumber > 0) {
      this.app.state.lastExplanationText = "[coach] Thinking…";
      this.app.util.refreshNotesBox();
      void this.app.coach.refreshSuggestedMoves();
      this.app.coach.startAnalysisPolling(targetMoveNumber, result.captured);
    } else {
      this.app.util.clearCoachNotesState();
      this.app.util.refreshNotesBox();
      void this.app.coach.refreshSuggestedMoves();
    }
  }

  // postLoadMovesRaw - posts raw review text to the load-moves endpoint
  async postLoadMovesRaw(raw) {
    const response = await fetch(`/api/games/${encodeURIComponent(this.app.state.currentGameId)}/load-moves`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ raw }),
    });
    if (!response.ok) {
      const errorMessage = await this.app.util.readErrorMessage(response, "Failed to load moves.");
      throw new Error(errorMessage || "Failed to load moves.");
    }
    return response.json();
  }

  // seekReviewPlayback - seeks review playback to a target ply by reloading moves
  async seekReviewPlayback(targetPly) {
    if (!Array.isArray(this.app.state.reviewPlaybackMoves) || !this.app.state.reviewPlaybackMoves.length) {
      this.app.util.setStatus("Load moves first to enable playback.", "error");
      return;
    }
    if (!this.app.state.currentGameId) {
      this.app.util.setStatus("Missing game session. Start a new game first.", "error");
      return;
    }
    const total = this.app.state.reviewPlaybackMoves.length;
    const ply = Math.max(0, Math.min(total, Number(targetPly) || 0));
    if (this.app.state.reviewPlaybackBusy) return;
    this.app.state.reviewPlaybackBusy = true;
    this.updateReviewPlaybackControls();
    try {
      const raw = ply <= 0 ? "" : this.app.state.reviewPlaybackMoves.slice(0, ply).join(" ");
      this.app.util.setStatus(ply <= 0 ? "Review: start position…" : `Review: ply ${ply} / ${total}…`, "success");
      const result = await this.postLoadMovesRaw(raw);
      this.app.state.reviewPlaybackPly = ply;
      this.applyLoadedGameSnapshot(result);
      this.app.util.setStatus(ply <= 0 ? "Review at start position." : `Review at ply ${ply} / ${total}.`, "success");
    } catch (error) {
      this.app.util.setStatus(error?.message || "Review seek failed.", "error");
    } finally {
      this.app.state.reviewPlaybackBusy = false;
      this.updateReviewPlaybackControls();
    }
  }
}

window.ReviewPlayback = ReviewPlayback;

// CM3070 FP code
// diagram_import.js - upload board diagram, confirm fen, load session + coach

// DiagramImport - owns diagram recognize / confirm / cancel flow
class DiagramImport {
  constructor(app) {
    this.app = app;
    this.app.state.diagramPending = null;
    this.app.state.diagramBusy = false;

    if (this.app.el.diagramImportRecognize) {
      this.app.el.diagramImportRecognize.addEventListener("click", () => {
        void this.recognizeSelectedFile();
      });
    }
    if (this.app.el.diagramImportFile) {
      this.app.el.diagramImportFile.addEventListener("change", () => {
        this.clearPending({ keepFile: true });
      });
    }
    if (this.app.el.diagramImportConfirmBtn) {
      this.app.el.diagramImportConfirmBtn.addEventListener("click", () => {
        void this.confirmLoad();
      });
    }
    if (this.app.el.diagramImportCancelBtn) {
      this.app.el.diagramImportCancelBtn.addEventListener("click", () => {
        this.clearPending({ keepFile: false });
        this.app.util.setStatus("Diagram import cancelled.", "success");
      });
    }
  }

  // selectedGameType - returns the setup game type for diagram recognition
  selectedGameType() {
    return String(this.app.el.gameTypeSelect?.value || "chess")
      .trim()
      .toLowerCase() || "chess";
  }

  // clearPending - hides confirm panel and clears pending fen state
  clearPending(opts = {}) {
    this.app.state.diagramPending = null;
    if (this.app.el.diagramImportConfirm) this.app.el.diagramImportConfirm.hidden = true;
    if (this.app.el.diagramImportFen) this.app.el.diagramImportFen.value = "";
    if (this.app.el.diagramImportPreview) this.app.el.diagramImportPreview.innerHTML = "";
    if (this.app.el.diagramImportNote) this.app.el.diagramImportNote.textContent = "";
    if (!opts.keepFile && this.app.el.diagramImportFile) this.app.el.diagramImportFile.value = "";
  }

  // setBusy - toggles recognize/confirm controls while a request is in flight
  setBusy(busy) {
    this.app.state.diagramBusy = Boolean(busy);
    if (this.app.el.diagramImportRecognize) this.app.el.diagramImportRecognize.disabled = busy;
    if (this.app.el.diagramImportConfirmBtn) this.app.el.diagramImportConfirmBtn.disabled = busy;
    if (this.app.el.diagramImportCancelBtn) this.app.el.diagramImportCancelBtn.disabled = busy;
  }

  // recognizeSelectedFile - posts the chosen image to /api/diagram/fen and opens confirm
  async recognizeSelectedFile() {
    if (this.app.state.diagramBusy) return;
    if (this.app.state.simulationRequestInFlight || this.app.state.isSimulationPlayback) {
      this.app.util.setStatus("Simulation is in progress. Please wait for it to finish.", "error");
      return;
    }
    const file = this.app.el.diagramImportFile?.files && this.app.el.diagramImportFile.files[0];
    if (!file) {
      this.app.util.setStatus("Choose a board diagram image first.", "error");
      return;
    }
    if (!this.app.state.currentGameId) {
      this.app.util.setStatus("Missing game session. Start a new game first.", "error");
      return;
    }

    const game = this.selectedGameType();
    this.setBusy(true);
    try {
      this.app.util.setStatus("Recognizing diagram…", "success");
      const form = new FormData();
      form.append("image", file, file.name || "diagram.png");
      form.append("game", game);
      const response = await fetch("/api/diagram/fen", { method: "POST", body: form });
      if (!response.ok) {
        const errorMessage = await this.app.util.readErrorMessage(response, "Diagram recognition failed.");
        throw new Error(errorMessage || "Diagram recognition failed.");
      }
      const payload = await response.json();
      const fen = String(payload?.fen || "").trim();
      if (!fen) throw new Error("Recognizer returned an empty FEN.");
      const pendingGame = String(payload?.game || game).trim().toLowerCase() || game;
      this.app.state.diagramPending = {
        fen,
        game: pendingGame === "xiangqi" ? "xianqi" : pendingGame,
        limitsNote: String(payload?.limits_note || "").trim(),
      };
      this.showConfirmPanel(this.app.state.diagramPending);
      this.app.util.setStatus("Diagram recognized. Confirm to load, or Cancel.", "success");
    } catch (error) {
      this.clearPending({ keepFile: true });
      this.app.util.setStatus(error?.message || "Diagram recognition failed.", "error");
    } finally {
      this.setBusy(false);
    }
  }

  // showConfirmPanel - fills fen / preview / limits before the user confirms load
  showConfirmPanel(pending) {
    if (!this.app.el.diagramImportConfirm) return;
    this.app.el.diagramImportConfirm.hidden = false;
    if (this.app.el.diagramImportFen) this.app.el.diagramImportFen.value = pending.fen;
    if (this.app.el.diagramImportPreview) {
      this.app.el.diagramImportPreview.innerHTML = this.buildPreviewHtml(pending.fen, pending.game);
    }
    if (this.app.el.diagramImportNote) {
      const notes = [];
      if (pending.limitsNote) notes.push(pending.limitsNote);
      if (pending.game === "shogi") {
        notes.push(
          "Shogi: hands are inferred from board inventory (heuristic; may differ from diagram komadai)."
        );
      }
      if (pending.game === "xianqi") {
        notes.push("Xiangqi: confirm the board carefully; recognition is less reliable than chess.");
      }
      this.app.el.diagramImportNote.textContent = notes.join(" ");
    }
  }

  // buildPreviewHtml - builds a small confirmation board (chess grid; fen text for variants)
  buildPreviewHtml(fen, game) {
    if (game === "chess") {
      const grid = this.chessFenPreviewGrid(fen);
      if (grid) return grid;
    }
    return `<pre class="diagram_import_fen_fallback">${this.escapeHtml(fen)}</pre>`;
  }

  // chessFenPreviewGrid - renders an 8x8 confirmation grid from a chess fen placement field
  chessFenPreviewGrid(fen) {
    const placement = String(fen || "").trim().split(/\s+/)[0] || "";
    const ranks = placement.split("/");
    if (ranks.length !== 8) return "";
    const kindByLetter = {
      p: "pawn",
      n: "knight",
      b: "bishop",
      r: "rook",
      q: "queen",
      k: "king",
    };
    let html = '<div class="diagram_preview_board" role="img" aria-label="Chess confirmation board">';
    for (const rank of ranks) {
      html += '<div class="diagram_preview_rank">';
      for (const ch of rank) {
        if (ch >= "1" && ch <= "8") {
          const n = Number(ch);
          for (let i = 0; i < n; i += 1) html += '<span class="diagram_preview_sq"></span>';
          continue;
        }
        const kind = kindByLetter[ch.toLowerCase()];
        if (!kind) return "";
        const tone = ch === ch.toUpperCase() ? "light" : "dark";
        html += `<span class="diagram_preview_sq"><img class="diagram_preview_piece" src="/pic/chess_pic/${kind}_${tone}.png" alt="${kind}" /></span>`;
      }
      html += "</div>";
    }
    html += "</div>";
    return html;
  }

  // escapeHtml - escapes fen text for the variant fallback preview
  escapeHtml(text) {
    return String(text)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  // confirmLoad - loads the pending fen into a new review session and starts coach polling
  async confirmLoad() {
    if (this.app.state.diagramBusy) return;
    const pending = this.app.state.diagramPending;
    if (!pending?.fen) {
      this.app.util.setStatus("Recognize a diagram first.", "error");
      return;
    }
    if (!this.app.state.currentGameId) {
      this.app.util.setStatus("Missing game session. Start a new game first.", "error");
      return;
    }
    if (this.app.state.simulationRequestInFlight || this.app.state.isSimulationPlayback) {
      this.app.util.setStatus("Simulation is in progress. Please wait for it to finish.", "error");
      return;
    }

    this.setBusy(true);
    try {
      this.app.util.setStatus("Loading confirmed position…", "success");
      const response = await fetch(`/api/games/${encodeURIComponent(this.app.state.currentGameId)}/load-fen`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ fen: pending.fen, game: pending.game }),
      });
      if (!response.ok) {
        const errorMessage = await this.app.util.readErrorMessage(response, "Failed to load FEN.");
        throw new Error(errorMessage || "Failed to load FEN.");
      }
      const result = await response.json();
      this.app.state.reviewPlaybackMoves = null;
      this.app.state.reviewPlaybackPly = 0;
      this.app.review.applyLoadedGameSnapshot(result, { analysisMoveNumber: 1 });
      this.clearPending({ keepFile: false });
      this.app.util.setStatus("Diagram position loaded. Coach will update for this FEN.", "success");
      this.app.el.input?.focus();
    } catch (error) {
      this.app.util.setStatus(error?.message || "Failed to load confirmed FEN.", "error");
    } finally {
      this.setBusy(false);
    }
  }
}

window.DiagramImport = DiagramImport;

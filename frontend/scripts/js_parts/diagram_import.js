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
    if (this.app.el.diagramImportFen) {
      this.app.el.diagramImportFen.addEventListener("input", () => {
        this.refreshPreviewFromFenInput();
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

  // refreshPreviewFromFenInput - rebuilds the confirm preview when the fen textarea changes
  refreshPreviewFromFenInput() {
    const pending = this.app.state.diagramPending;
    if (!pending || !this.app.el.diagramImportPreview) return;
    const fen = String(this.app.el.diagramImportFen?.value || "").trim();
    this.app.el.diagramImportPreview.innerHTML = this.buildPreviewHtml(fen || pending.fen, pending.game);
  }

  // buildPreviewHtml - builds a small confirmation board for chess / xianqi / shogi
  buildPreviewHtml(fen, game) {
    const normalized = game === "xiangqi" ? "xianqi" : game;
    if (normalized === "xianqi") {
      const pointBoard = this.xiangqiFenPreviewHtml(fen);
      if (pointBoard) return pointBoard;
    } else {
      const grid = this.variantFenPreviewGrid(fen, normalized);
      if (grid) return grid;
    }
    return `<pre class="diagram_import_fen_fallback">${this.escapeHtml(fen)}</pre>`;
  }

  // xiangqiFenPreviewHtml - renders a point-board confirm preview (lines + river + palace), not a square grid
  xiangqiFenPreviewHtml(fen) {
    const ranks = this.fenPlacementToken(fen).split("/");
    if (ranks.length !== 10) return "";
    const pieces = [];
    for (let i = 0; i < ranks.length; i += 1) {
      const rank = 10 - i;
      const cells = this.expandFenRank(ranks[i], 9, "xianqi");
      if (!cells) return "";
      for (let f = 0; f < cells.length; f += 1) {
        const cell = cells[f];
        if (!cell) continue;
        const src = this.previewPieceSrc(cell.kind, cell.color, "xianqi");
        if (!src) return "";
        pieces.push({
          file: f + 1,
          rank,
          kind: cell.kind,
          color: cell.color,
          src,
        });
      }
    }

    let art = '<div class="diagram_preview_xq_art" aria-hidden="true">';
    for (let j = 0; j <= 9; j += 1) {
      art += `<div class="diagram_preview_xq_h" style="top:${(j / 9) * 100}%"></div>`;
    }
    for (let i = 0; i <= 8; i += 1) {
      const outer = i === 0 || i === 8;
      art += `<div class="diagram_preview_xq_v${outer ? " diagram_preview_xq_v_outer" : " diagram_preview_xq_v_inner"}" style="left:${(i / 8) * 100}%"></div>`;
    }
    art += '<div class="diagram_preview_xq_palace diagram_preview_xq_palace_top"></div>';
    art += '<div class="diagram_preview_xq_palace diagram_preview_xq_palace_bottom"></div>';
    art += "</div>";

    let points = '<div class="diagram_preview_xq_points">';
    for (const p of pieces) {
      const left = ((p.file - 1) / 8) * 100;
      const top = ((10 - p.rank) / 9) * 100;
      points += `<span class="diagram_preview_xq_point" style="left:${left}%;top:${top}%"><img class="diagram_preview_xq_piece" src="${p.src}" alt="${p.kind}" data-color="${p.color}" /></span>`;
    }
    points += "</div>";

    return `<div class="diagram_preview_xq" role="img" aria-label="Xiangqi confirmation board"><div class="diagram_preview_xq_field">${art}${points}</div></div>`;
  }

  // fenPlacementToken - returns the placement field with any shogi hand bracket stripped
  fenPlacementToken(fen) {
    let placement = String(fen || "").trim().split(/\s+/)[0] || "";
    const bracket = placement.indexOf("[");
    if (bracket >= 0) placement = placement.slice(0, bracket);
    return placement;
  }

  // fenHandToken - returns shogi hand letters inside [] when present
  fenHandToken(fen) {
    const raw = String(fen || "").trim().split(/\s+/)[0] || "";
    const start = raw.indexOf("[");
    if (start < 0) return "";
    const end = raw.indexOf("]", start);
    if (end < 0) return raw.slice(start + 1);
    return raw.slice(start + 1, end);
  }

  // variantFenPreviewGrid - renders a confirmation grid (+ shogi hands) from a fen placement field
  variantFenPreviewGrid(fen, game) {
    const files = game === "chess" ? 8 : 9;
    const rankCount = game === "xianqi" ? 10 : game === "shogi" ? 9 : 8;
    const ranks = this.fenPlacementToken(fen).split("/");
    if (ranks.length !== rankCount) return "";

    let html = `<div class="diagram_preview_board" data-game="${this.escapeHtml(game)}" data-files="${files}" role="img" aria-label="${this.escapeHtml(game)} confirmation board">`;
    for (const rank of ranks) {
      html += '<div class="diagram_preview_rank">';
      const cells = this.expandFenRank(rank, files, game);
      if (!cells) return "";
      for (const cell of cells) {
        if (!cell) {
          html += '<span class="diagram_preview_sq"></span>';
          continue;
        }
        const src = this.previewPieceSrc(cell.kind, cell.color, game);
        if (!src) return "";
        const rotate =
          game === "shogi" && cell.color === "black"
            ? ' data-color="black"'
            : cell.color
              ? ` data-color="${cell.color}"`
              : "";
        html += `<span class="diagram_preview_sq"><img class="diagram_preview_piece" src="${src}" alt="${cell.kind}"${rotate} /></span>`;
      }
      html += "</div>";
    }
    html += "</div>";
    if (game === "shogi") {
      const hands = this.shogiHandPreviewHtml(this.fenHandToken(fen));
      if (hands) html += hands;
    }
    return html;
  }

  // expandFenRank - expands one fen rank into files cells ({kind,color} or null); null if width wrong
  expandFenRank(rankText, files, game) {
    const cells = [];
    let i = 0;
    const text = String(rankText || "");
    while (i < text.length) {
      const ch = text[i];
      if (ch >= "1" && ch <= "9") {
        const n = Number(ch);
        for (let k = 0; k < n; k += 1) cells.push(null);
        i += 1;
        continue;
      }
      let promoted = false;
      if (ch === "+") {
        if (game !== "shogi") return null;
        promoted = true;
        i += 1;
        if (i >= text.length) return null;
      }
      const letter = text[i];
      const kind = this.previewKindFromLetter(letter, promoted, game);
      if (!kind) return null;
      const color = letter === letter.toUpperCase() ? "white" : "black";
      cells.push({ kind, color });
      i += 1;
    }
    return cells.length === files ? cells : null;
  }

  // previewKindFromLetter - maps a fen letter (+ promoted flag) to an api piece kind
  previewKindFromLetter(letter, promoted, game) {
    const ch = String(letter || "").toLowerCase();
    if (game === "chess") {
      return { p: "pawn", n: "knight", b: "bishop", r: "rook", q: "queen", k: "king" }[ch] || "";
    }
    if (game === "xianqi") {
      return {
        r: "rook",
        n: "knight",
        h: "knight",
        b: "elephant",
        e: "elephant",
        a: "advisor",
        k: "king",
        c: "cannon",
        p: "pawn",
      }[ch] || "";
    }
    if (game === "shogi") {
      if (promoted) {
        return {
          p: "promoted_pawn",
          l: "promoted_lance",
          n: "promoted_knight",
          s: "promoted_silver",
          b: "horse",
          r: "dragon",
        }[ch] || "";
      }
      return {
        p: "pawn",
        l: "lance",
        n: "knight",
        s: "silver",
        g: "gold",
        b: "bishop",
        r: "rook",
        k: "king",
      }[ch] || "";
    }
    return "";
  }

  // previewPieceSrc - resolves a preview image url for the given game
  previewPieceSrc(kind, color, game) {
    if (game === "chess") {
      const tone = color === "black" ? "dark" : "light";
      return `/pic/chess_pic/${kind}_${tone}.png`;
    }
    if (game === "xianqi") {
      const file = {
        king: "general",
        advisor: "advisor",
        elephant: "bear",
        knight: "horse",
        rook: "chariot",
        cannon: "cannon",
        pawn: "soldier",
      }[kind];
      if (!file) return "";
      return `/pic/xianqi_pic/${file}_${color === "black" ? "black" : "white"}.png`;
    }
    if (game === "shogi") {
      return `/pic/shogi_pic/${kind}.svg`;
    }
    return "";
  }

  // parseShogiHandPieces - expands hand text (repeated letters or 2P-style counts) into {kind,color}[]
  parseShogiHandPieces(handText) {
    const kindByLetter = {
      p: "pawn",
      l: "lance",
      n: "knight",
      s: "silver",
      g: "gold",
      b: "bishop",
      r: "rook",
    };
    const out = [];
    const text = String(handText || "");
    let i = 0;
    while (i < text.length) {
      const ch = text[i];
      if (ch === " " || ch === "-") {
        i += 1;
        continue;
      }
      let count = 1;
      if (ch >= "1" && ch <= "9") {
        count = Number(ch);
        i += 1;
        if (i >= text.length) break;
      }
      const letter = text[i];
      const kind = kindByLetter[String(letter).toLowerCase()];
      if (!kind) {
        i += 1;
        continue;
      }
      const color = letter === letter.toUpperCase() ? "white" : "black";
      for (let n = 0; n < count; n += 1) out.push({ kind, color });
      i += 1;
    }
    return out;
  }

  // shogiHandPreviewHtml - renders inferred/edited hand icons under the shogi board preview
  shogiHandPreviewHtml(handText) {
    const pieces = this.parseShogiHandPieces(handText);
    if (!pieces.length) {
      return '<div class="diagram_preview_hands"><span class="diagram_preview_hand_label">Hands: (empty)</span></div>';
    }
    let html = '<div class="diagram_preview_hands" aria-label="Shogi hands preview">';
    for (const side of ["white", "black"]) {
      html += `<div class="diagram_preview_hand" data-side="${side}"><span class="diagram_preview_hand_label">${side === "white" ? "Sente" : "Gote"}</span>`;
      const sidePieces = pieces.filter((p) => p.color === side);
      if (!sidePieces.length) {
        html += '<span class="diagram_preview_hand_empty">—</span>';
      } else {
        for (const p of sidePieces) {
          html += `<img class="diagram_preview_hand_piece" src="/pic/shogi_pic/${p.kind}.svg" alt="${p.kind}" data-color="${side}" />`;
        }
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

  // confirmLoad - loads the confirmed (optionally edited) fen into a new review session
  async confirmLoad() {
    if (this.app.state.diagramBusy) return;
    const pending = this.app.state.diagramPending;
    if (!pending?.fen) {
      this.app.util.setStatus("Recognize a diagram first.", "error");
      return;
    }
    const fen = String(this.app.el.diagramImportFen?.value || pending.fen).trim();
    if (!fen) {
      this.app.util.setStatus("FEN is empty. Edit the recognized FEN or Cancel.", "error");
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
        body: JSON.stringify({ fen, game: pending.game }),
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

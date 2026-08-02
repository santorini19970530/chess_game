// CM3070 FP code
// hints_coach.js - suggested-move hints and analysis polling for the puzzle page
// HintsCoach - owns top-move hints, highlights, and analysis poll triggers
class HintsCoach {
  constructor(app) {
    this.app = app;
    this.app.state.selectedSuggestedMoves = [];
    this.app.state.lastHandHints = [];
    this.bindHintHotkeys();
  }

  // bindHintHotkeys - shows top-move hints while shift is held
  bindHintHotkeys() {
    this.app.state.hintsVisible = false;
    document.addEventListener("keydown", (e) => {
      if (e.key === "Shift" && !this.app.state.hintsVisible) {
        this.app.state.hintsVisible = true;
        void this.showTopMoves();
      }
    });
    document.addEventListener("keyup", (e) => {
      if (e.key === "Shift") this.app.state.hintsVisible = false;
    });
  }

  // showTopMoves - fetches top-move hints for the shift hotkey
  async showTopMoves() {
    if (!this.app.state.currentGameId) return;
    try {
      const res = await fetch(`/api/games/${encodeURIComponent(this.app.state.currentGameId)}/top-moves?k=3`);
      if (!res.ok) return;
      const data = await res.json();
      this.highlightSuggestedMoves(Array.isArray(data?.suggestions) ? data.suggestions : []);
    } catch (_) {}
  }

  // parseUciMove - parses a uci board move or shogi drop into from/to fields
  parseUciMove(move) {
    const raw = String(move || "")
      .trim()
      .toLowerCase();
    const board = raw.match(/^([a-i])(\d{1,2})([a-i])(\d{1,2})(\+?)$/);
    if (board) {
      return {
        from: { file: board[1].charCodeAt(0) - 96, rank: Number(board[2]) },
        to: { file: board[3].charCodeAt(0) - 96, rank: Number(board[4]) },
        dropKind: null,
        promote: board[5] === "+",
      };
    }
    const drop = raw.match(/^([plnsgbr])[*@]([a-i])([1-9])$/);
    if (drop) {
      return {
        from: null,
        to: { file: drop[2].charCodeAt(0) - 96, rank: Number(drop[3]) },
        dropKind: drop[1].toUpperCase(),
        promote: false,
      };
    }
    return null;
  }

  // squareAt - returns the board square element for a file/rank
  squareAt(file, rank) {
    if (file < 1 || file > this.app.state.boardFiles || rank < 1 || rank > this.app.state.boardMaxRank) return null;
    const sequence = this.app.board.sequenceByFileRank(file, rank);
    return this.app.el.boardElement.querySelector(`.chess_board_square[data-sequence="${sequence}"]`);
  }

  // pieceKindAt - reads the piece kind on a file/rank square
  pieceKindAt(file, rank) {
    const sq = this.squareAt(file, rank);
    const kind = sq?.querySelector(".piece_img")?.getAttribute("data-kind");
    return kind ? String(kind) : "piece";
  }

  // formatHintLine - formats one suggested-move line for the notes box
  formatHintLine(rank, parsed, scoreCp) {
    const sc = typeof scoreCp === "number" ? ` (${scoreCp > 0 ? "+" : ""}${scoreCp})` : "";
    if (!parsed) return `${rank}. ????${sc}`;
    const toLab = `${String.fromCharCode(96 + parsed.to.file)}${parsed.to.rank}`;
    if (parsed.dropKind) {
      const kind = this.app.SHOGI_DROP_KIND_FROM_CHAR[parsed.dropKind.toLowerCase()] || parsed.dropKind;
      return `${rank}. drop ${kind} from hand → ${toLab}${sc}`;
    }
    const fromLab = `${String.fromCharCode(96 + parsed.from.file)}${parsed.from.rank}`;
    const kind = this.pieceKindAt(parsed.from.file, parsed.from.rank);
    const promo = parsed.promote ? " (promote)" : "";
    return `${rank}. ${kind} ${fromLab} → ${toLab}${promo}${sc}`;
  }

  // clearSuggestedHighlights - removes suggested-move and hand-hint highlights
  clearSuggestedHighlights() {
    this.app.el.boardElement
      .querySelectorAll(
        `.${this.app.SUGGESTED_MOVE_CLASS}, .${this.app.SUGGESTED_FROM_CLASS}, .${this.app.SUGGESTED_DROP_CLASS}`
      )
      .forEach((square) => {
        square.classList.remove(
          this.app.SUGGESTED_MOVE_CLASS,
          this.app.SUGGESTED_FROM_CLASS,
          this.app.SUGGESTED_DROP_CLASS
        );
        square.removeAttribute("data-hint-rank");
      });
    document.querySelectorAll(`.${this.app.HAND_HINT_CLASS}`).forEach((el) => {
      el.classList.remove(this.app.HAND_HINT_CLASS);
      el.removeAttribute("data-hint-rank");
    });
    this.app.state.lastHandHints = [];
  }

  // appendHintRank - merges a hint rank label onto a square element
  appendHintRank(el, rankLabel) {
    if (!el) return;
    const label = String(rankLabel || "").trim();
    if (!label) return;
    const prev = el.getAttribute("data-hint-rank");
    if (!prev) {
      el.setAttribute("data-hint-rank", label);
      return;
    }
    const parts = prev
      .split(/[·,/]/)
      .map((s) => s.trim())
      .filter(Boolean);
    if (!parts.includes(label)) parts.push(label);
    parts.sort((a, b) => Number(a) - Number(b));
    el.setAttribute("data-hint-rank", parts.join("·"));
  }

  // mergeHandHintRank - merges a hint rank into the cached hand-hint list
  mergeHandHintRank(side, kind, rankLabel) {
    const hit = this.app.state.lastHandHints.find((h) => h.side === side && h.kind === kind);
    if (!hit) {
      this.app.state.lastHandHints.push({ side, kind, rank: String(rankLabel) });
      return;
    }
    const parts = String(hit.rank)
      .split(/[·,/]/)
      .map((s) => s.trim())
      .filter(Boolean);
    if (!parts.includes(String(rankLabel))) parts.push(String(rankLabel));
    parts.sort((a, b) => Number(a) - Number(b));
    hit.rank = parts.join("·");
  }

  // applyHandHintToNode - applies a cached hand hint rank onto a hand chip node
  applyHandHintToNode(node, side, kind) {
    const hit = this.app.state.lastHandHints.find((h) => h.side === side && h.kind === kind);
    if (!hit) return;
    node.classList.add(this.app.HAND_HINT_CLASS);
    node.setAttribute("data-hint-rank", hit.rank);
  }

  // refreshSuggestedMoves - fetches and paints top-move suggestions for the current position
  async refreshSuggestedMoves(retry = true) {
    if (this.app.state.isSimulationPlayback) return;
    if (!this.app.state.currentGameId || this.app.state.gameOver) {
      return;
    }
    try {
      const profile = String(this.app.el.aiStrengthSelect?.value || "intermediate");
      const url = `/api/games/${encodeURIComponent(this.app.state.currentGameId)}/top-moves?profile=${encodeURIComponent(profile)}&k=3`;
      const resp = await fetch(url);
      if (!resp.ok) {
        // Transient 503 (engine starting) or 404/500 — retry once; keep previous suggestions (no clear/flicker).
        if (retry && !this.app.state.gameOver) {
          window.setTimeout(() => {
            void this.refreshSuggestedMoves(false);
          }, 1200);
        }
        return;
      }
      if (this.app.state.gameOver) return;
      const data = await resp.json();
      if (this.app.state.gameOver) return;
      const suggestions = Array.isArray(data?.suggestions) ? data.suggestions : [];
      if (suggestions.length) {
        this.highlightSuggestedMoves(suggestions);
      }
      // Empty payload: keep lastSuggestionsText until a non-empty update arrives.
    } catch (_) {
      if (retry && !this.app.state.gameOver) {
        window.setTimeout(() => {
          void this.refreshSuggestedMoves(false);
        }, 1200);
      }
    }
  }

  // highlightSuggestedMoves - paints suggestion highlights and updates suggestion notes text
  highlightSuggestedMoves(suggestions) {
    this.clearSuggestedHighlights();

    this.app.state.selectedSuggestedMoves = [];
    const top = Array.isArray(suggestions) ? suggestions.slice(0, 3) : [];

    if (!top.length) {
      // Keep prior notes text; clearing here made Suggested moves jump/blank between plies.
      this.app.util.refreshNotesBox();
      return;
    }

    // Board move: amber origin + blue dest. Drop: highlight hand chip + dashed dest.
    top.forEach((sug, idx) => {
      const parsed = this.parseUciMove(sug?.move || "");
      if (!parsed) return;
      const move = sug.move || "";
      const rankLabel = String(idx + 1);

      if (parsed.dropKind) {
        const kind = this.app.SHOGI_DROP_KIND_FROM_CHAR[parsed.dropKind.toLowerCase()];
        const side = String(this.app.state.currentTurn || "white").toLowerCase();
        if (kind) {
          this.mergeHandHintRank(side, kind, rankLabel);
          document
            .querySelectorAll(`.shogi_hand_piece[data-side="${side}"][data-kind="${kind}"]`)
            .forEach((el) => this.applyHandHintToNode(el, side, kind));
        }
      } else if (parsed.from) {
        const fromSq = this.squareAt(parsed.from.file, parsed.from.rank);
        if (fromSq) {
          fromSq.classList.add(this.app.SUGGESTED_FROM_CLASS);
          this.appendHintRank(fromSq, rankLabel);
        }
      }

      const toSq = this.squareAt(parsed.to.file, parsed.to.rank);
      if (toSq) {
        toSq.classList.add(this.app.SUGGESTED_MOVE_CLASS);
        if (parsed.dropKind) toSq.classList.add(this.app.SUGGESTED_DROP_CLASS);
        this.appendHintRank(toSq, rankLabel);
        this.app.state.selectedSuggestedMoves.push({
          sequence: Number(toSq.getAttribute("data-sequence")),
          move,
        });
      }
    });

    const gt = String(this.app.state.boardGameType || this.app.el.gameTypeSelect?.value || "chess").toLowerCase();
    const header = gt === "shogi" ? "Suggested moves (including drops from hand):\n" : "Suggested moves:\n";
    let text = header;
    top.forEach((sug, idx) => {
      text += `${this.formatHintLine(idx + 1, this.parseUciMove(sug?.move || ""), sug.score_cp)}\n`;
    });
    this.app.state.lastSuggestionsText = text.trim();
    this.app.util.refreshNotesBox();
  }

  // loadSuggestedMovesForSelection - loads suggestions when a piece selection changes
  async loadSuggestedMovesForSelection(sequence) {
    if (this.app.state.isSimulationPlayback || this.app.state.gameOver) return; // Suppress during simulation / after end
    if (!this.app.state.currentGameId) {
      this.highlightSuggestedMoves([]);
      return;
    }
    const source = this.app.board.fileRankFromSequence(sequence);
    if (!source) {
      this.highlightSuggestedMoves([]);
      return;
    }
    try {
      const profile = String(this.app.el.aiStrengthSelect?.value || "intermediate");
      const url = `/api/games/${encodeURIComponent(this.app.state.currentGameId)}/top-moves?profile=${encodeURIComponent(profile)}&k=3`;
      const resp = await fetch(url);
      if (!resp.ok) {
        this.highlightSuggestedMoves([]);
        return;
      }
      const data = await resp.json();
      const suggestions = Array.isArray(data?.suggestions) ? data.suggestions : [];
      this.highlightSuggestedMoves(suggestions);
    } catch (_) {
      this.highlightSuggestedMoves([]);
    }
  }

  // startAnalysisPolling - starts analysis polling or socket fallback for a target move number
  startAnalysisPolling(targetMoveNumber, capturedSnapshot) {
    this.app.socket.stopAnalysisPolling();
    // keep composed notes (suggestions + coach Thinking…); analysis will refresh threat/suggestions
    const target = Number(targetMoveNumber) || 0;
    const expectedGameId = String(this.app.state.currentGameId || "").trim();
    if (!expectedGameId || target <= 0) return;
    const generation = Number(this.app.state.analysisPollGeneration || 0);
    this.app.state.pendingAnalysisTargetMove = target;
    this.app.state.pendingAnalysisCapturedSnapshot = capturedSnapshot || this.app.state.cachedCapturedSummary;

    if (this.app.socket.isSocketConnected()) {
      this.app.state.analysisPollFallbackTimer = window.setTimeout(() => {
        if (generation !== this.app.state.analysisPollGeneration) return;
        if (this.app.state.currentGameId !== expectedGameId) return;
        if (!this.app.socket.isSocketConnected() && this.app.state.pendingAnalysisTargetMove > 0) {
          this.startAnalysisPolling(
            this.app.state.pendingAnalysisTargetMove,
            this.app.state.pendingAnalysisCapturedSnapshot
          );
        }
      }, 1500);
      return;
    }

    const pollOnce = async () => {
      try {
        if (generation !== this.app.state.analysisPollGeneration) return;
        if (!this.app.state.currentGameId || this.app.state.currentGameId !== expectedGameId) {
          this.app.socket.stopAnalysisPolling();
          return;
        }
        const response = await fetch(`/api/games/${encodeURIComponent(expectedGameId)}/analysis/latest`, {
          method: "GET",
        });
        if (!response.ok) return;
        if (generation !== this.app.state.analysisPollGeneration) return;
        if (this.app.state.currentGameId !== expectedGameId) return;
        const payload = await response.json();
        const latestMoveNumber = Number(payload?.latest_move_number || 0);
        const latestAnalysis = payload?.latest?.analysis;
        if (!latestAnalysis) return;
        if (latestMoveNumber < target) return;
        this.app.gameInfo.renderGameInfo(
          this.app.state.pendingAnalysisCapturedSnapshot || capturedSnapshot,
          latestAnalysis
        );
        this.app.socket.stopAnalysisPolling();
      } catch (_) {
        // ignore polling errors; next poll may recover
      }
    };

    void pollOnce();
    this.app.state.analysisPollTimer = window.setInterval(() => {
      void pollOnce();
    }, 700);
  }
}

window.HintsCoach = HintsCoach;

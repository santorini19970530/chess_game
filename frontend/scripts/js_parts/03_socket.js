// CM3070 FP code
// 03_socket.js - game websocket client and live event handling for the puzzle page
// SocketClient - owns game websocket connect, reconnect, and event dispatch
class SocketClient {
  constructor(app) {
    this.app = app;
  }

  // syncGameIdFromResult - stores the game id from an api payload and reconnects when it changes
  syncGameIdFromResult(result) {
    const nextId = String(result?.game?.id || "").trim();
    if (!nextId) return;
    const changed = nextId !== this.app.state.currentGameId;
    this.app.state.currentGameId = nextId;
    if (this.app.el.gameIdInput) this.app.el.gameIdInput.value = nextId;
    if (changed) {
      // stop analysis polls for the old id — load-moves creates a new session each seek
      this.stopAnalysisPolling();
      this.connectGameSocket(nextId);
    }
  }

  // stopAnalysisPolling - clears analysis poll timers and pending analysis targets
  stopAnalysisPolling() {
    this.app.state.analysisPollGeneration = Number(this.app.state.analysisPollGeneration || 0) + 1;
    if (this.app.state.analysisPollTimer != null) {
      window.clearInterval(this.app.state.analysisPollTimer);
      this.app.state.analysisPollTimer = null;
    }
    if (this.app.state.analysisPollFallbackTimer != null) {
      window.clearTimeout(this.app.state.analysisPollFallbackTimer);
      this.app.state.analysisPollFallbackTimer = null;
    }
    this.app.state.pendingAnalysisTargetMove = 0;
    this.app.state.pendingAnalysisCapturedSnapshot = null;
  }

  // isSocketConnected - reports whether the game websocket is open
  isSocketConnected() {
    return Boolean(this.app.state.gameSocket && this.app.state.gameSocket.readyState === WebSocket.OPEN);
  }

  // clearSocketReconnectTimer - cancels a pending websocket reconnect timer
  clearSocketReconnectTimer() {
    if (this.app.state.gameSocketReconnectTimer != null) {
      window.clearTimeout(this.app.state.gameSocketReconnectTimer);
      this.app.state.gameSocketReconnectTimer = null;
    }
  }

  // socketURLForGame - builds the websocket url for a game id
  socketURLForGame(gameId) {
    const protocol = window.location.protocol === "https:" ? "wss" : "ws";
    return `${protocol}://${window.location.host}/ws/game?gameId=${encodeURIComponent(gameId)}`;
  }

  // closeGameSocket - closes the game websocket and optionally allows reconnect
  closeGameSocket(allowReconnect) {
    this.app.state.gameSocketAllowReconnect = Boolean(allowReconnect);
    this.clearSocketReconnectTimer();
    if (this.app.state.gameSocket) {
      try {
        this.app.state.gameSocket.close();
      } catch (_) {
        // ignore close errors
      }
    }
    this.app.state.gameSocket = null;
  }

  // refreshGameSnapshotFromAPI - reloads board, clocks, and coach state from rest after socket events
  async refreshGameSnapshotFromAPI(gameId) {
    const targetGameId = String(gameId || this.app.state.currentGameId || "").trim();
    if (!targetGameId) return;
    try {
      const response = await fetch(`/api/games/${encodeURIComponent(targetGameId)}`, {
        method: "GET",
      });
      if (!response.ok) return;
      const result = await response.json();
      this.app.applyGameSnapshot(result, { board: true });
      if (this.app.state.gameOver) {
        this.stopAnalysisPolling();
        return;
      }
      void this.app.coach.refreshSuggestedMoves();
      this.app.syncAnalysisAfterSnapshot(result);
    } catch (_) {
      // ignore transient refresh errors — rest remains the fallback
    }
  }

  // onMoveApplied - refreshes snapshot and coach placeholder after a live move
  onMoveApplied(gameId, data) {
    if (data?.clock || data?.remaining) {
      this.app.clocks.applyServerClock(data.clock, data.remaining);
    }
    void this.refreshGameSnapshotFromAPI(gameId);
    this.app.dom.playMoveSound(Boolean(data?.isCapture));
    // coach text arrives later — keep a placeholder so notes do not jump empty → full
    this.app.state.lastExplanationText = "[coach] Thinking…";
    this.app.util.refreshNotesBox();
  }

  // onTurnChanged - paints turn/check and syncs the active clock side
  onTurnChanged(data) {
    this.app.gameInfo.renderCurrentTurn(data?.current_turn);
    this.app.gameInfo.renderCheckState(data?.checked_side);
    if (data?.clock || data?.remaining) {
      this.app.clocks.applyServerClock(data.clock, data.remaining);
    } else if (data?.current_turn && this.app.state.clockEnabledLocal) {
      this.app.state.clockActiveSide = String(data.current_turn).toLowerCase();
      this.app.state.clockLastTickAt = Date.now();
    }
  }

  // onAnalysisStatusUpdate - paints analysis ready/error updates from the socket
  onAnalysisStatusUpdate(data) {
    const statusText = String(data?.status || "").toLowerCase();
    switch (statusText) {
      case "pending":
        break;
      case "ready":
        if (!data?.analysis) break;
        this.app.gameInfo.renderGameInfo(
          this.app.state.pendingAnalysisCapturedSnapshot || this.app.state.cachedCapturedSummary,
          data.analysis
        );
        this.stopAnalysisPolling();
        if (this.app.state.gameOver) break;
        void this.app.coach.refreshSuggestedMoves();
        break;
      case "error":
        if (this.app.state.gameOver) break;
        {
          const safeMessage = String(data?.last_error || "").trim();
          if (safeMessage) {
            this.app.state.lastThreatSummary = safeMessage;
            this.app.util.refreshNotesBox();
          }
        }
        void this.app.coach.refreshSuggestedMoves();
        break;
      default:
        break;
    }
  }

  // onExplanationReady - writes a coach explanation into the notes box
  onExplanationReady(data) {
    if (!this.app.el.gameInfoNotesBox || this.app.state.gameOver) return;
    const expl = String(data?.explanation || data?.analysis_explanation || "").trim();
    if (!expl) return;
    const skill = String(data?.skill_level || "").trim().toLowerCase();
    const bits = [];
    if (skill) bits.push(`coach:${skill}`);
    if (data?.source === "heuristic_fallback") bits.push("heuristic");
    const prefix = bits.length ? `[${bits.join(" · ")}] ` : "";
    this.app.state.lastExplanationText = prefix + expl;
    this.app.util.refreshNotesBox();
  }

  // onSimulationMove - paints a live streamed simulation move for observers
  onSimulationMove(data) {
    if (this.app.state.simulationData || this.app.state.isSimulationPlayback) return;
    const move = data?.move || "";
    const gameNum = data?.game_num || "";
    setTimeout(() => {
      if (move && this.app.el.boardElement) {
        this.app.board.applyUciMoveToBoard(move);
        this.app.dom.playMoveSound(false);
        this.app.coach.highlightSuggestedMoves([]);
      }
      const moveNum = data?.move_num || 0;
      const side = moveNum % 2 === 1 ? "white" : "black";
      const listEl = side === "white" ? this.app.el.moveHistoryWhiteList : this.app.el.moveHistoryBlackList;
      if (listEl) {
        const placeholder = listEl.querySelector(".chess_move_history_placeholder");
        if (placeholder) placeholder.remove();
        const item = document.createElement("li");
        item.textContent = move;
        listEl.appendChild(item);
        listEl.scrollTop = listEl.scrollHeight;
      }
      this.app.util.appendNotesLine(gameNum ? `Game ${gameNum}: ${move}` : move);
    }, 300);
  }

  // handleSocketMessage - routes live game, coach, and simulation websocket events
  handleSocketMessage(payload) {
    const event = String(payload?.event || "");
    const gameId = String(payload?.game_id || "");
    // simulation_* events may arrive before currentGameId is synced
    const isSimulationEvent = event.startsWith("simulation_");
    if (!event || (!isSimulationEvent && gameId !== this.app.state.currentGameId)) return;
    const data = payload?.data || {};

    switch (event) {
      case "move_applied":
        this.onMoveApplied(gameId, data);
        break;
      case "turn_changed":
        this.onTurnChanged(data);
        break;
      case "game_outcome":
        this.app.gameInfo.renderGameOutcome({
          result: data?.result,
          outcome: data?.outcome || {},
        });
        if (this.app.state.gameOver) this.app.clocks.stopClockTick();
        break;
      case "analysis_status_update":
        this.onAnalysisStatusUpdate(data);
        break;
      case "explanation_ready":
        this.onExplanationReady(data);
        break;
      case "simulation_move":
        this.onSimulationMove(data);
        break;
      case "simulation_game_end": {
        if (this.app.state.isSimulationPlayback) break;
        const status = data?.status || "finished";
        const gameNum = data?.game_num || 0;
        this.app.util.appendNotesLine(`[Game ${gameNum} ${status}]`);
        if (status === "started" && gameNum > 1) {
          this.app.simulation.resetBoardToInitialState();
          this.app.simulation.resetSimulationHistoryPanels();
        }
        break;
      }
      case "simulation_completed":
        if (this.app.state.isSimulationPlayback || !data) break;
        this.app.simulation.renderSimulationSummary(data);
        break;
      default:
        break;
    }
  }

  // connectGameSocket - opens or reuses the game websocket and schedules reconnect on close
  connectGameSocket(gameId) {
    const targetGameId = String(gameId || "").trim();
    if (!targetGameId || typeof WebSocket === "undefined") return;
    if (
      this.app.state.gameSocket &&
      this.app.state.gameSocketGameId === targetGameId &&
      (this.app.state.gameSocket.readyState === WebSocket.OPEN || this.app.state.gameSocket.readyState === WebSocket.CONNECTING)
    ) {
      return;
    }

    this.closeGameSocket(false);
    this.app.state.gameSocketAllowReconnect = true;
    this.app.state.gameSocketGameId = targetGameId;

    try {
      const ws = new WebSocket(this.socketURLForGame(targetGameId));
      this.app.state.gameSocket = ws;

      ws.addEventListener("open", () => {
        this.app.state.gameSocketReconnectAttempts = 0;
        this.clearSocketReconnectTimer();
        // resync after connect — do not trust stale local clock remaining
        void this.refreshGameSnapshotFromAPI(targetGameId);
      });

      ws.addEventListener("message", (evt) => {
        try {
          const payload = JSON.parse(String(evt.data || "{}"));
          this.handleSocketMessage(payload);
        } catch (_) {
          // ignore malformed socket payloads
        }
      });

      ws.addEventListener("close", () => {
        const sameSocket = ws === this.app.state.gameSocket;
        if (sameSocket) this.app.state.gameSocket = null;
        // do not resurrect an old session after load-moves / review seek advanced currentGameId
        if (
          !this.app.state.gameSocketAllowReconnect ||
          this.app.state.gameSocketGameId !== targetGameId ||
          this.app.state.currentGameId !== targetGameId
        ) {
          return;
        }

        // rest polling remains the fallback while the socket is down
        if (this.app.state.pendingAnalysisTargetMove > 0) {
          this.app.coach.startAnalysisPolling(
            this.app.state.pendingAnalysisTargetMove,
            this.app.state.pendingAnalysisCapturedSnapshot || this.app.state.cachedCapturedSummary
          );
        }

        this.clearSocketReconnectTimer();
        this.app.state.gameSocketReconnectAttempts += 1;
        const delay = Math.min(4000, 500 * Math.pow(2, this.app.state.gameSocketReconnectAttempts - 1));
        this.app.state.gameSocketReconnectTimer = window.setTimeout(() => {
          if (this.app.state.currentGameId !== targetGameId) return;
          this.connectGameSocket(targetGameId);
        }, delay);
      });

      ws.addEventListener("error", () => {
        try {
          ws.close();
        } catch (_) {
          // ignore
        }
      });
    } catch (_) {
      // socket init failed — existing rest flow stays source of truth
    }
  }
}

window.SocketClient = SocketClient;

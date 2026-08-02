// CM3070 FP code
// interaction.js - square selection, legal moves, and drag for the puzzle page
// BoardInteraction - owns click/drag interaction and legal-destination highlights
class BoardInteraction {
  constructor(app) {
    this.app = app;
  }

  // getSquareSequence - reads the board sequence number from a square element
  getSquareSequence(square) {
    if (!square) return NaN;
    return Number(square.getAttribute("data-sequence"));
  }

  // isCurrentTurnPiece - reports whether a square holds a piece the current player may move
  isCurrentTurnPiece(square) {
    const piece = this.getPieceOnSquare(square);
    if (!piece) return false;
    const pieceColor = String(piece.getAttribute("data-color") || "").toLowerCase();

    // In Human vs AI mode, only allow the human to move their chosen color
    // (use the stored humanColor from the game config, not the select box)
    const mode = String(this.app.el.gameModeSelect?.value || "");
    if (mode === "human_vs_ai") {
      if (pieceColor !== this.app.state.humanColor) {
        return false;
      }
    }

    return pieceColor === this.app.state.currentTurn;
  }

  // legalMoveAt - finds a cached legal move matching a destination sequence
  legalMoveAt(toSequence) {
    const target = this.app.board.fileRankFromSequence(toSequence);
    if (!target) return null;
    return (
      this.app.state.selectedLegalMoves.find(
        (move) => Number(move?.file) === target.file && Number(move?.rank) === target.rank
      ) || null
    );
  }

  // requiresPromotion - reports whether a chess destination requires a promotion choice
  requiresPromotion(toSequence) {
    if (this.app.state.boardGameType !== "chess") return false;
    return Boolean(this.legalMoveAt(toSequence)?.requiresPromotion);
  }

  // shogiPromotionFlags - returns must/can promote flags for a shogi destination
  shogiPromotionFlags(toSequence) {
    const move = this.legalMoveAt(toSequence);
    if (!move) return { must: false, can: false };
    return {
      must: Boolean(move.requiresPromotion),
      can: Boolean(move.canPromote),
    };
  }

  // clearSelectedSquare - clears piece selection, drop selection, and legal highlights
  clearSelectedSquare() {
    this.app.state.selectedSquareSequence = null;
    this.app.state.selectedDropKind = null;
    this.app.state.selectedLegalMoves = [];
    this.app.state.legalMovesRequestVersion += 1;
    this.app.el.boardElement
      .querySelectorAll(`.piece_img.${this.app.SELECTED_PIECE_CLASS}`)
      .forEach((piece) => piece.classList.remove(this.app.SELECTED_PIECE_CLASS));
    this.app.el.boardElement
      .querySelectorAll(
        `.${this.app.LEGAL_DESTINATION_CLASS}, .${this.app.LEGAL_PROMOTION_DESTINATION_CLASS}, .${this.app.LEGAL_CAPTURE_DESTINATION_CLASS}`
      )
      .forEach((square) => {
        square.classList.remove(this.app.LEGAL_DESTINATION_CLASS);
        square.classList.remove(this.app.LEGAL_PROMOTION_DESTINATION_CLASS);
        square.classList.remove(this.app.LEGAL_CAPTURE_DESTINATION_CLASS);
      });
    document.querySelectorAll(".shogi_hand_piece_selected").forEach((el) => {
      el.classList.remove("shogi_hand_piece_selected");
    });
    // Do not clear FS suggestion highlights here; they are independent of piece selection.
  }

  // selectShogiHandPiece - selects a shogi hand piece and loads drop destinations
  async selectShogiHandPiece(side, kind) {
    if (this.app.state.boardGameType !== "shogi" || this.app.state.gameOver || this.app.state.isSubmitting) return;
    if (!this.app.gameInfo.isHandSidePlayable(side)) return;
    if (
      this.app.state.selectedDropKind &&
      this.app.state.selectedDropKind.side === side &&
      this.app.state.selectedDropKind.kind === kind
    ) {
      this.clearSelectedSquare();
      return;
    }
    this.clearSelectedSquare();
    this.app.state.selectedDropKind = { side, kind };
    document
      .querySelectorAll(`.shogi_hand_piece[data-side="${side}"][data-kind="${kind}"]`)
      .forEach((el) => el.classList.add("shogi_hand_piece_selected"));
    const requestVersion = ++this.app.state.legalMovesRequestVersion;
    try {
      if (!this.app.state.currentGameId) return;
      const response = await fetch(
        `/api/games/${encodeURIComponent(this.app.state.currentGameId)}/legal-moves?dropKind=${encodeURIComponent(kind)}`
      );
      if (!response.ok) {
        if (requestVersion === this.app.state.legalMovesRequestVersion) this.highlightLegalDestinations([]);
        return;
      }
      const result = await response.json();
      if (requestVersion !== this.app.state.legalMovesRequestVersion) return;
      if (!this.app.state.selectedDropKind || this.app.state.selectedDropKind.kind !== kind) return;
      const moves = Array.isArray(result?.legalMoves) ? result.legalMoves : [];
      this.app.state.selectedLegalMoves = moves;
      this.highlightLegalDestinations(moves);
    } catch (_error) {
      if (requestVersion === this.app.state.legalMovesRequestVersion) this.highlightLegalDestinations([]);
    }
  }

  // submitShogiDrop - submits a shogi drop command for the selected hand piece
  async submitShogiDrop(toSequence) {
    if (!this.app.state.selectedDropKind) return false;
    const target = this.app.board.fileRankFromSequence(toSequence);
    if (!target) return false;
    const legal = this.app.state.selectedLegalMoves.some(
      (move) => Number(move?.file) === target.file && Number(move?.rank) === target.rank
    );
    if (!legal) return false;
    const ch = this.app.SHOGI_DROP_CHAR[this.app.state.selectedDropKind.kind];
    if (!ch) return false;
    const fileLetter = String.fromCharCode("a".charCodeAt(0) + target.file - 1);
    return this.app.setup.submitCommand(`${ch}*${fileLetter}${target.rank}`);
  }

  // highlightLegalDestinations - paints legal move, capture, and promotion destination squares
  highlightLegalDestinations(moves) {
    this.app.el.boardElement
      .querySelectorAll(
        `.${this.app.LEGAL_DESTINATION_CLASS}, .${this.app.LEGAL_PROMOTION_DESTINATION_CLASS}, .${this.app.LEGAL_CAPTURE_DESTINATION_CLASS}`
      )
      .forEach((square) => {
        square.classList.remove(this.app.LEGAL_DESTINATION_CLASS);
        square.classList.remove(this.app.LEGAL_PROMOTION_DESTINATION_CLASS);
        square.classList.remove(this.app.LEGAL_CAPTURE_DESTINATION_CLASS);
      });
    if (!Array.isArray(moves)) return;
    const selectedSource = this.app.board.fileRankFromSequence(this.app.state.selectedSquareSequence);
    const selectedSquare = selectedSource
      ? this.app.el.boardElement.querySelector(
          `.chess_board_square[data-sequence="${this.app.board.sequenceByFileRank(selectedSource.file, selectedSource.rank)}"]`
        )
      : null;
    const selectedPiece = this.getPieceOnSquare(selectedSquare);
    const selectedPieceKind = String(selectedPiece?.getAttribute("data-kind") || "").toLowerCase();
    for (const move of moves) {
      const fileNum = Number(move?.file);
      const rankNum = Number(move?.rank);
      if (Number.isNaN(fileNum) || Number.isNaN(rankNum)) continue;
      const sequence = this.app.board.sequenceByFileRank(fileNum, rankNum);
      const destinationSquare = this.app.el.boardElement.querySelector(
        `.chess_board_square[data-sequence="${sequence}"]`
      );
      if (!destinationSquare) continue;
      const isCapture = Boolean(move?.isCapture);
      if (isCapture) {
        let markerSquare = destinationSquare;
        // En passant: destination is empty, captured pawn is on source rank.
        const destinationPiece = this.getPieceOnSquare(destinationSquare);
        if (!destinationPiece && selectedSource && selectedPieceKind === "pawn" && selectedSource.file !== fileNum) {
          const capturedSequence = this.app.board.sequenceByFileRank(fileNum, selectedSource.rank);
          const capturedSquare = this.app.el.boardElement.querySelector(
            `.chess_board_square[data-sequence="${capturedSequence}"]`
          );
          if (capturedSquare) markerSquare = capturedSquare;
        }
        markerSquare.classList.add(this.app.LEGAL_CAPTURE_DESTINATION_CLASS);
      } else {
        destinationSquare.classList.add(this.app.LEGAL_DESTINATION_CLASS);
      }
      if (Boolean(move?.requiresPromotion) || Boolean(move?.canPromote)) {
        destinationSquare.classList.add(this.app.LEGAL_PROMOTION_DESTINATION_CLASS);
      }
    }
  }

  // loadLegalDestinationsForSelection - fetches legal destinations for a selected board piece
  async loadLegalDestinationsForSelection(sequence) {
    const source = this.app.board.fileRankFromSequence(sequence);
    if (!source) {
      this.highlightLegalDestinations([]);
      return;
    }
    const requestVersion = ++this.app.state.legalMovesRequestVersion;
    try {
      if (!this.app.state.currentGameId) return;
      const response = await fetch(
        `/api/games/${encodeURIComponent(this.app.state.currentGameId)}/legal-moves?file=${source.file}&rank=${source.rank}`
      );
      if (!response.ok) {
        if (requestVersion === this.app.state.legalMovesRequestVersion) {
          this.highlightLegalDestinations([]);
        }
        return;
      }
      const result = await response.json();
      if (requestVersion !== this.app.state.legalMovesRequestVersion) return;
      if (this.app.state.selectedSquareSequence !== Number(sequence)) return;
      const moves = Array.isArray(result?.legalMoves) ? result.legalMoves : [];
      this.app.state.selectedLegalMoves = moves;
      this.highlightLegalDestinations(moves);
    } catch (_error) {
      if (requestVersion === this.app.state.legalMovesRequestVersion) {
        this.highlightLegalDestinations([]);
      }
    }
  }

  // onBoardClick - handles click-to-select and click-to-move on the board
  async onBoardClick(event) {
    if (this.app.state.gameOver || this.app.state.isSubmitting || this.app.state.pendingPromotionResolve) return;
    const targetSquare = this.getSquareElement(event.target);
    if (!targetSquare) return;

    const targetSequence = this.getSquareSequence(targetSquare);
    if (Number.isNaN(targetSequence)) return;

    if (this.app.state.selectedDropKind) {
      const dropped = await this.submitShogiDrop(targetSequence);
      if (dropped) this.clearSelectedSquare();
      return;
    }

    const targetHasCurrentTurnPiece = this.isCurrentTurnPiece(targetSquare);

    if (this.app.state.selectedSquareSequence == null) {
      if (targetHasCurrentTurnPiece) this.app.board.setSelectedSquare(targetSequence);
      return;
    }

    if (targetSequence === this.app.state.selectedSquareSequence) {
      this.clearSelectedSquare();
      return;
    }

    if (targetHasCurrentTurnPiece) {
      this.app.board.setSelectedSquare(targetSequence);
      return;
    }

    const moved = await this.app.setup.submitBoardMove(this.app.state.selectedSquareSequence, targetSequence);
    if (moved) this.clearSelectedSquare();
  }

  // setShogiBlackDragImage - builds a rotated drag ghost for shogi black pieces
  setShogiBlackDragImage(event, piece) {
    if (
      !event.dataTransfer ||
      !(piece instanceof HTMLImageElement) ||
      String(this.app.el.boardWrapper?.dataset?.gameType || "").toLowerCase() !== "shogi" ||
      String(piece.getAttribute("data-color") || "").toLowerCase() !== "black"
    ) {
      return;
    }
    const rect = piece.getBoundingClientRect();
    const w = Math.max(1, Math.round(rect.width));
    const h = Math.max(1, Math.round(rect.height));
    const canvas = document.createElement("canvas");
    canvas.width = w;
    canvas.height = h;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.translate(w / 2, h / 2);
    ctx.rotate(Math.PI);
    ctx.drawImage(piece, -w / 2, -h / 2, w, h);
    event.dataTransfer.setDragImage(canvas, w / 2, h / 2);
  }

  // onBoardDragStart - starts a drag from a current-turn piece
  onBoardDragStart(event) {
    if (this.app.state.gameOver || this.app.state.isSubmitting || this.app.state.pendingPromotionResolve) {
      event.preventDefault();
      return;
    }
    const piece = event.target instanceof Element ? event.target.closest(".piece_img") : null;
    if (!piece) return;
    const sourceSquare = this.getSquareElement(piece);
    if (!sourceSquare || !this.isCurrentTurnPiece(sourceSquare)) {
      event.preventDefault();
      return;
    }
    const sourceSequence = this.getSquareSequence(sourceSquare);
    if (Number.isNaN(sourceSequence)) {
      event.preventDefault();
      return;
    }
    this.app.state.dragSourceSequence = sourceSequence;
    this.app.board.setSelectedSquare(sourceSequence);
    piece.classList.add("piece_img_dragging");
    event.dataTransfer?.setData("text/plain", String(sourceSequence));
    if (event.dataTransfer) event.dataTransfer.effectAllowed = "move";
    this.setShogiBlackDragImage(event, piece);
  }

  // onBoardDrop - completes a drag-and-drop move onto a destination square
  async onBoardDrop(event) {
    if (this.app.state.gameOver || this.app.state.isSubmitting || this.app.state.pendingPromotionResolve) return;
    const targetSquare = this.getSquareElement(event.target);
    if (!targetSquare) return;
    event.preventDefault();

    let sourceSequence = this.app.state.dragSourceSequence;
    if (sourceSequence == null) {
      const payload = Number(event.dataTransfer?.getData("text/plain"));
      if (!Number.isNaN(payload)) sourceSequence = payload;
    }
    const targetSequence = this.getSquareSequence(targetSquare);
    if (sourceSequence == null || Number.isNaN(targetSequence) || sourceSequence === targetSequence) return;

    const moved = await this.app.setup.submitBoardMove(sourceSequence, targetSequence);
    if (moved) this.clearSelectedSquare();
  }

  // initMouseMoveControls - wires board click and drag-and-drop move controls
  initMouseMoveControls() {
    this.app.el.boardElement.addEventListener("click", (event) => {
      void this.onBoardClick(event);
    });
    this.app.el.boardElement.addEventListener("dragstart", (event) => this.onBoardDragStart(event));
    this.app.el.boardElement.addEventListener("dragover", (event) => {
      const targetSquare = this.getSquareElement(event.target);
      if (!targetSquare) return;
      event.preventDefault();
      if (event.dataTransfer) event.dataTransfer.dropEffect = "move";
    });
    this.app.el.boardElement.addEventListener("drop", (event) => {
      void this.onBoardDrop(event);
    });
    this.app.el.boardElement.addEventListener("dragend", (event) => {
      const piece = event.target instanceof Element ? event.target.closest(".piece_img") : null;
      if (piece) piece.classList.remove("piece_img_dragging");
      this.app.state.dragSourceSequence = null;
    });
  }

  // getSquareElement - finds the nearest board square element from an event target
  getSquareElement(target) {
    return target instanceof Element ? target.closest(".chess_board_square[data-sequence]") : null;
  }

  // getPieceOnSquare - returns the piece image on a square, if any
  getPieceOnSquare(square) {
    return square?.querySelector(".piece_img") || null;
  }
}

window.BoardInteraction = BoardInteraction;

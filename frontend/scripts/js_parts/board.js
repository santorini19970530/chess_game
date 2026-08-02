// CM3070 FP code
// board.js - board geometry, piece images, and state rendering for the puzzle page

// BoardView - owns board grid rebuild, piece images, and board state paint
class BoardView {
  constructor(app) {
    this.app = app;
    // api kinds → xianqi_pic filenames (bear = elephant)
    this.app.XIANQI_KIND_FILE = {
      king: "general",
      advisor: "advisor",
      elephant: "bear",
      knight: "horse",
      rook: "chariot",
      cannon: "cannon",
      pawn: "soldier",
    };
    // api kinds → shogi_pic/*.svg (black via css rotate)
    this.app.SHOGI_KINDS = new Set([
      "pawn", "lance", "knight", "silver", "gold", "bishop", "rook", "king",
      "promoted_pawn", "promoted_lance", "promoted_knight", "promoted_silver",
      "horse", "dragon",
    ]);
  }

  // geometryForGameType - returns files, max rank, and type for a game type string
  geometryForGameType(type) {
    switch (String(type || "chess").toLowerCase()) {
      case "xianqi":
        return { files: 9, maxRank: 10, type: "xianqi" };
      case "shogi":
        return { files: 9, maxRank: 9, type: "shogi" };
      default:
        return { files: 8, maxRank: 8, type: "chess" };
    }
  }

  // rebuildBoardLabels - rebuilds rank and file gutter labels for the current geometry
  rebuildBoardLabels() {
    const ranksEl = this.app.el.boardWrapper.querySelector(".board_ranks");
    if (ranksEl) {
      ranksEl.replaceChildren(
        ...Array.from({ length: this.app.state.boardMaxRank }, (_, i) => {
          const span = document.createElement("span");
          span.className = "board_label";
          span.textContent = String(this.app.state.boardMaxRank - i);
          return span;
        })
      );
    }
    const filesEl = this.app.el.boardWrapper.querySelector(".board_files");
    if (filesEl) {
      // chess/xiangqi: a..i; shogi ui: 1..9
      const numericFiles = this.app.state.boardGameType === "shogi";
      filesEl.replaceChildren(
        ...Array.from({ length: this.app.state.boardFiles }, (_, i) => {
          const span = document.createElement("span");
          span.className = "board_label";
          span.textContent = numericFiles
            ? String(i + 1)
            : String.fromCharCode("a".charCodeAt(0) + i);
          return span;
        })
      );
    }
  }

  // rebuildXiangqiBoard - builds the xiangqi point board field, lines, and palace marks
  rebuildXiangqiBoard() {
    // lines at x=i/8, y=j/9 inside .xianqi_field; padding holds edge-piece overhang
    this.app.el.boardElement.classList.add("xianqi_board");
    const field = document.createElement("div");
    field.className = "xianqi_field";

    const art = document.createElement("div");
    art.className = "xianqi_artwork";
    art.setAttribute("aria-hidden", "true");

    for (let j = 0; j <= 9; j++) {
      const h = document.createElement("div");
      h.className = "xianqi_h_line";
      h.style.top = `${(j / 9) * 100}%`;
      art.appendChild(h);
    }
    for (let i = 0; i <= 8; i++) {
      const v = document.createElement("div");
      v.className = i === 0 || i === 8 ? "xianqi_v_line xianqi_v_outer" : "xianqi_v_line xianqi_v_inner";
      v.style.left = `${(i / 8) * 100}%`;
      art.appendChild(v);
    }
    for (const side of ["top", "bottom"]) {
      const palace = document.createElement("div");
      palace.className = `xianqi_palace xianqi_palace_${side}`;
      art.appendChild(palace);
    }

    const points = document.createElement("div");
    points.className = "xianqi_points";
    for (let seq = 0; seq < 90; seq++) {
      const file = (seq % 9) + 1;
      const rank = 10 - Math.floor(seq / 9);
      const sq = document.createElement("div");
      sq.className = "chess_board_square chess_board_square_light";
      sq.setAttribute("data-sequence", String(seq));
      sq.setAttribute("data-file", String(file));
      sq.setAttribute("data-rank", String(rank));
      sq.style.left = `${((file - 1) / 8) * 100}%`;
      sq.style.top = `${((10 - rank) / 9) * 100}%`;
      points.appendChild(sq);
    }

    field.append(art, points);
    this.app.el.boardElement.replaceChildren(field);
  }

  // rebuildSquareGridBoard - builds a chess/shogi square grid for the current geometry
  rebuildSquareGridBoard() {
    this.app.el.boardElement.classList.remove("xianqi_board");
    const n = this.app.state.boardFiles * this.app.state.boardMaxRank;
    const squares = [];
    for (let seq = 0; seq < n; seq++) {
      const file = (seq % this.app.state.boardFiles) + 1;
      const rank = this.app.state.boardMaxRank - Math.floor(seq / this.app.state.boardFiles);
      const row = Math.floor(seq / this.app.state.boardFiles);
      const col = seq % this.app.state.boardFiles;
      const isLight = (row + col) % 2 === 0;
      const div = document.createElement("div");
      div.className = [
        "chess_board_square",
        isLight ? "chess_board_square_light" : "chess_board_square_dark",
      ].join(" ");
      div.setAttribute("data-sequence", String(seq));
      div.setAttribute("data-file", String(file));
      div.setAttribute("data-rank", String(rank));
      squares.push(div);
    }
    this.app.el.boardElement.replaceChildren(...squares);
  }

  // syncXiangqiCoordGutters - copies board size/padding into css vars for file/rank labels
  syncXiangqiCoordGutters() {
    if (!this.app.el.boardWrapper || !this.app.el.boardElement) return;
    if (String(this.app.el.boardWrapper.dataset.gameType || "") !== "xianqi") {
      this.app.el.boardWrapper.style.removeProperty("--xq-board-w");
      this.app.el.boardWrapper.style.removeProperty("--xq-board-h");
      this.app.el.boardWrapper.style.removeProperty("--xq-label-pad-x");
      this.app.el.boardWrapper.style.removeProperty("--xq-label-pad-y");
      return;
    }
    const cs = window.getComputedStyle(this.app.el.boardElement);
    const padX = (parseFloat(cs.borderLeftWidth) || 0) + (parseFloat(cs.paddingLeft) || 0);
    const padY = (parseFloat(cs.borderTopWidth) || 0) + (parseFloat(cs.paddingTop) || 0);
    this.app.el.boardWrapper.style.setProperty("--xq-board-w", `${this.app.el.boardElement.offsetWidth}px`);
    this.app.el.boardWrapper.style.setProperty("--xq-board-h", `${this.app.el.boardElement.offsetHeight}px`);
    this.app.el.boardWrapper.style.setProperty("--xq-label-pad-x", `${padX}px`);
    this.app.el.boardWrapper.style.setProperty("--xq-label-pad-y", `${padY}px`);
  }

  // rebuildBoardGrid - rebuilds labels and squares/points for the current game type
  rebuildBoardGrid() {
    if (!this.app.el.boardElement || !this.app.el.boardWrapper) return;
    this.app.el.boardWrapper.dataset.gameType = this.app.state.boardGameType;
    this.app.el.boardWrapper.style.setProperty("--board-files", String(this.app.state.boardFiles));
    this.app.el.boardWrapper.style.setProperty("--board-ranks", String(this.app.state.boardMaxRank));
    this.rebuildBoardLabels();
    if (this.app.state.boardGameType === "xianqi") this.rebuildXiangqiBoard();
    else this.rebuildSquareGridBoard();
    // after layout: align a..i / 10..1 with the grid lines
    window.requestAnimationFrame(() => this.syncXiangqiCoordGutters());
  }

  // ensureBoardGeometry - rebuilds the board when game type geometry changes
  ensureBoardGeometry(type) {
    const g = this.geometryForGameType(type);
    if (g.files === this.app.state.boardFiles && g.maxRank === this.app.state.boardMaxRank && g.type === this.app.state.boardGameType) {
      if (this.app.el.boardWrapper) this.app.el.boardWrapper.dataset.gameType = this.app.state.boardGameType;
      return false;
    }
    this.app.state.boardFiles = g.files;
    this.app.state.boardMaxRank = g.maxRank;
    this.app.state.boardGameType = g.type;
    this.rebuildBoardGrid();
    return true;
  }

  // previewBoardForGameType - paints a start layout preview without creating a session
  previewBoardForGameType(type) {
    const t = String(type || this.app.el.gameTypeSelect?.value || this.app.state.boardGameType || "chess").toLowerCase();
    this.ensureBoardGeometry(t);
    switch (t) {
      case "xianqi":
        this.renderBoardFromState(this.app.simulation.initialXiangqiState(), "xianqi");
        break;
      case "shogi":
        this.renderBoardFromState(this.app.simulation.initialShogiState(), "shogi");
        break;
      default:
        this.renderBoardFromState(this.app.simulation.initialChessState(), "chess");
        break;
    }
  }

  // setSelectedSquare - selects a board square and loads legal destinations for that piece
  setSelectedSquare(sequence) {
    this.app.interaction.clearSelectedSquare();
    this.app.state.selectedSquareSequence = Number(sequence);
    if (Number.isNaN(this.app.state.selectedSquareSequence)) {
      this.app.state.selectedSquareSequence = null;
      return;
    }
    const selectedSquare = this.app.el.boardElement.querySelector(
      `.chess_board_square[data-sequence="${this.app.state.selectedSquareSequence}"]`
    );
    const selectedPiece = this.app.interaction.getPieceOnSquare(selectedSquare);
    if (selectedPiece) {
      selectedPiece.classList.add(this.app.SELECTED_PIECE_CLASS);
      void this.app.interaction.loadLegalDestinationsForSelection(this.app.state.selectedSquareSequence);
      // do not refresh fs suggestions on selection — only after a real move or new game
    }
  }

  // imagePathFromPiece - resolves the image url for a piece on the current game type
  imagePathFromPiece(piece) {
    const kind = String(piece?.kind || "").toLowerCase();
    const color = String(piece?.color || "").toLowerCase();
    if (!kind || !color) return "";
    switch (this.app.state.boardGameType) {
      case "xianqi": {
        const file = this.app.XIANQI_KIND_FILE[kind];
        if (!file) return "";
        const side = color === "black" ? "black" : "white";
        return `/pic/xianqi_pic/${file}_${side}.png`;
      }
      case "shogi":
        if (!this.app.SHOGI_KINDS.has(kind)) return "";
        return `/pic/shogi_pic/${kind}.svg`;
      default: {
        const tone = color === "black" ? "dark" : "light";
        return `/pic/chess_pic/${kind}_${tone}.png`;
      }
    }
  }

  // renderBoardFromState - syncs the board dom from a backend piece list
  renderBoardFromState(state, typeHint) {
    if (!Array.isArray(state)) return false;
    this.ensureBoardGeometry(typeHint || this.app.el.gameTypeSelect?.value || this.app.state.boardGameType);

    const boardSquares = this.app.el.boardElement
      ? this.app.el.boardElement.querySelectorAll(".chess_board_square[data-sequence]")
      : document.querySelectorAll(".chess_board_square[data-sequence]");
    boardSquares.forEach((square) => {
      square.querySelectorAll(".piece_img").forEach((el) => el.remove());
    });

    for (const piece of state) {
      if (!piece || !piece.file || !piece.rank) continue;
      const sequence = this.sequenceByFileRank(piece.file, piece.rank);
      const square = this.app.el.boardElement
        ? this.app.el.boardElement.querySelector(`.chess_board_square[data-sequence="${sequence}"]`)
        : document.querySelector(`.chess_board_square[data-sequence="${sequence}"]`);
      if (!square) continue;
      const imagePath = this.imagePathFromPiece(piece);
      if (!imagePath) continue;

      const img = document.createElement("img");
      img.className = "piece_img";
      img.src = imagePath;
      img.alt = `piece_${piece.file}_${piece.rank}`;
      img.setAttribute("draggable", "true");
      if (piece.color) img.setAttribute("data-color", String(piece.color).toLowerCase());
      if (piece.kind) img.setAttribute("data-kind", String(piece.kind).toLowerCase());
      square.appendChild(img);
    }

    return true;
  }

  // applyUciMoveToBoard - moves a piece image on the board for a uci string
  applyUciMoveToBoard(uci) {
    if (!uci || uci.length < 4) return;
    const match = String(uci).match(/^([a-i])(\d{1,2})([a-i])(\d{1,2})/i);
    if (!match) return;
    const fromFile = match[1].toLowerCase().charCodeAt(0) - "a".charCodeAt(0) + 1;
    const fromRank = parseInt(match[2], 10);
    const toFile = match[3].toLowerCase().charCodeAt(0) - "a".charCodeAt(0) + 1;
    const toRank = parseInt(match[4], 10);
    if (
      fromFile < 1 || fromFile > this.app.state.boardFiles || fromRank < 1 || fromRank > this.app.state.boardMaxRank ||
      toFile < 1 || toFile > this.app.state.boardFiles || toRank < 1 || toRank > this.app.state.boardMaxRank
    ) {
      return;
    }
    const fromSeq = this.sequenceByFileRank(fromFile, fromRank);
    const toSeq = this.sequenceByFileRank(toFile, toRank);

    const fromEl = this.app.el.boardElement.querySelector(`.chess_board_square[data-sequence="${fromSeq}"]`);
    const toEl = this.app.el.boardElement.querySelector(`.chess_board_square[data-sequence="${toSeq}"]`);

    if (!fromEl || !toEl) return;

    const piece = fromEl.querySelector(".piece_img");
    if (!piece) return;

    const captured = toEl.querySelector(".piece_img");
    if (captured) captured.remove();

    toEl.appendChild(piece);
  }

  // sequenceByFileRank - maps 1-based file/rank to board data-sequence
  sequenceByFileRank(fileNum, rankNum) {
    return (this.app.state.boardMaxRank - rankNum) * this.app.state.boardFiles + (fileNum - 1);
  }

  // maxSequence - returns the last board sequence index for current geometry
  maxSequence() {
    return this.app.state.boardFiles * this.app.state.boardMaxRank - 1;
  }

  // sequenceToSquare - converts a board sequence into algebraic square text
  sequenceToSquare(sequence) {
    const seq = Number(sequence);
    if (Number.isNaN(seq) || seq < 0 || seq > this.maxSequence()) return "";
    const fileChar = String.fromCharCode("a".charCodeAt(0) + (seq % this.app.state.boardFiles));
    const rankNum = this.app.state.boardMaxRank - Math.floor(seq / this.app.state.boardFiles);
    return `${fileChar}${rankNum}`;
  }

  // moveCommandFromSequence - builds a uci command from two board sequences
  moveCommandFromSequence(fromSequence, toSequence) {
    const fromSquare = this.sequenceToSquare(fromSequence);
    const toSquare = this.sequenceToSquare(toSequence);
    if (!fromSquare || !toSquare) return "";
    return `${fromSquare}${toSquare}`;
  }

  // rankFromSequence - returns the rank number for a board sequence
  rankFromSequence(sequence) {
    const seq = Number(sequence);
    if (Number.isNaN(seq) || seq < 0 || seq > this.maxSequence()) return NaN;
    return this.app.state.boardMaxRank - Math.floor(seq / this.app.state.boardFiles);
  }

  // fileRankFromSequence - converts a board sequence into file/rank numbers
  fileRankFromSequence(sequence) {
    const seq = Number(sequence);
    if (Number.isNaN(seq) || seq < 0 || seq > this.maxSequence()) return null;
    return {
      file: (seq % this.app.state.boardFiles) + 1,
      rank: this.app.state.boardMaxRank - Math.floor(seq / this.app.state.boardFiles),
    };
  }
}

if (typeof window !== "undefined") {
  window.BoardView = BoardView;
} else {
  // self-check: sequence ↔ file/rank round-trips for chess / xiangqi / shogi sizes
  const roundTrip = (files, maxRank) => {
    const seqOf = (file, rank) => (maxRank - rank) * files + (file - 1);
    const frOf = (seq) => ({
      file: (seq % files) + 1,
      rank: maxRank - Math.floor(seq / files),
    });
    for (let file = 1; file <= files; file++) {
      for (let rank = 1; rank <= maxRank; rank++) {
        const back = frOf(seqOf(file, rank));
        if (back.file !== file || back.rank !== rank) {
          throw new Error(`geometry fail files=${files} ${file},${rank} -> ${back.file},${back.rank}`);
        }
      }
    }
  };
  roundTrip(8, 8);
  roundTrip(9, 10);
  roundTrip(9, 9);
  console.log("board geometry self-check ok");
}

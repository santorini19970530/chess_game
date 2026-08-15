// CM3070 FP code
// move_history.js - white/black move history lists for the puzzle page

// MoveHistoryView - owns move history paint and history list rows
class MoveHistoryView {
  constructor(app) {
    this.app = app;
  }

  // movePieceIcon - returns a unicode fallback icon for a history piece
  movePieceIcon(side, pieceKind) {
    const color = String(side || "").toLowerCase() === "black" ? "black" : "white";
    const kind = String(pieceKind || "").toLowerCase();
    const iconMap = {
      white: {
        pawn: "♙",
        rook: "♖",
        knight: "♘",
        bishop: "♗",
        queen: "♕",
        king: "♔",
        // Xiangqi API kinds (unicode fallback when not using piece PNGs)
        cannon: "砲",
        advisor: "仕",
        elephant: "相",
        lance: "L",
        silver: "S",
        gold: "G",
        promoted_pawn: "+P",
        promoted_lance: "+L",
        promoted_knight: "+N",
        promoted_silver: "+S",
        dragon: "D",
        horse: "H",
      },
      black: {
        pawn: "♟",
        rook: "♜",
        knight: "♞",
        bishop: "♝",
        queen: "♛",
        king: "♚",
        cannon: "炮",
        advisor: "士",
        elephant: "象",
        lance: "l",
        silver: "s",
        gold: "g",
        promoted_pawn: "+p",
        promoted_lance: "+l",
        promoted_knight: "+n",
        promoted_silver: "+s",
        dragon: "d",
        horse: "h",
      },
    };
    return iconMap[color]?.[kind] || kind.slice(0, 1).toUpperCase() || "?";
  }

  // movePieceWord - returns a game-aware piece label for compact history rows
  movePieceWord(pieceKind) {
    const kind = String(pieceKind || "").trim().toLowerCase();
    const game = String(this.app?.state?.boardGameType || "chess").toLowerCase();
    if (game === "xianqi") {
      // shared api kinds use xiangqi names (pawn→soldier, rook→chariot, …)
      const xq = {
        pawn: "soldier",
        rook: "chariot",
        knight: "horse",
        king: "general",
        advisor: "advisor",
        cannon: "cannon",
        elephant: "elephant",
        bishop: "elephant",
      };
      return xq[kind] || kind.replace(/_/g, " ") || "piece";
    }
    if (game === "shogi") {
      const sg = {
        pawn: "pawn",
        lance: "lance",
        knight: "knight",
        silver: "silver",
        gold: "gold",
        bishop: "bishop",
        rook: "rook",
        king: "king",
        promoted_pawn: "tokin",
        promoted_lance: "+lance",
        promoted_knight: "+knight",
        promoted_silver: "+silver",
        dragon: "dragon",
        horse: "horse",
      };
      return sg[kind] || kind.replace(/_/g, " ") || "piece";
    }
    // chess
    const chess = {
      pawn: "pawn",
      rook: "rook",
      knight: "knight",
      bishop: "bishop",
      queen: "queen",
      king: "king",
    };
    return chess[kind] || kind.replace(/_/g, " ") || "piece";
  }

  // fillHistoryPieceIcon - fills a history list icon from png or unicode fallback
  fillHistoryPieceIcon(el, side, pieceKind, opts = {}) {
    el.className = "chess_move_history_piece_icon";
    const word = this.movePieceWord(pieceKind);
    el.setAttribute("data-label", opts.drop ? `drop ${word}` : word);
    el.replaceChildren();
    if (this.app.state?.boardGameType === "xianqi" || this.app.state?.boardGameType === "shogi") {
      const path = this.app.board?.imagePathFromPiece?.({ kind: pieceKind, color: side });
      if (path) {
        const img = document.createElement("img");
        img.src = path;
        img.alt = opts.drop ? `drop ${word}` : word;
        img.setAttribute("data-color", String(side || "").toLowerCase());
        el.appendChild(img);
        return;
      }
    }
    el.textContent = this.movePieceIcon(side, pieceKind);
  }

  // destinationFromCommand - extracts the destination square text from a uci command
  destinationFromCommand(command) {
    const text = String(command || "")
      .trim()
      .toLowerCase();
    if (!text) return "";
    // Chess a-h/1-8 (+ promo); Xiangqi/Shogi a-i and ranks to 10 (+ optional '+')
    const match = text.match(/([a-i]\d{1,2})(?:[qrbn]|\+)?$/i);
    return match ? match[1] : text;
  }

  // formatHistorySquare - formats a square using the same file/rank style as the board gutters
  formatHistorySquare(square) {
    const text = String(square || "").trim().toLowerCase();
    const m = text.match(/^([a-i])(\d{1,2})$/i);
    if (!m) return String(square || "").trim();
    const fileLetter = m[1];
    const rank = m[2];
    const fileNum = fileLetter.charCodeAt(0) - "a".charCodeAt(0) + 1;
    const game = String(this.app?.state?.boardGameType || "chess").toLowerCase();
    if (game === "shogi") {
      // board gutters use numeric files 1–9 (not a–i)
      return `${fileNum}${rank}`;
    }
    // chess (a–h) and xiangqi (a–i): letter file + rank, including xiangqi rank 10
    return `${fileLetter}${rank}`;
  }

  // isDropCommand - reports whether a command is a shogi drop (piece*square)
  isDropCommand(command) {
    return String(command || "").includes("*");
  }

  // captureMark - returns the compact capture connector for the active game
  captureMark() {
    const game = String(this.app?.state?.boardGameType || "chess").toLowerCase();
    if (game === "xianqi") return "takes";
    if (game === "shogi") return "x";
    return "x";
  }

  // opponentSide - returns the opposite color for a history side label
  opponentSide(side) {
    return String(side || "").toLowerCase() === "black" ? "white" : "black";
  }

  // appendHistoryMove - appends one move row to a white or black history list
  appendHistoryMove(listEl, side, pieceKind, toSquare, fallbackText, isCapture, capturedPieceKind, command = "") {
    const item = document.createElement("li");
    const iconSpan = document.createElement("span");
    this.fillHistoryPieceIcon(iconSpan, side, pieceKind, { drop: this.isDropCommand(command) });
    const textSpan = document.createElement("span");
    textSpan.className = "chess_move_history_move_text";
    const moveText = this.formatHistorySquare(toSquare || fallbackText || "");
    if (isCapture) {
      textSpan.appendChild(document.createTextNode(`${moveText} ${this.captureMark()} `));
      if (capturedPieceKind) {
        const capturedIcon = document.createElement("span");
        this.fillHistoryPieceIcon(capturedIcon, this.opponentSide(side), capturedPieceKind);
        textSpan.appendChild(capturedIcon);
      }
    } else {
      textSpan.textContent = moveText;
    }
    item.appendChild(iconSpan);
    item.appendChild(document.createTextNode(" "));
    item.appendChild(textSpan);
    listEl.appendChild(item);
  }

  // clearHistoryPlaceholder - removes the empty-history placeholder from a list
  clearHistoryPlaceholder(listEl) {
    const placeholder = listEl.querySelector(".chess_move_history_placeholder");
    if (placeholder) placeholder.remove();
  }

  // renderMoveHistory - rebuilds white/black move history from plain or detailed history
  renderMoveHistory(history, historyDetailed) {
    this.app.el.moveHistoryWhiteList.innerHTML = "";
    this.app.el.moveHistoryBlackList.innerHTML = "";
    if (
      (!Array.isArray(history) || history.length === 0) &&
      (!Array.isArray(historyDetailed) || historyDetailed.length === 0)
    ) {
      const whitePlaceholder = document.createElement("li");
      whitePlaceholder.className = "chess_move_history_placeholder";
      whitePlaceholder.textContent = "No moves yet.";
      this.app.el.moveHistoryWhiteList.appendChild(whitePlaceholder);

      const blackPlaceholder = document.createElement("li");
      blackPlaceholder.className = "chess_move_history_placeholder";
      blackPlaceholder.textContent = "No moves yet.";
      this.app.el.moveHistoryBlackList.appendChild(blackPlaceholder);
      return;
    }

    if (Array.isArray(historyDetailed) && historyDetailed.length > 0) {
      for (const move of historyDetailed) {
        const side = String(move?.side || "white");
        const toSquare = String(move?.to || "");
        const pieceKind = String(move?.pieceKind || "pawn");
        const command = String(move?.command || "");
        const fallbackText = this.destinationFromCommand(command);
        const isCapture = Boolean(move?.isCapture);
        const capturedPieceKind = String(move?.capturedPieceKind || "");
        if (side.toLowerCase() === "black") {
          this.appendHistoryMove(
            this.app.el.moveHistoryBlackList,
            side,
            pieceKind,
            toSquare,
            fallbackText,
            isCapture,
            capturedPieceKind,
            command
          );
        } else {
          this.appendHistoryMove(
            this.app.el.moveHistoryWhiteList,
            side,
            pieceKind,
            toSquare,
            fallbackText,
            isCapture,
            capturedPieceKind,
            command
          );
        }
      }
    } else if (Array.isArray(history)) {
      for (const move of history) {
        if (move.startsWith("White:")) {
          const commandText = move.replace(/^White:\s*/, "");
          this.appendHistoryMove(
            this.app.el.moveHistoryWhiteList,
            "white",
            "pawn",
            this.destinationFromCommand(commandText),
            commandText,
            false,
            ""
          );
        } else if (move.startsWith("Black:")) {
          const commandText = move.replace(/^Black:\s*/, "");
          this.appendHistoryMove(
            this.app.el.moveHistoryBlackList,
            "black",
            "pawn",
            this.destinationFromCommand(commandText),
            commandText,
            false,
            ""
          );
        } else {
          const commandText = String(move || "");
          this.appendHistoryMove(
            this.app.el.moveHistoryWhiteList,
            "white",
            "pawn",
            this.destinationFromCommand(commandText),
            commandText,
            false,
            ""
          );
        }
      }
    }

    if (!this.app.el.moveHistoryWhiteList.children.length) {
      const whitePlaceholder = document.createElement("li");
      whitePlaceholder.className = "chess_move_history_placeholder";
      whitePlaceholder.textContent = "No moves yet.";
      this.app.el.moveHistoryWhiteList.appendChild(whitePlaceholder);
    }
    if (!this.app.el.moveHistoryBlackList.children.length) {
      const blackPlaceholder = document.createElement("li");
      blackPlaceholder.className = "chess_move_history_placeholder";
      blackPlaceholder.textContent = "No moves yet.";
      this.app.el.moveHistoryBlackList.appendChild(blackPlaceholder);
    }

    this.app.el.moveHistoryWhiteList.scrollTop = this.app.el.moveHistoryWhiteList.scrollHeight;
    this.app.el.moveHistoryBlackList.scrollTop = this.app.el.moveHistoryBlackList.scrollHeight;
  }

}

if (typeof window !== "undefined") {
  window.MoveHistoryView = MoveHistoryView;
} else {
  // self-check: capture history needs opponentSide (missing helper aborted snapshot paint)
  const view = new MoveHistoryView({ state: { boardGameType: "chess" } });
  if (view.opponentSide("white") !== "black" || view.opponentSide("Black") !== "white") {
    throw new Error("opponentSide self-check failed");
  }
  if (view.movePieceWord("knight") !== "knight") {
    throw new Error("chess movePieceWord self-check failed");
  }
  const xq = new MoveHistoryView({ state: { boardGameType: "xianqi" } });
  if (xq.movePieceWord("pawn") !== "soldier" || xq.movePieceWord("rook") !== "chariot" || xq.movePieceWord("king") !== "general") {
    throw new Error("xiangqi movePieceWord self-check failed");
  }
  const sg = new MoveHistoryView({ state: { boardGameType: "shogi" } });
  if (sg.movePieceWord("promoted_pawn") !== "tokin" || sg.movePieceWord("dragon") !== "dragon") {
    throw new Error("shogi movePieceWord self-check failed");
  }
  if (!sg.isDropCommand("P*e5") || sg.isDropCommand("c3c4")) {
    throw new Error("isDropCommand self-check failed");
  }
  const chessSq = new MoveHistoryView({ state: { boardGameType: "chess" } });
  if (chessSq.formatHistorySquare("e4") !== "e4") {
    throw new Error("chess formatHistorySquare self-check failed");
  }
  const xqSq = new MoveHistoryView({ state: { boardGameType: "xianqi" } });
  if (xqSq.formatHistorySquare("a10") !== "a10") {
    throw new Error("xiangqi formatHistorySquare self-check failed");
  }
  if (sg.formatHistorySquare("c4") !== "34") {
    throw new Error("shogi formatHistorySquare self-check failed");
  }
  console.log("move history self-check ok");
}

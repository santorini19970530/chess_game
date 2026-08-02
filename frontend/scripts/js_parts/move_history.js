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

  // fillHistoryPieceIcon - fills a history list icon from png or unicode fallback
  fillHistoryPieceIcon(el, side, pieceKind) {
    el.className = "chess_move_history_piece_icon";
    el.replaceChildren();
    if (this.app.state.boardGameType === "xianqi" || this.app.state.boardGameType === "shogi") {
      const path = this.app.board.imagePathFromPiece({ kind: pieceKind, color: side });
      if (path) {
        const img = document.createElement("img");
        img.src = path;
        img.alt = String(pieceKind || "");
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

  // opponentSide - returns the opposite color for a history side label
  opponentSide(side) {
    return String(side || "").toLowerCase() === "black" ? "white" : "black";
  }

  // appendHistoryMove - appends one move row to a white or black history list
  appendHistoryMove(listEl, side, pieceKind, toSquare, fallbackText, isCapture, capturedPieceKind) {
    const item = document.createElement("li");
    const iconSpan = document.createElement("span");
    this.fillHistoryPieceIcon(iconSpan, side, pieceKind);
    const textSpan = document.createElement("span");
    textSpan.className = "chess_move_history_move_text";
    const moveText = toSquare || fallbackText || "";
    if (isCapture) {
      textSpan.textContent = `${moveText} x `;
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
        const fallbackText = this.destinationFromCommand(move?.command);
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
            capturedPieceKind
          );
        } else {
          this.appendHistoryMove(
            this.app.el.moveHistoryWhiteList,
            side,
            pieceKind,
            toSquare,
            fallbackText,
            isCapture,
            capturedPieceKind
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
  const view = new MoveHistoryView({});
  if (view.opponentSide("white") !== "black" || view.opponentSide("Black") !== "white") {
    throw new Error("opponentSide self-check failed");
  }
  console.log("move history self-check ok");
}

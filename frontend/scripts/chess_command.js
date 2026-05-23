// CM3070 FP code
// chess_command.js
// records the movement command from user
// this is operating on frontend level

(() => {
  const input = document.getElementById("chess_command");
  const button = document.getElementById("chess_command_submit");
  const status = document.getElementById("chess_command_status");
  const moveHistoryWhiteList = document.getElementById("chess_move_history_white");
  const moveHistoryBlackList = document.getElementById("chess_move_history_black");
  const moveSound = new Audio("/sounds/chess_movement.wav");

  if (!input || !button || !status || !moveHistoryWhiteList || !moveHistoryBlackList) return;

  input.focus();

  // set current status
  const setStatus = (message, type) => {
    status.textContent = message;
    status.className = `command_status ${type}`;
  };

  // update move history from backend source of truth
  const renderMoveHistory = (history) => {
    moveHistoryWhiteList.innerHTML = "";
    moveHistoryBlackList.innerHTML = "";
    if (!Array.isArray(history) || history.length === 0) {
      const whitePlaceholder = document.createElement("li");
      whitePlaceholder.className = "chess_move_history_placeholder";
      whitePlaceholder.textContent = "No moves yet.";
      moveHistoryWhiteList.appendChild(whitePlaceholder);

      const blackPlaceholder = document.createElement("li");
      blackPlaceholder.className = "chess_move_history_placeholder";
      blackPlaceholder.textContent = "No moves yet.";
      moveHistoryBlackList.appendChild(blackPlaceholder);
      return;
    }

    for (const move of history) {
      const item = document.createElement("li");
      if (move.startsWith("White:")) {
        item.textContent = move.replace(/^White:\s*/, "");
        moveHistoryWhiteList.appendChild(item);
      } else if (move.startsWith("Black:")) {
        item.textContent = move.replace(/^Black:\s*/, "");
        moveHistoryBlackList.appendChild(item);
      } else {
        item.textContent = move;
        moveHistoryWhiteList.appendChild(item);
      }
    }

    if (!moveHistoryWhiteList.children.length) {
      const whitePlaceholder = document.createElement("li");
      whitePlaceholder.className = "chess_move_history_placeholder";
      whitePlaceholder.textContent = "No moves yet.";
      moveHistoryWhiteList.appendChild(whitePlaceholder);
    }
    if (!moveHistoryBlackList.children.length) {
      const blackPlaceholder = document.createElement("li");
      blackPlaceholder.className = "chess_move_history_placeholder";
      blackPlaceholder.textContent = "No moves yet.";
      moveHistoryBlackList.appendChild(blackPlaceholder);
    }

    moveHistoryWhiteList.scrollTop = moveHistoryWhiteList.scrollHeight;
    moveHistoryBlackList.scrollTop = moveHistoryBlackList.scrollHeight;
  };

  // highlight the box being mentioned by user
  const squareSelectorByFileRank = (fileChar, rankChar) => {
    const fileIndex = fileChar.charCodeAt(0) - "a".charCodeAt(0) + 1;
    const rankNum = Number(rankChar);
    if (fileIndex < 1 || fileIndex > 8 || rankNum < 1 || rankNum > 8) return "";

    const sequence = (8 - rankNum) * 8 + (fileIndex - 1);
    return `.chess_board_square[data-sequence="${sequence}"]`;
  };

  // move chess piece
  const applyMoveOnBoard = (fromFile, fromRank, toFile, toRank) => {
    const fromSquare = document.querySelector(
      squareSelectorByFileRank(fromFile, fromRank)
    );
    const toSquare = document.querySelector(
      squareSelectorByFileRank(toFile, toRank)
    );
    if (!fromSquare || !toSquare) return; // check if there is such box

    // update the position of the picture of the piece
    const pieceEl = fromSquare.querySelector(".piece_img");
    if (!pieceEl) return;

    const captured = toSquare.querySelector(".piece_img");
    if (captured) captured.remove();

    toSquare.appendChild(pieceEl);
  };

  // send the movement command to backend
  const submitCommand = async () => {
    const command = input.value.trim();
    if (!command) {
      setStatus("Please enter a chess movement command.", "error");
      return;
    }

    try {
      const body = new URLSearchParams({ command });
      const response = await fetch("/command", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: body.toString(),
      });

      if (!response.ok) {
        const errorMessage = (await response.text()).trim();

        setStatus(errorMessage || "Invalid command format", "error");
        input.focus();

        return;
      }

      const result = await response.json();
      if (!result?.from || !result?.to) {
        setStatus("Invalid move response from server", "error");
        input.focus();
        return;
      }

      input.value = "";
      setStatus("Command submitted", "success");
      renderMoveHistory(result.history);
      applyMoveOnBoard(
        result.from.file,
        String(result.from.rank),
        result.to.file,
        String(result.to.rank)
      );
      try {
        moveSound.currentTime = 0;
        await moveSound.play();
      } catch (_) {
        // ignore play errors
      }
      input.focus();
    } catch (_error) {
      setStatus("Network error. Please try again.", "error");
      input.focus();
    }
  };

  button.addEventListener("click", submitCommand);
  input.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      submitCommand();
    }
  });
})();

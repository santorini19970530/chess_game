(() => {
  const input = document.getElementById("chess_command");
  const button = document.getElementById("chess_command_submit");
  const status = document.getElementById("chess_command_status");
  const moveHistoryList = document.getElementById("chess_move_history");
  const moveSound = new Audio("/sounds/chess_movement.wav");

  if (!input || !button || !status || !moveHistoryList) {
    return;
  }
  input.focus();

  const setStatus = (message, type) => {
    status.textContent = message;
    status.className = `command_status ${type}`;
  };

  const appendMoveToHistory = (command) => {
    const placeholder = moveHistoryList.querySelector(
      ".chess_move_history_placeholder"
    );
    if (placeholder) {
      placeholder.remove();
    }

    const item = document.createElement("li");
    item.textContent = command;
    moveHistoryList.appendChild(item);
    moveHistoryList.scrollTop = moveHistoryList.scrollHeight;
  };

  const squareSelectorByFileRank = (fileChar, rankChar) => {
    const fileIndex = fileChar.charCodeAt(0) - "a".charCodeAt(0) + 1;
    const rankNum = Number(rankChar);
    if (fileIndex < 1 || fileIndex > 8 || rankNum < 1 || rankNum > 8) {
      return "";
    }
    const sequence = (8 - rankNum) * 8 + (fileIndex - 1);
    return `.chess_board_square[data-sequence="${sequence}"]`;
  };

  const applyMoveOnBoard = (rawCommand) => {
    const command = rawCommand.trim().toLowerCase();
    if (command.length < 4) {
      return;
    }

    let fromFile = "";
    let fromRank = "";
    let toFile = "";
    let toRank = "";

    if (/[1-8]/.test(command[1])) {
      fromFile = command[0];
      fromRank = command[1];
      toFile = command[2];
      toRank = command[3];
    } else {
      fromFile = command[1];
      fromRank = command[2];
      toFile = command[3];
      toRank = command[4];
    }

    const fromSquare = document.querySelector(squareSelectorByFileRank(fromFile, fromRank));
    const toSquare = document.querySelector(squareSelectorByFileRank(toFile, toRank));
    if (!fromSquare || !toSquare) {
      return;
    }

    const pieceEl = fromSquare.querySelector(".piece_img");
    if (!pieceEl) {
      return;
    }

    const captured = toSquare.querySelector(".piece_img");
    if (captured) {
      captured.remove();
    }
    toSquare.appendChild(pieceEl);
  };

  const submitCommand = async () => {
    const command = input.value.trim();
    if (!command) {
      setStatus("Please enter a chess command.", "error");
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
        input.value = "";
        input.focus();
        return;
      }

      input.value = "";
      setStatus("Command submitted", "success");
      appendMoveToHistory(command);
      applyMoveOnBoard(command);
      try {
        moveSound.currentTime = 0;
        await moveSound.play();
      } catch (_) {
        // ignore play errors
      }
      input.focus();
    } catch (_error) {
      setStatus("Network error. Please try again.", "error");
      input.value = "";
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

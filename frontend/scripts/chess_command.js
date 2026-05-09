(() => {
  const input = document.getElementById("chess_command");
  const button = document.getElementById("chess_command_submit");
  const status = document.getElementById("chess_command_status");
  if (!input || !button || !status) {
    return;
  }

  const setStatus = (message, type) => {
    status.textContent = message;
    status.className = `command_status ${type}`;
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
        return;
      }

      input.value = "";
      setStatus("Command submitted", "success");
    } catch (_error) {
      setStatus("Network error. Please try again.", "error");
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

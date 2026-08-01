// CM3070 FP code
// 02_util.js - status, coach notes, and shared fetch/error helpers for the puzzle page

// Util - shared status, notes composition, and error helpers
class Util {
  constructor(app) {
    this.app = app;
  }

  // setStatus - updates the command status line text and style
  setStatus(message, type) {
    this.app.el.status.textContent = message;
    this.app.el.status.className = `command_status ${type}`;
  }

  // setNotesText - writes the notes box contents and scrolls to the end
  setNotesText(text) {
    if (!this.app.el.gameInfoNotesBox) return;
    this.app.el.gameInfoNotesBox.value = String(text || "");
    this.app.el.gameInfoNotesBox.scrollTop = this.app.el.gameInfoNotesBox.scrollHeight;
  }

  // composeNotesText - joins suggestions, threat, and coach explanation into one notes string
  composeNotesText() {
    const parts = [];
    if (this.app.state.lastSuggestionsText) parts.push(this.app.state.lastSuggestionsText);
    if (this.app.state.lastThreatSummary) parts.push(this.app.state.lastThreatSummary);
    if (this.app.state.lastExplanationText) parts.push(this.app.state.lastExplanationText);
    return parts.join("\n\n");
  }

  // refreshNotesBox - paints composed notes; keep suggestions when analysis/explain updates
  refreshNotesBox() {
    if (!this.app.el.gameInfoNotesBox) return;
    this.setNotesText(this.composeNotesText());
    if (this.app.state.lastSuggestionsText) this.app.el.gameInfoNotesBox.dataset.fsSuggestions = "1";
    else delete this.app.el.gameInfoNotesBox.dataset.fsSuggestions;
  }

  // clearCoachNotesState - clears coach explanation, suggestions, and threat text
  clearCoachNotesState() {
    this.app.state.lastExplanationText = "";
    this.app.state.lastSuggestionsText = "";
    this.app.state.lastThreatSummary = "";
  }

  // showGameEndedNotes - replaces coach notes with a game-ended message
  showGameEndedNotes(message) {
    this.clearCoachNotesState();
    this.app.coach.clearSuggestedHighlights();
    this.app.state.selectedSuggestedMoves = [];
    this.app.state.lastSuggestionsText = String(message || "Game has ended.").trim();
    this.refreshNotesBox();
  }

  // appendNotesLine - appends one line to the notes box
  appendNotesLine(line) {
    if (!this.app.el.gameInfoNotesBox) return;
    const next = String(line || "").trim();
    if (!next) return;
    const current = this.app.el.gameInfoNotesBox.value.trim();
    this.app.el.gameInfoNotesBox.value = current ? `${current}\n${next}` : next;
    this.app.el.gameInfoNotesBox.scrollTop = this.app.el.gameInfoNotesBox.scrollHeight;
  }

  // isAIVsAIModeSelected - reports whether the setup mode is ai vs ai
  isAIVsAIModeSelected() {
    return String(this.app.el.gameModeSelect?.value || "") === "ai_vs_ai";
  }

  // downloadTextFile - triggers a browser download for a text blob
  downloadTextFile(filename, mimeType, content) {
    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    link.click();
    URL.revokeObjectURL(url);
  }

  // readErrorMessage - reads a message field from a failed api response body
  async readErrorMessage(response, fallback) {
    try {
      const payload = await response.json();
      const message = String(payload?.message || "").trim();
      return message || fallback;
    } catch (_) {
      const text = (await response.text()).trim();
      return text || fallback;
    }
  }

  // setCatchStatus - maps thrown errors onto the command status line
  setCatchStatus(error, networkMsg = "Network error. Please try again.") {
    console.error(error);
    const msg = String(error?.message || "");
    const isNetwork =
      error instanceof TypeError &&
      (/failed to fetch|networkerror|load failed|network request failed/i.test(msg) || msg === "Failed to fetch");
    if (isNetwork) {
      this.setStatus(networkMsg, "error");
      return;
    }
    this.setStatus(msg ? `Error: ${msg}` : "Something went wrong. Check the console.", "error");
  }
}

window.Util = Util;

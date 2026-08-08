// CM3070 FP code
// promotion.js - chess/shogi promotion picker overlay for the puzzle page
// PromotionPicker - owns promotion picker UI and pending choice promises

class PromotionPicker {
  constructor(app) {
    this.app = app;
  }

  // closePromotionPicker - hides the promotion picker overlay
  closePromotionPicker() {
    if (!this.app.el.promotionPicker) return;
    this.app.el.promotionPicker.classList.remove("promotion_picker_visible");
    this.app.el.promotionPicker.classList.add("promotion_picker_hidden");
    this.app.el.promotionPicker.setAttribute("aria-hidden", "true");
  }

  // openPromotionPicker - shows the promotion picker overlay
  openPromotionPicker() {
    if (!this.app.el.promotionPicker) return;
    this.app.el.promotionPicker.classList.remove("promotion_picker_hidden");
    this.app.el.promotionPicker.classList.add("promotion_picker_visible");
    this.app.el.promotionPicker.setAttribute("aria-hidden", "false");
  }

  // configurePromotionPicker - fills promotion picker buttons for chess or shogi
  configurePromotionPicker(mode) {
    if (!this.app.el.promotionPicker) return;
    const title = this.app.el.promotionPicker.querySelector("#promotion_picker_title");
    const choices = this.app.el.promotionPicker.querySelector(".promotion_picker_choices");
    if (!title || !choices) return;
    if (mode === "shogi") {
      title.textContent = "Promote this piece?";
      choices.innerHTML =
        `<button type="button" class="promotion_choice_btn" data-promotion="+">Promote</button>` +
        `<button type="button" class="promotion_choice_btn" data-promotion="-">Do not promote</button>`;
      return;
    }
    title.textContent = "Choose promotion piece";
    choices.innerHTML =
      `<button type="button" class="promotion_choice_btn" data-promotion="q">Queen</button>` +
      `<button type="button" class="promotion_choice_btn" data-promotion="r">Rook</button>` +
      `<button type="button" class="promotion_choice_btn" data-promotion="b">Bishop</button>` +
      `<button type="button" class="promotion_choice_btn" data-promotion="n">Knight</button>`;
  }

  // resolvePromotionChoice - resolves a pending promotion promise with the chosen code
  resolvePromotionChoice(pieceCode) {
    if (!this.app.state.pendingPromotionResolve) return;
    const resolver = this.app.state.pendingPromotionResolve;
    this.app.state.pendingPromotionResolve = null;
    this.closePromotionPicker();
    resolver(pieceCode);
  }

  // initPromotionPicker - wires promotion picker clicks and escape cancel
  initPromotionPicker() {
    if (!this.app.el.promotionPicker) return;
    this.closePromotionPicker();
    // Delegate so chess/shogi button sets can be swapped per open.
    this.app.el.promotionPicker.addEventListener("click", (event) => {
      const buttonEl =
        event.target instanceof Element ? event.target.closest(".promotion_choice_btn[data-promotion]") : null;
      if (buttonEl) {
        const choice = String(buttonEl.getAttribute("data-promotion") || "");
        if (!choice) return;
        this.resolvePromotionChoice(choice);
        return;
      }
      if (event.target === this.app.el.promotionPicker && this.app.state.pendingPromotionResolve) {
        this.resolvePromotionChoice("");
      }
    });
    document.addEventListener("keydown", (event) => {
      if (event.key === "Escape" && this.app.state.pendingPromotionResolve) {
        this.resolvePromotionChoice("");
      }
    });
  }

  // requestPromotionChoice - requests a chess or shogi promotion choice
  requestPromotionChoice(mode = "chess") {
    return new Promise((resolve) => {
      if (!this.app.el.promotionPicker) {
        resolve(mode === "shogi" ? "+" : "q");
        return;
      }
      this.configurePromotionPicker(mode);
      this.app.state.pendingPromotionResolve = resolve;
      this.openPromotionPicker();
    });
  }
}

window.PromotionPicker = PromotionPicker;

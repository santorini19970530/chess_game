// CM3070 FP code
// how_it_works.js - how-to popup open/close for the puzzle page
// HowItWorksGuide - owns the how-it-works dialog, help button, and first-visit open

class HowItWorksGuide {
  constructor(app) {
    this.app = app;
    this.storageKey = "fyp_how_it_works_seen";
  }

  // isOpen - reports whether the how-it-works dialog is visible
  isOpen() {
    const dialog = this.app.el.howItWorksDialog;
    return Boolean(dialog && !dialog.hasAttribute("hidden"));
  }

  // open - shows the how-it-works dialog and focuses Close
  open() {
    const dialog = this.app.el.howItWorksDialog;
    if (!dialog) return;
    dialog.removeAttribute("hidden");
    const closeBtn = this.app.el.howItWorksClose;
    if (closeBtn && typeof closeBtn.focus === "function") closeBtn.focus();
  }

  // close - hides the how-it-works dialog and returns focus to Help
  close() {
    const dialog = this.app.el.howItWorksDialog;
    if (!dialog) return;
    dialog.setAttribute("hidden", "");
    const helpBtn = this.app.el.howItWorksHelp;
    if (helpBtn && typeof helpBtn.focus === "function") helpBtn.focus();
  }

  // markSeen - records that the how-to has been shown at least once
  markSeen() {
    try {
      globalThis.localStorage.setItem(this.storageKey, "1");
    } catch (_) {}
  }

  // shouldAutoOpen - true when first visit has not marked the how-to as seen
  shouldAutoOpen() {
    try {
      return globalThis.localStorage.getItem(this.storageKey) !== "1";
    } catch (_) {
      return false;
    }
  }

  // bind - wires help/close/backdrop/Escape and optional first-visit open
  bind() {
    const helpBtn = this.app.el.howItWorksHelp;
    const closeBtn = this.app.el.howItWorksClose;
    const dialog = this.app.el.howItWorksDialog;
    if (!helpBtn || !closeBtn || !dialog) return;

    helpBtn.addEventListener("click", () => {
      this.open();
      this.markSeen();
    });
    closeBtn.addEventListener("click", () => this.close());
    dialog.querySelectorAll("[data-how-it-works-dismiss]").forEach((el) => {
      el.addEventListener("click", () => this.close());
    });
    document.addEventListener("keydown", (event) => {
      if (event.key === "Escape" && this.isOpen()) {
        event.preventDefault();
        this.close();
      }
    });

    if (this.shouldAutoOpen()) {
      this.open();
      this.markSeen();
    }
  }
}

if (typeof window !== "undefined") {
  window.HowItWorksGuide = HowItWorksGuide;
} else {
  // self-check: first-visit flag and open/close toggle the hidden attribute
  const dialog = { hiddenAttr: true, hasAttribute(name) { return name === "hidden" && this.hiddenAttr; }, removeAttribute() { this.hiddenAttr = false; }, setAttribute() { this.hiddenAttr = true; }, querySelectorAll() { return []; } };
  const closeBtn = { focus() {} };
  const helpBtn = { focus() {}, addEventListener() {} };
  const app = { el: { howItWorksDialog: dialog, howItWorksClose: closeBtn, howItWorksHelp: helpBtn } };
  const guide = new HowItWorksGuide(app);
  const store = {};
  guide.storageKey = "test_how_it_works";
  const prevStorage = globalThis.localStorage;
  globalThis.localStorage = {
    getItem(k) {
      return Object.prototype.hasOwnProperty.call(store, k) ? store[k] : null;
    },
    setItem(k, v) {
      store[k] = String(v);
    },
  };
  if (!guide.shouldAutoOpen()) throw new Error("shouldAutoOpen expected true");
  guide.open();
  if (!guide.isOpen()) throw new Error("open should clear hidden");
  guide.close();
  if (guide.isOpen()) throw new Error("close should set hidden");
  guide.markSeen();
  if (guide.shouldAutoOpen()) throw new Error("markSeen should stop auto-open");
  globalThis.localStorage = prevStorage;
  console.log("how it works self-check ok");
}

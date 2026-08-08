// CM3070 FP code
// chess_command.js - constructs puzzle page controllers and starts the game session

(() => {
  const app = new GameApp();
  app.dom = new DomState(app);
  if (app.ready === false) return;

  app.util = new Util(app);
  app.socket = new SocketClient(app);
  app.clocks = new ClockController(app);
  app.board = new BoardView(app);
  app.interaction = new BoardInteraction(app);
  app.promotion = new PromotionPicker(app);
  app.coach = new HintsCoach(app);
  app.gameInfo = new GameInfoView(app);
  app.moveHistory = new MoveHistoryView(app);
  app.setup = new SetupCommand(app);
  app.session = new SessionActions(app);
  app.review = new ReviewPlayback(app);
  app.diagram = new DiagramImport(app);
  app.simulation = new SimulationPanel(app);
  window.gameApp = app;

  // wireCommandEnter - submits the command box on Enter
  app.el.input.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      app.setup.submitCommand();
    }
  });
  app.promotion.initPromotionPicker();
  app.interaction.initMouseMoveControls();
  if (typeof ResizeObserver !== "undefined" && app.el.boardElement) {
    const xqGutterRo = new ResizeObserver(() => app.board.syncXiangqiCoordGutters());
    xqGutterRo.observe(app.el.boardElement);
  }
  window.addEventListener("beforeunload", () => app.socket.closeGameSocket(false));

  app.gameInfo.renderGameInfo(null, null);
  app.simulation.clearSimulationSummary();
  app.gameInfo.renderCheckState("");
  app.gameInfo.renderGameOutcome({ status: "in_progress", result: "in_progress" });
  app.clocks.renderClocks(null);
  app.clocks.applyClockPresetToInputs();
  app.setup.renderGameConfig({
    type: "chess",
    mode: "human_vs_human",
    config: { humanColor: "white", aiGameCount: 1, startFen: "" },
  });
  app.setup.updateSetupControlState();
  void app.session.createSessionOnLoad();

  // applyAIMoveFromResult - paints a companion ai uci move onto the board when the move api returns one
  window.applyAIMoveFromResult = (result) => {
    if (!result || !result.aiMove || typeof app.board.applyUciMoveToBoard !== "function") return false;
    const uci = String(result.aiMove);
    if (!/^([a-i])(\d{1,2})([a-i])(\d{1,2})/i.test(uci)) return false;
    app.board.applyUciMoveToBoard(uci);
    return true;
  };
})();

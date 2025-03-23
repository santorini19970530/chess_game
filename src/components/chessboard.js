import { ChessBoardSquare } from "./chessboardsquare.js";

export class ChessBoard {
  #squares = [];

  constructor() {
    const board_div = document.createElement("div");

    board_div.id = "chess_board";
    board_div.classList.add("chess_board");
    document.body.appendChild(board_div);

    for (let i = 0; i < 64; i++) {
      this.#squares.push(new ChessBoardSquare(i));
    }
  }

  draw() {
    this.#squares.forEach((square) => {
      square.draw();
    });
  }
}

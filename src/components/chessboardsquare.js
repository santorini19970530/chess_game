export class ChessBoardSquare {
  #sequence;
  #isLight;
  static box_width;

  constructor(sequence) {
    this.#sequence = sequence;
    this.#isLight =
      Math.floor(this.#sequence / 8) % 2 === 0
        ? this.#sequence % 2 === 0
        : this.#sequence % 2 === 1;
    this.box_width = "30px";
  }

  draw() {
    let square_div = document.createElement("div");
    square_div.classList.add("chess_board_square");
    square_div.classList.add(
      this.#isLight ? "chess_board_square_light" : "chess_board_square_dark"
    );
    square_div.textContent = this.#sequence;

    const container = document.getElementById("chess_board");
    container.appendChild(square_div);
  }
}

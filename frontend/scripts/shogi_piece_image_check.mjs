// Runnable check: Shogi API kinds → pic/shogi_pic/*.svg (color via CSS, not filename).
import { existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const SHOGI_KINDS = [
  "pawn", "lance", "knight", "silver", "gold", "bishop", "rook", "king",
  "promoted_pawn", "promoted_lance", "promoted_knight", "promoted_silver",
  "horse", "dragon",
];

function imagePath(kind) {
  if (!SHOGI_KINDS.includes(kind)) return "";
  return `/pic/shogi_pic/${kind}.svg`;
}

const cases = [
  ["pawn", "/pic/shogi_pic/pawn.svg"],
  ["promoted_silver", "/pic/shogi_pic/promoted_silver.svg"],
  ["horse", "/pic/shogi_pic/horse.svg"],
  ["dragon", "/pic/shogi_pic/dragon.svg"],
  ["queen", ""],
];

for (const [kind, want] of cases) {
  const got = imagePath(kind);
  if (got !== want) {
    console.error(`fail: ${kind} → ${got}, want ${want}`);
    process.exit(1);
  }
}

const picDir = join(dirname(fileURLToPath(import.meta.url)), "../pic/shogi_pic");
for (const kind of SHOGI_KINDS) {
  const path = join(picDir, `${kind}.svg`);
  if (!existsSync(path)) {
    console.error(`fail: missing asset ${path}`);
    process.exit(1);
  }
}

console.log("shogi_piece_image_check: ok");

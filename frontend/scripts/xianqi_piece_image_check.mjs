// Runnable check: Xiangqi API kinds → pic/xianqi_pic filenames.
const XIANQI_KIND_FILE = {
  king: "general",
  advisor: "advisor",
  elephant: "bear",
  knight: "horse",
  rook: "chariot",
  cannon: "cannon",
  pawn: "soldier",
};

function imagePath(kind, color) {
  const file = XIANQI_KIND_FILE[kind];
  if (!file) return "";
  const side = color === "black" ? "black" : "white";
  return `/pic/xianqi_pic/${file}_${side}.png`;
}

const cases = [
  ["king", "white", "/pic/xianqi_pic/general_white.png"],
  ["elephant", "black", "/pic/xianqi_pic/bear_black.png"],
  ["rook", "white", "/pic/xianqi_pic/chariot_white.png"],
  ["cannon", "white", "/pic/xianqi_pic/cannon_white.png"],
  ["pawn", "black", "/pic/xianqi_pic/soldier_black.png"],
];

for (const [kind, color, want] of cases) {
  const got = imagePath(kind, color);
  if (got !== want) {
    console.error(`fail: ${kind}/${color} → ${got}, want ${want}`);
    process.exit(1);
  }
}
console.log("xianqi_piece_image_check: ok");

package handlers

import (
	"os"
	"path/filepath"
	"testing"
)

func frontendPicDir(t *testing.T, sub string) string {
	t.Helper()
	candidates := []string{
		filepath.Join("..", "..", "frontend", "pic", sub),
		filepath.Join("..", "frontend", "pic", sub),
		filepath.Join("frontend", "pic", sub),
	}
	for _, dir := range candidates {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			return dir
		}
	}
	t.Fatalf("frontend/pic/%s not found", sub)
	return ""
}

func TestXiangqiPieceAssetsExist(t *testing.T) {
	// API kind → xianqi_pic filename stem (same map as FE imagePathFromPiece).
	kindFile := map[string]string{
		"king": "general", "advisor": "advisor", "elephant": "bear",
		"knight": "horse", "rook": "chariot", "cannon": "cannon", "pawn": "soldier",
	}
	dir := frontendPicDir(t, "xianqi_pic")
	for kind, file := range kindFile {
		for _, side := range []string{"white", "black"} {
			path := filepath.Join(dir, file+"_"+side+".png")
			if _, err := os.Stat(path); err != nil {
				t.Errorf("%s/%s missing: %s", kind, side, path)
			}
		}
	}
}

func TestShogiPieceAssetsExist(t *testing.T) {
	kinds := []string{
		"pawn", "lance", "knight", "silver", "gold", "bishop", "rook", "king",
		"promoted_pawn", "promoted_lance", "promoted_knight", "promoted_silver",
		"horse", "dragon",
	}
	dir := frontendPicDir(t, "shogi_pic")
	for _, kind := range kinds {
		path := filepath.Join(dir, kind+".svg")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%s missing: %s", kind, path)
		}
	}
}

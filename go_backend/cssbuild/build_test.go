package cssbuild

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewestCSSSourceTime_SeesPartials(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.css")
	partsDir := filepath.Join(dir, "css_parts")
	if err := os.MkdirAll(partsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("@import \"./css_parts/board.css\";\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	part := filepath.Join(partsDir, "board.css")
	if err := os.WriteFile(part, []byte(".x{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Make part newer than input
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(part, later, later); err != nil {
		t.Fatal(err)
	}

	newest, err := newestCSSSourceTime(input)
	if err != nil {
		t.Fatal(err)
	}
	if newest.Before(later.Add(-time.Second)) {
		t.Fatalf("expected newest to reflect css_parts mtime, got %v want ~%v", newest, later)
	}
}

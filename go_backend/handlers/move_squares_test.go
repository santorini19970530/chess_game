package handlers

import "testing"

func TestParseVariantUCISquares(t *testing.T) {
	ff, fr, tf, tr, err := parseVariantUCISquares("i4i5")
	if err != nil || ff != "i" || fr != 4 || tf != "i" || tr != 5 {
		t.Fatalf("i4i5 -> %s%d %s%d err=%v", ff, fr, tf, tr, err)
	}
	ff, fr, tf, tr, err = parseVariantUCISquares("h3h10")
	if err != nil || ff != "h" || fr != 3 || tf != "h" || tr != 10 {
		t.Fatalf("h3h10 -> %s%d %s%d err=%v", ff, fr, tf, tr, err)
	}
	ff, fr, tf, tr, err = parseVariantUCISquares("e8e9+")
	if err != nil || ff != "e" || fr != 8 || tf != "e" || tr != 9 {
		t.Fatalf("e8e9+ -> %s%d %s%d err=%v", ff, fr, tf, tr, err)
	}
}

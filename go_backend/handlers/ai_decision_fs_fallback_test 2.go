package handlers

import "testing"

func TestFsProfileFallbackChain(t *testing.T) {
	got := fsProfileFallbackChain("master")
	want := []string{"master", "advanced", "intermediate", "beginner"}
	if len(got) != len(want) {
		t.Fatalf("len=%d", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	got = fsProfileFallbackChain("intermediate")
	if got[0] != "intermediate" || got[len(got)-1] != "beginner" {
		t.Fatalf("got %v", got)
	}
}

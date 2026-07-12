package session

import "testing"

func TestProfileForSide_PrefersSideProfiles(t *testing.T) {
	cfg := GameConfig{
		AIProfile:      "intermediate",
		WhiteAIProfile: "beginner",
		BlackAIProfile: "master",
	}
	if got := ProfileForSide(cfg, "white"); got != "beginner" {
		t.Fatalf("white: got %q", got)
	}
	if got := ProfileForSide(cfg, "black"); got != "master" {
		t.Fatalf("black: got %q", got)
	}
}

func TestProfileForSide_FallsBackToAIProfile(t *testing.T) {
	cfg := GameConfig{AIProfile: "advanced"}
	if got := ProfileForSide(cfg, "white"); got != "advanced" {
		t.Fatalf("got %q", got)
	}
}

func TestParseAIProfile_RejectsUnknown(t *testing.T) {
	if _, ok := ParseAIProfile("grandmaster"); ok {
		t.Fatal("expected reject")
	}
	if p, ok := ParseAIProfile("Beginner"); !ok || p != "beginner" {
		t.Fatalf("got %q ok=%v", p, ok)
	}
}

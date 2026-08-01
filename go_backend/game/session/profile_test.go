// CM3070 FP code
// profile_test.go - tests for profile

package session

import "testing"

// TestCreateAndUpdateGameConfig_StoresAIProfile - checks create and update game config stores ai profile
func TestCreateAndUpdateGameConfig_StoresAIProfile(t *testing.T) {
	game, err := CreateGame(GameModeHumanVsAI, GameTypeChess, "white", 1, "", "master")
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	if game.Config.AIProfile != "master" {
		t.Fatalf("create AIProfile=%q, want master", game.Config.AIProfile)
	}
	if game.Config.WhiteAIProfile != "master" || game.Config.BlackAIProfile != "master" {
		t.Fatalf("create side profiles not synced: white=%q black=%q", game.Config.WhiteAIProfile, game.Config.BlackAIProfile)
	}

	updated, err := UpdateGameConfigByID(game.ID, GameModeHumanVsAI, GameTypeChess, "white", 1, "", "beginner")
	if err != nil {
		t.Fatalf("UpdateGameConfigByID: %v", err)
	}
	if updated.Config.AIProfile != "beginner" {
		t.Fatalf("update AIProfile=%q, want beginner", updated.Config.AIProfile)
	}

	got, err := GetGameSessionByID(game.ID)
	if err != nil {
		t.Fatalf("GetGameSessionByID: %v", err)
	}
	if got.Config.AIProfile != "beginner" {
		t.Fatalf("persisted AIProfile=%q, want beginner", got.Config.AIProfile)
	}
	// master create → skillLevel advanced by default; update preserves until SetSkillLevel
	if game.Config.SkillLevel != "advanced" {
		t.Fatalf("create SkillLevel=%q, want advanced (from master)", game.Config.SkillLevel)
	}
	set, err := SetSkillLevelByID(game.ID, "beginner")
	if err != nil {
		t.Fatalf("SetSkillLevelByID: %v", err)
	}
	if set.Config.SkillLevel != "beginner" {
		t.Fatalf("SkillLevel=%q, want beginner", set.Config.SkillLevel)
	}
	if ResolveSkillLevel(set.Config.SkillLevel, set.Config.AIProfile) != "beginner" {
		t.Fatalf("ResolveSkillLevel should prefer explicit coach level")
	}
}

// TestProfileForSide_PrefersSideProfiles - checks profile for side prefers side profiles
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

// TestProfileForSide_FallsBackToAIProfile - checks profile for side falls back to ai profile
func TestProfileForSide_FallsBackToAIProfile(t *testing.T) {
	cfg := GameConfig{AIProfile: "advanced"}
	if got := ProfileForSide(cfg, "white"); got != "advanced" {
		t.Fatalf("got %q", got)
	}
}

// TestParseAIProfile_RejectsUnknown - checks parse ai profile rejects unknown
func TestParseAIProfile_RejectsUnknown(t *testing.T) {
	if _, ok := ParseAIProfile("grandmaster"); ok {
		t.Fatal("expected reject")
	}
	if p, ok := ParseAIProfile("Beginner"); !ok || p != "beginner" {
		t.Fatalf("got %q ok=%v", p, ok)
	}
}

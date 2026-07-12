package main

import "testing"

func TestResolveMatchProfiles_SideOverrides(t *testing.T) {
	white, black, err := resolveMatchProfiles("intermediate", "beginner", "master")
	if err != nil {
		t.Fatal(err)
	}
	if white != "beginner" || black != "master" {
		t.Fatalf("got white=%s black=%s", white, black)
	}
}

func TestResolveMatchProfiles_ShorthandBothSides(t *testing.T) {
	white, black, err := resolveMatchProfiles("advanced", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if white != "advanced" || black != "advanced" {
		t.Fatalf("got white=%s black=%s", white, black)
	}
}

func TestResolveMatchProfiles_RejectsUnknown(t *testing.T) {
	if _, _, err := resolveMatchProfiles("nope", "", ""); err == nil {
		t.Fatal("expected error")
	}
}

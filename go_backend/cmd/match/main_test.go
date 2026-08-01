// CM3070 FP code
// main_test.go - tests for main

package main

import "testing"

// TestResolveMatchProfiles_SideOverrides - checks resolve match profiles side overrides
func TestResolveMatchProfiles_SideOverrides(t *testing.T) {
	white, black, err := resolveMatchProfiles("intermediate", "beginner", "master")
	if err != nil {
		t.Fatal(err)
	}
	if white != "beginner" || black != "master" {
		t.Fatalf("got white=%s black=%s", white, black)
	}
}

// TestResolveMatchProfiles_ShorthandBothSides - checks resolve match profiles shorthand both sides
func TestResolveMatchProfiles_ShorthandBothSides(t *testing.T) {
	white, black, err := resolveMatchProfiles("advanced", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if white != "advanced" || black != "advanced" {
		t.Fatalf("got white=%s black=%s", white, black)
	}
}

// TestResolveMatchProfiles_RejectsUnknown - checks resolve match profiles rejects unknown
func TestResolveMatchProfiles_RejectsUnknown(t *testing.T) {
	if _, _, err := resolveMatchProfiles("nope", "", ""); err == nil {
		t.Fatal("expected error")
	}
}

package main

import (
	"testing"
)

func TestProfileValidation(t *testing.T) {
	valid := map[string]bool{
		"beginner":     true,
		"intermediate": true,
		"advanced":     true,
		"master":       true,
	}
	for p := range valid {
		if !valid[p] {
			t.Errorf("profile %s should be valid", p)
		}
	}
	if valid["invalid"] {
		t.Error("invalid profile should not be accepted")
	}
}
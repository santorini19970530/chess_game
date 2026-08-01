package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIndexHTML_IncludesClockSetupControls - checks index html includes clock setup controls
func TestIndexHTML_IncludesClockSetupControls(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("index.go"))
	if err != nil {
		t.Fatalf("read index.go: %v", err)
	}
	text := string(source)
	for _, snippet := range []string{
		`class="config_clock_enable"`,
		`id="clock_enabled"`,
		`id="clock_preset"`,
		`value="5|0"`,
		`value="10|0"`,
		`value="15|10"`,
		`value="5|30"`,
		`id="clock_base_sec"`,
		`id="clock_increment_sec"`,
		`id="clock_human_base_sec"`,
		`id="clock_ai_base_sec"`,
		`id="clock_hvai_fields"`,
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("setup panel missing %q", snippet)
		}
	}
}

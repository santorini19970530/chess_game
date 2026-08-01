// CM3070 FP code
// clock_setup_html_test.go - tests for clock setup html

package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIndexHTML_IncludesClockSetupControls - checks game_config puzzle includes clock setup controls
func TestIndexHTML_IncludesClockSetupControls(t *testing.T) {
	candidates := []string{
		filepath.Join("..", "..", "frontend", "html_puzzles", "game_config.html"),
		filepath.Join("..", "frontend", "html_puzzles", "game_config.html"),
		filepath.Join("frontend", "html_puzzles", "game_config.html"),
	}
	var text string
	var lastErr error
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err == nil {
			text = string(data)
			break
		}
		lastErr = err
	}
	if text == "" {
		t.Fatalf("read game_config.html: %v", lastErr)
	}
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

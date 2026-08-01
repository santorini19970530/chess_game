// CM3070 FP code
// analysis_export_path_test.go - tests for analysis export path

package handlers

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAnalysisExportDir_UsesGoModDirectory - checks exports land under module data/ not handlers/data
func TestAnalysisExportDir_UsesGoModDirectory(t *testing.T) {
	modDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	nested := filepath.Join(modDir, "handlers")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir handlers: %v", err)
	}
	t.Chdir(nested)
	if resolved, err := filepath.EvalSymlinks(modDir); err == nil {
		modDir = resolved
	}
	got := analysisExportDir()
	want := filepath.Join(modDir, "data", "analysis_exports")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

// TestAnalysisExportDir_RespectsEnvOverride - checks ANALYSIS_EXPORT_DIR overrides the default
func TestAnalysisExportDir_RespectsEnvOverride(t *testing.T) {
	t.Setenv("ANALYSIS_EXPORT_DIR", "/tmp/custom-analysis-exports")
	if got := analysisExportDir(); got != "/tmp/custom-analysis-exports" {
		t.Fatalf("expected env override, got %q", got)
	}
}

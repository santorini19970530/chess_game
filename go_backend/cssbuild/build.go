// CM3070 FP code
// build.go - builds and writes bundled frontend css

package cssbuild

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// global mutex for the css build
var buildMu sync.Mutex

// EnsureStyleCSS - rebuilds style.css from input.css when input or any imported partial under the same styles directory is newer than the output
func EnsureStyleCSS(inputPath, outputPath, tailwindPath string) error {
	buildMu.Lock()
	defer buildMu.Unlock()

	newest, err := newestCSSSourceTime(inputPath)
	if err != nil {
		return err
	}

	outputInfo, err := os.Stat(outputPath)
	if err == nil && !newest.After(outputInfo.ModTime()) {
		return nil
	}

	cmd := exec.Command(tailwindPath, "-i", inputPath, "-o", outputPath)
	cmd.Dir = tailwindProjectRoot(inputPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("tailwind build failed: %v\n%s", err, out)
		return err
	}

	log.Printf("rebuilt %s from %s (+ css_parts)", filepath.Base(outputPath), filepath.Base(inputPath))
	return nil
}

// tailwindProjectRoot - chess_game/ from styles/input.css (Tailwind v4 content scan cwd)
func tailwindProjectRoot(inputPath string) string {
	return filepath.Clean(filepath.Join(filepath.Dir(inputPath), "..", ".."))
}

// newestCSSSourceTime - performs newest css source time
func newestCSSSourceTime(inputPath string) (time.Time, error) {
	info, err := os.Stat(inputPath)
	if err != nil {
		return time.Time{}, err
	}
	newest := info.ModTime()

	stylesDir := filepath.Dir(inputPath)
	err = filepath.Walk(stylesDir, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fi.IsDir() {
			// skip nested junk; allow css_parts/
			base := filepath.Base(path)
			if base == "node_modules" || base == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".css" {
			return nil
		}
		// style.css is the build output — ignore it
		if filepath.Base(path) == "style.css" {
			return nil
		}
		if fi.ModTime().After(newest) {
			newest = fi.ModTime()
		}
		return nil
	})
	return newest, err
}

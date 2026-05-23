package cssbuild

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var buildMu sync.Mutex

func EnsureStyleCSS(inputPath, outputPath, tailwindPath string) error {
	buildMu.Lock()
	defer buildMu.Unlock()

	inputInfo, err := os.Stat(inputPath)
	if err != nil {
		return err
	}

	outputInfo, err := os.Stat(outputPath)
	if err == nil && !inputInfo.ModTime().After(outputInfo.ModTime()) {
		return nil
	}

	cmd := exec.Command(tailwindPath, "-i", inputPath, "-o", outputPath)
	cmd.Dir = filepath.Dir(inputPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("tailwind build failed: %v\n%s", err, out)
		return err
	}

	log.Printf("rebuilt %s from %s", filepath.Base(outputPath), filepath.Base(inputPath))
	return nil
}

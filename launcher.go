package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// launchInDir is an optional helper to run a shell in the given directory.
// It is intentionally conservative and off by default to avoid surprising UI behavior.
// If the env var GO_DASH_LAUNCH_IN_DIR is set to a non-empty value, we will actually
// launch a simple shell (e.g., bash) in the target dir. Otherwise we just print a message.
func launchInDir(dir string) error {
	if v := os.Getenv("GO_DASH_LAUNCH_IN_DIR"); strings.TrimSpace(v) != "" {
		// Basic safety: ensure the directory exists
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			cmd := exec.Command("bash")
			cmd.Dir = dir
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}
		return fmt.Errorf("directory does not exist: %s", dir)
	}
	// No-launch mode: just print a message
	fmt.Printf("Would launch shell in: %s\n", dir)
	return nil
}

// NFC: utility to ensure dir exists or resolve a path safely
func resolveDirPath(path string) string {
	if path == "" {
		return "/"
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

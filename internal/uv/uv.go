// Package uv manages the uv binary used to install and run Python CLI tools
// (currently streamrip). uv is a single self-contained executable that
// bootstraps its own Python, so no system Python is required.
package uv

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mmrmagno/mangolib/internal/ui"
)

// installScript is the official non-interactive uv installer (macOS + Linux).
const installScript = "curl -LsSf https://astral.sh/uv/install.sh | sh"

// ManagedPath is where the install script places the uv binary.
func ManagedPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin", "uv")
}

// Locate finds uv: any existing install on PATH first (so we never duplicate
// a system/homebrew/pip uv), then the script-managed path.
func Locate() (string, error) {
	if p, err := exec.LookPath("uv"); err == nil {
		return p, nil
	}
	if p := ManagedPath(); fileExists(p) {
		return p, nil
	}
	return "", fmt.Errorf("uv not found")
}

// EnsureInstalled installs uv via the official script only if it is missing,
// and returns the path to the usable binary.
func EnsureInstalled() (string, error) {
	if p, err := Locate(); err == nil {
		return p, nil
	}
	ui.Step("uv not found — installing via astral.sh script...")
	cmd := exec.Command("sh", "-c", installScript)
	if ui.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("installing uv: %w (install manually: %s)", err, installScript)
	}
	p, err := Locate()
	if err != nil {
		return "", fmt.Errorf("uv installed but not found — ensure ~/.local/bin is on PATH")
	}
	ui.Success("uv installed at " + p)
	return p, nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

package ipod

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/mmrmagno/mangolib/internal/config"
	"github.com/mmrmagno/mangolib/internal/ui"
)

// Sync rsyncs the music library to the configured iPod mount.
func Sync(cfg *config.Config, dryRun bool) error {
	src := strings.TrimRight(cfg.MusicLibrary, "/") + "/"
	dst := cfg.IPodMount
	return runSync(cfg, src, dst, dryRun)
}

// SyncFrom rsyncs from the iPod back into the music library.
func SyncFrom(cfg *config.Config, dryRun bool) error {
	src := strings.TrimRight(cfg.IPodMount, "/") + "/"
	dst := cfg.MusicLibrary
	return runSync(cfg, src, dst, dryRun)
}

func runSync(cfg *config.Config, src, dst string, dryRun bool) error {
	if cfg.IPodMount == "" {
		ui.Warn("ipod_mount is not set in mangolib.toml")
		return nil
	}
	if _, err := os.Stat(cfg.IPodMount); err != nil {
		ui.Warn(fmt.Sprintf("iPod not mounted at %s", cfg.IPodMount))
		return nil
	}
	if _, err := exec.LookPath("rsync"); err != nil {
		return fmt.Errorf("rsync not found: install it to sync your iPod")
	}

	args := []string{
		"-av",
		"--no-owner",
		"--no-group",
		"--ignore-existing",
		"--exclude=.DS_Store",
	}
	if dryRun {
		args = append(args, "--dry-run")
	}
	args = append(args, src, dst)

	cmd := exec.Command("rsync", args...)
	// In verbose mode or dry-run, show rsync output so user sees what's happening.
	if ui.Verbose || dryRun {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rsync: %w", err)
	}
	return nil
}

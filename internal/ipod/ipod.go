package ipod

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mmrmagno/mangolib/internal/config"
	"github.com/mmrmagno/mangolib/internal/ui"
)

// SyncIPod rsyncs the music library to the configured iPod mount.
func Sync(cfg *config.Config) error {
	if cfg.IPodMount == "" {
		ui.Warn("ipod_mount is not set in mangolib.toml — skipping sync")
		return nil
	}

	if _, err := os.Stat(cfg.IPodMount); err != nil {
		ui.Warn(fmt.Sprintf("iPod not mounted at %s — skipping sync", cfg.IPodMount))
		return nil
	}

	if _, err := exec.LookPath("rsync"); err != nil {
		return fmt.Errorf("rsync not found — install it to sync your iPod")
	}

	src := strings.TrimRight(cfg.MusicLibrary, "/") + "/"
	dst := cfg.IPodMount

	ui.Step(fmt.Sprintf("Syncing %s → %s", src, dst))

	cmd := exec.Command("rsync",
		"-av",
		"--no-owner",
		"--no-group",
		"--ignore-existing",
		"--exclude=.DS_Store",
		src,
		dst,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rsync: %w", err)
	}

	ui.Success("Sync complete")
	return nil
}

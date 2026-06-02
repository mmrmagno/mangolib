// Package streamrip installs and drives the streamrip `rip` CLI to download
// music from Tidal. streamrip is installed in an isolated tool environment via
// uv, so it does not touch the user's system Python.
package streamrip

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mmrmagno/mangolib/internal/ui"
	"github.com/mmrmagno/mangolib/internal/uv"
	"github.com/pelletier/go-toml/v2"
)

// QualityArg maps mangolib's Tidal quality enum (config [tidal] quality) to
// streamrip's quality integer. streamrip Tidal ints:
//
//	0 = 256k AAC, 1 = 320k AAC, 2 = HiFi FLAC, 3 = MQA FLAC.
func QualityArg(q string) string {
	switch strings.ToUpper(strings.TrimSpace(q)) {
	case "LOW":
		return "0"
	case "HIGH":
		return "1"
	case "LOSSLESS":
		return "2"
	case "HI_RES", "HI_RES_LOSSLESS":
		return "3"
	default:
		return "3"
	}
}

// buildRipArgs assembles the `rip` argument vector. Global flags MUST precede
// the `url` subcommand (streamrip uses a click group).
//
// --no-db disables streamrip's persistent "already downloaded" database: mangolib
// downloads into an ephemeral temp dir and does its own library dedup via
// catalog.OrganizeFile + Library.DuplicateAction, so streamrip's DB would only
// cause re-downloads to be silently skipped.
func buildRipArgs(configPath, folder, quality, url string) []string {
	return []string{
		"--config-path", configPath,
		"--folder", folder,
		"--quality", quality,
		"--no-db",
		"url", url,
	}
}

// ConfigPath is the mangolib-owned streamrip config file. Pointing streamrip
// at our own path keeps the Tidal OAuth tokens in a known, persistent location
// and lets us reliably detect whether login has happened.
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "mangolib", "streamrip.toml")
}

// IsAuthenticated reports whether streamrip's config holds a Tidal access token.
func IsAuthenticated(configPath string) bool {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	return isAuthenticatedFromTOML(data)
}

// ManagedPath is where `uv tool install streamrip` places the rip shim.
func ManagedPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin", "rip")
}

// Locate finds the rip CLI: any existing install on PATH first (so we never
// reinstall a streamrip the user already has), then the uv-managed shim.
func Locate() (string, error) {
	if p, err := exec.LookPath("rip"); err == nil {
		return p, nil
	}
	if p := ManagedPath(); fileExists(p) {
		return p, nil
	}
	return "", fmt.Errorf("rip (streamrip) not found")
}

// EnsureInstalled installs streamrip via uv only if rip is missing.
func EnsureInstalled() error {
	if _, err := Locate(); err == nil {
		return nil
	}
	uvBin, err := uv.EnsureInstalled()
	if err != nil {
		return err
	}
	ui.Step("Installing streamrip via uv...")
	// Install from git: PyPI has 2.1.0 which has broken Tidal OAuth credentials.
	// The fix landed in 2.2.0 (git HEAD) but is not yet published to PyPI.
	// CFLAGS: pillow 10.x uses function pointer signatures that gcc 14+ treats as
	// hard errors; downgrading to a warning lets it compile without changing behaviour.
	cmd := exec.Command(uvBin, "tool", "install", "git+https://github.com/nathom/streamrip")
	cmd.Env = append(os.Environ(), "CFLAGS=-Wno-error=incompatible-pointer-types")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"streamrip auto-install failed\n" +
				"  Arch Linux:  yay -S streamrip-git\n" +
				"  macOS/other: uv tool install streamrip",
		)
	}
	if _, err := Locate(); err != nil {
		return fmt.Errorf("streamrip installed but rip not found — ensure ~/.local/bin is on PATH")
	}
	ui.Success("streamrip installed")
	return nil
}

// Download runs `rip url` with the terminal attached so streamrip shows its own
// progress and, on first use with no stored token, drives the Tidal OAuth
// device-code login (prints a verification URL/code and waits for the user).
// authenticated=false forces output visible regardless of ui.Verbose so the
// OAuth device-code prompt always reaches the user.
func Download(ctx context.Context, configPath, folder, quality, url string, authenticated bool) error {
	bin, err := Locate()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, bin, buildRipArgs(configPath, folder, quality, url)...)
	cmd.Stdin = os.Stdin
	if !authenticated || ui.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}
	return cmd.Run()
}

// disableVideos sets [tidal] download_videos = false in a streamrip config,
// preserving all other keys (including stored auth tokens). Idempotent.
func disableVideos(data []byte) ([]byte, error) {
	var cfg map[string]any
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	tidal, ok := cfg["tidal"].(map[string]any)
	if !ok {
		tidal = map[string]any{}
		cfg["tidal"] = tidal
	}
	tidal["download_videos"] = false
	return toml.Marshal(cfg)
}

// EnsureConfig makes sure streamrip's config exists at configPath and has video
// downloads disabled (mangolib is an audio library). Safe to call repeatedly.
func EnsureConfig(configPath string) error {
	if !fileExists(configPath) {
		bin, err := Locate()
		if err != nil {
			return err
		}
		// `rip config path` makes streamrip write its default config to
		// configPath (its group callback auto-creates it) non-interactively.
		cmd := exec.Command(bin, "--config-path", configPath, "config", "path")
		if ui.Verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("initializing streamrip config: %w", err)
		}
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading streamrip config: %w", err)
	}
	out, err := disableVideos(data)
	if err != nil {
		return fmt.Errorf("updating streamrip config: %w", err)
	}
	return os.WriteFile(configPath, out, 0644)
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func isAuthenticatedFromTOML(data []byte) bool {
	var c struct {
		Tidal struct {
			AccessToken string `toml:"access_token"`
		} `toml:"tidal"`
	}
	if err := toml.Unmarshal(data, &c); err != nil {
		return false
	}
	return c.Tidal.AccessToken != ""
}

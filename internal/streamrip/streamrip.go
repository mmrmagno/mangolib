// Package streamrip installs and drives the streamrip `rip` CLI to download
// music from Tidal. streamrip is installed in an isolated tool environment via
// uv, so it does not touch the user's system Python.
package streamrip

import (
	"os"
	"path/filepath"
	"strings"

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
func buildRipArgs(configPath, folder, quality, url string) []string {
	return []string{
		"--config-path", configPath,
		"--folder", folder,
		"--quality", quality,
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

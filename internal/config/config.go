package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	MusicLibrary string   `mapstructure:"music_library"`
	IPodMount    string   `mapstructure:"ipod_mount"`
	Spotify      Spotify  `mapstructure:"spotify"`
	Download     Download `mapstructure:"download"`
	Tidal        Tidal    `mapstructure:"tidal"`
	Library      Library  `mapstructure:"library"`
	Covers       Covers   `mapstructure:"covers"`
}

type Spotify struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

type Download struct {
	AudioFormat  string `mapstructure:"audio_format"`
	AudioQuality string `mapstructure:"audio_quality"`
}

type Tidal struct {
	Quality string `mapstructure:"quality"`
}

type Library struct {
	DuplicateAction string `mapstructure:"duplicate_action"`
}

type Covers struct {
	Size int `mapstructure:"size"`
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "mangolib", "mangolib.toml")
}

// ConfigPath can be overridden by the --config flag before Load() is called.
var ConfigPath string

func Load() (*Config, error) {
	path := ConfigPath
	if path == "" {
		path = DefaultPath()
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := writeDefaults(path); err != nil {
			return nil, fmt.Errorf("creating default config: %w", err)
		}
		fmt.Printf("Created default config at %s\n", path)
		fmt.Println("Edit it to set ipod_mount and Spotify credentials.")
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("toml")

	// Defaults for any keys absent from the file.
	home, _ := os.UserHomeDir()
	v.SetDefault("music_library", filepath.Join(home, "Music", "mangolib"))
	v.SetDefault("ipod_mount", "")
	v.SetDefault("download.audio_format", "mp3")
	v.SetDefault("download.audio_quality", "320")
	v.SetDefault("tidal.quality", "HI_RES_LOSSLESS")
	v.SetDefault("library.duplicate_action", "skip")
	v.SetDefault("covers.size", 500)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg.MusicLibrary = expandHome(cfg.MusicLibrary)
	cfg.IPodMount = expandHome(cfg.IPodMount)
	return &cfg, nil
}

func expandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, path[2:])
}

func writeDefaults(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	const defaults = `music_library = "~/Music/mangolib"
ipod_mount    = ""

[spotify]
client_id     = ""
client_secret = ""

[download]
audio_format  = "mp3"
audio_quality = "320"

[tidal]
quality = "HI_RES_LOSSLESS"

[library]
duplicate_action = "skip"

[covers]
size = 500
`
	return os.WriteFile(path, []byte(defaults), 0644)
}

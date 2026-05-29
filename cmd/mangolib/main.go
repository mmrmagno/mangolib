package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mmrmagno/mangolib/internal/config"
	"github.com/mmrmagno/mangolib/internal/covers"
	"github.com/mmrmagno/mangolib/internal/download"
	"github.com/mmrmagno/mangolib/internal/catalog"
	"github.com/mmrmagno/mangolib/internal/ipod"
	"github.com/mmrmagno/mangolib/internal/ui"
	"github.com/mmrmagno/mangolib/internal/ytdlp"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	ui.Banner()
	fmt.Println()

	root := &cobra.Command{
		Use:           "mangolib",
		Short:         "Music library manager, downloader, and iPod sync tool",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		cmdInit(),
		cmdLs(),
		cmdImport(),
		cmdReorganize(),
		cmdGetCovers(),
		cmdSync(),
		cmdDownload(),
		cmdUpdate(),
	)

	if err := root.Execute(); err != nil {
		ui.Fatal(err.Error())
	}
}

func loadCfg() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		ui.Fatal("loading config: " + err.Error())
	}
	return cfg
}

func cmdInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Non-interactively import all music already in the library",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadCfg()
			return catalog.ScanAndTag(cfg)
		},
	}
}

func cmdLs() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List all tracks in the library",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadCfg()
			return catalog.ListTracks(cfg)
		},
	}
}

func cmdImport() *cobra.Command {
	return &cobra.Command{
		Use:   "import <path>",
		Short: "Import audio files from <path> into the library",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadCfg()
			return catalog.Import(cfg, args[0])
		},
	}
}

func cmdReorganize() *cobra.Command {
	return &cobra.Command{
		Use:   "reorganize",
		Short: "Reorganize existing library into Artist/Album/Track folder structure using embedded tags",
		RunE: func(cmd *cobra.Command, args []string) error {
			return catalog.ScanAndTag(loadCfg())
		},
	}
}

func cmdGetCovers() *cobra.Command {
	return &cobra.Command{
		Use:   "get-covers",
		Short: "Fetch and embed missing album art for every track in the library",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadCfg()
			runGetCovers(cfg)
			return nil
		},
	}
}

func runGetCovers(cfg *config.Config) {
	size := cfg.Covers.Size
	if size == 0 {
		size = 500
	}

	found, embedded, skipped := 0, 0, 0

	_ = filepath.Walk(cfg.MusicLibrary, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".mp3" && ext != ".m4a" && ext != ".flac" {
			return nil
		}
		found++

		if catalog.HasEmbeddedCover(path) {
			skipped++
			return nil
		}

		meta := catalog.ReadTags(path)
		if meta.Artist == "" && meta.Album == "" {
			return nil
		}

		art, mime, err := covers.Fetch(meta.Artist, meta.Album, size)
		if err != nil || art == nil {
			return nil
		}

		meta.CoverArt = art
		meta.CoverMime = mime

		switch ext {
		case ".mp3":
			err = catalog.WriteTagsMP3(path, meta)
		default:
			err = catalog.WriteTagsFFmpeg(path, meta)
		}
		if err != nil {
			ui.Warn(fmt.Sprintf("embed failed for %s: %v", filepath.Base(path), err))
			return nil
		}
		ui.Success(fmt.Sprintf("cover embedded: %s", filepath.Base(path)))
		embedded++
		return nil
	})

	ui.Success(fmt.Sprintf("%d tracks scanned — %d covers embedded, %d already had art", found, embedded, skipped))
}

func cmdSync() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync music library to iPod",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadCfg()
			return ipod.Sync(cfg)
		},
	}
}

func cmdDownload() *cobra.Command {
	var (
		flagSpotify bool
		flagYouTube bool
		flagTidal   bool
		flagAlbum   string
		flagArtist  string
	)

	cmd := &cobra.Command{
		Use:   "download [--spotify|--youtube|--tidal] [--album NAME] [--artist NAME] <URL>",
		Short: "Download music from Spotify, YouTube, or Tidal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			cfg := loadCfg()

			service := ""
			switch {
			case flagSpotify:
				service = download.ServiceSpotify
			case flagYouTube:
				service = download.ServiceYouTube
			case flagTidal:
				service = download.ServiceTidal
			default:
				var err error
				service, err = download.DetectService(url)
				if err != nil {
					return err
				}
			}

			var d download.Downloader
			switch service {
			case download.ServiceSpotify:
				d = download.Spotify{}
			case download.ServiceYouTube:
				d = download.YouTube{Album: flagAlbum, Artist: flagArtist}
			case download.ServiceTidal:
				d = download.Tidal{}
			}

			if err := d.Download(url, cfg); err != nil {
				return err
			}
			runGetCovers(cfg)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagSpotify, "spotify", false, "force Spotify download")
	cmd.Flags().BoolVar(&flagYouTube, "youtube", false, "force YouTube download")
	cmd.Flags().BoolVar(&flagTidal, "tidal", false, "force Tidal download")
	cmd.Flags().StringVar(&flagAlbum, "album", "", "override album name (YouTube only)")
	cmd.Flags().StringVar(&flagArtist, "artist", "", "override artist name (YouTube only)")

	return cmd
}

func cmdUpdate() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update yt-dlp to the latest release",
		RunE: func(cmd *cobra.Command, args []string) error {
			if v, err := ytdlp.Version(); err == nil {
				fmt.Printf("Current yt-dlp version: %s\n", v)
			}
			if err := ytdlp.Update(); err != nil {
				return err
			}
			if v, err := ytdlp.Version(); err == nil {
				ui.Success("yt-dlp updated to " + v)
			}
			return nil
		},
	}
}

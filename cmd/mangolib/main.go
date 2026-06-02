package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/mmrmagno/mangolib/internal/catalog"
	"github.com/mmrmagno/mangolib/internal/config"
	"github.com/mmrmagno/mangolib/internal/covers"
	"github.com/mmrmagno/mangolib/internal/download"
	"github.com/mmrmagno/mangolib/internal/ipod"
	"github.com/mmrmagno/mangolib/internal/ui"
	"github.com/mmrmagno/mangolib/internal/ytdlp"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	printBanner := func() {
		ui.Banner(version)
		fmt.Println()
	}

	root := &cobra.Command{
		Use:           "mangolib",
		Short:         "Music library manager, downloader, and iPod sync tool",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		// Show banner + help when running bare `mangolib` with no subcommand.
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
		// PersistentPreRun fires for subcommands. Skip for root — Help() handles it there.
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if cmd.HasParent() {
				printBanner()
			}
		},
	}

	// Override help to include the banner.
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		printBanner()
		cmd.Usage()
	})

	root.PersistentFlags().BoolVarP(&ui.Verbose, "verbose", "v", false, "show raw output from yt-dlp, ffmpeg, rsync")
	root.PersistentFlags().StringVar(&config.ConfigPath, "config", "", "path to config file (default ~/.config/mangolib/mangolib.toml)")

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

	// Catch SIGINT/SIGTERM and exit cleanly (Bubble Tea restores terminal on its own).
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, ui.Dim("cancelled"))
		os.Exit(130)
	}()

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

func cmdGetCovers() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "get-covers",
		Short: "Fetch and embed missing album art for every track in the library",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadCfg()
			runGetCovers(cfg, force)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "re-fetch and overwrite existing cover.jpg files")
	return cmd
}

func runGetCovers(cfg *config.Config, force bool) {
	size := cfg.Covers.Size
	if size == 0 {
		size = 500
	}

	found, extracted, fetched, skipped, coverFiles := 0, 0, 0, 0, 0

	_ = filepath.Walk(cfg.MusicLibrary, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".mp3" && ext != ".m4a" && ext != ".flac" {
			return nil
		}
		found++

		coverPath := filepath.Join(filepath.Dir(path), "cover.jpg")
		if !force {
			if _, err := os.Stat(coverPath); err == nil {
				skipped++
				return nil
			}
		}

		// If the file already has embedded art (Spotify, Tidal, manual), extract it
		// directly for cover.jpg — no iTunes fetch needed and no risky re-embed.
		if catalog.HasEmbeddedCover(path) {
			m := catalog.ReadTags(path)
			if len(m.CoverArt) > 0 {
				if err := catalog.WriteCoverFile(filepath.Dir(path), m.CoverArt, size, force); err == nil {
					coverFiles++
				}
			}
			extracted++
			return nil
		}

		// No embedded art — fetch from iTunes / MusicBrainz.
		meta := catalog.ReadTags(path)
		if meta.Artist == "" && meta.Album == "" {
			return nil
		}

		art, mime, err := covers.Fetch(meta.Artist, meta.Album)
		if err != nil || art == nil {
			ui.Warn(fmt.Sprintf("no cover found: %s - %s", meta.Artist, meta.Album))
			return nil
		}

		meta.CoverArt = art
		meta.CoverMime = mime

		var embedErr error
		switch ext {
		case ".mp3":
			embedErr = catalog.WriteTagsMP3(path, meta)
		default:
			embedErr = catalog.WriteTagsFFmpeg(path, meta)
		}
		if embedErr != nil {
			ui.Warn(fmt.Sprintf("cover embed failed for %s: %v", filepath.Base(path), embedErr))
		} else {
			ui.Success(fmt.Sprintf("cover embedded: %s", filepath.Base(path)))
			fetched++
		}
		if err := catalog.WriteCoverFile(filepath.Dir(path), art, size, true); err == nil {
			coverFiles++
		}
		return nil
	})

	ui.Success(fmt.Sprintf("%d tracks: %d covers fetched, %d extracted from tags, %d skipped, %d cover.jpg written", found, fetched, extracted, skipped, coverFiles))
}

func cmdSync() *cobra.Command {
	var fromIPod bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync music library to iPod (or use --from-ipod to pull from iPod to library)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadCfg()
			if fromIPod {
				if err := ui.RunWithSpinner("Syncing from iPod...", func() error {
					return ipod.SyncFrom(cfg, dryRun)
				}); err != nil {
					return err
				}
				if dryRun {
					ui.Success("Dry run complete — no files changed")
					return nil
				}
				if err := ui.RunWithSpinner("Reorganizing imported files...", func() error {
					return catalog.ScanAndTag(cfg)
				}); err != nil {
					return err
				}
				ui.Success("Import from iPod complete")
				return nil
			}
			if err := ui.RunWithSpinner("Syncing to iPod...", func() error {
				return ipod.Sync(cfg, dryRun)
			}); err != nil {
				return err
			}
			if dryRun {
				ui.Success("Dry run complete — no files changed")
			} else {
				ui.Success("Sync complete")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&fromIPod, "from-ipod", false, "pull music from iPod into the library and reorganize")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be synced without transferring files")
	return cmd
}

func cmdReorganize() *cobra.Command {
	var clean bool
	cmd := &cobra.Command{
		Use:   "reorganize",
		Short: "Reorganize library into Artist/Album/Track structure; use --clean to fix YouTube title noise",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadCfg()
			if clean {
				if err := ui.RunWithSpinner("Cleaning YouTube title noise...", func() error {
					return catalog.CleanLibraryTitles(cfg)
				}); err != nil {
					return err
				}
				ui.Success("Titles cleaned")
			}
			return ui.RunWithSpinner("Reorganizing library...", func() error {
				return catalog.ScanAndTag(cfg)
			})
		},
	}
	cmd.Flags().BoolVar(&clean, "clean", false, "strip YouTube title noise from all track titles before reorganizing")
	return cmd
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
			runGetCovers(cfg, false)
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

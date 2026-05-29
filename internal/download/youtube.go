package download

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mmrmagno/mangolib/internal/config"
	"github.com/mmrmagno/mangolib/internal/ui"
	"github.com/mmrmagno/mangolib/internal/ytdlp"
)

type YouTube struct {
	Album  string // overrides album tag (useful for single video downloads)
	Artist string // overrides artist/uploader tag
}

func (y YouTube) Download(url string, cfg *config.Config) error {
	if err := ytdlp.EnsureInstalled(); err != nil {
		return err
	}

	format := cfg.Download.AudioFormat
	if format == "" {
		format = "mp3"
	}
	quality := cfg.Download.AudioQuality
	if quality == "" {
		quality = "320"
	}

	isPlaylist := strings.Contains(url, "list=") || strings.Contains(url, "/playlist")

	artistSeg := "%(artist,uploader)s"
	if y.Artist != "" {
		artistSeg = y.Artist
	}
	albumSeg := "%(album,playlist_title,Unknown Album)s"
	if y.Album != "" {
		albumSeg = y.Album
	}
	outTmpl := filepath.Join(
		cfg.MusicLibrary,
		artistSeg,
		albumSeg,
		"%(playlist_index,track_number)02d. %(track,title)s.%(ext)s",
	)

	ui.Step(fmt.Sprintf("Downloading from YouTube: %s", url))

	args := []string{
		"-x",
		"--audio-format", format,
		"--audio-quality", quality + "k",
		"--embed-thumbnail",
		"--add-metadata",
		"--output", outTmpl,
	}
	if !isPlaylist {
		args = append(args, "--no-playlist")
	}
	args = append(args, url)

	if err := ytdlp.Run(args...); err != nil {
		return fmt.Errorf("yt-dlp: %w", err)
	}

	ui.Success("Download complete")
	return nil
}

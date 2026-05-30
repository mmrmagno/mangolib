package download

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mmrmagno/mangolib/internal/catalog"
	"github.com/mmrmagno/mangolib/internal/config"
	"github.com/mmrmagno/mangolib/internal/ui"
	"github.com/mmrmagno/mangolib/internal/ytdlp"
)

type YouTube struct {
	Album  string
	Artist string
}

func (y YouTube) Download(url string, cfg *config.Config) error {
	if err := ytdlp.EnsureInstalled(); err != nil {
		return err
	}
	isPlaylist := strings.Contains(url, "list=") || strings.Contains(url, "/playlist")
	if isPlaylist {
		return y.downloadPlaylist(url, cfg)
	}
	return y.downloadSingle(url, cfg)
}

// downloadSingle downloads one video using a tmpDir so we can rewrite tags and
// apply title cleaning before organizing — same flow as downloadPlaylist.
func (y YouTube) downloadSingle(url string, cfg *config.Config) error {
	format, quality := audioArgs(cfg)

	tmpDir, err := os.MkdirTemp("", "mangolib-yt-*")
	if err != nil {
		return fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	outTmpl := filepath.Join(tmpDir, "download.%(ext)s")
	args := []string{
		"-x", "--audio-format", format, "--audio-quality", quality,
		"--add-metadata", "--no-playlist",
		"--output", outTmpl, url,
	}

	if err := ui.RunWithSpinner("Downloading from YouTube...", func() error {
		return ytdlp.Run(args...)
	}); err != nil {
		return fmt.Errorf("yt-dlp: %w", err)
	}

	matches, _ := filepath.Glob(filepath.Join(tmpDir, "download.*"))
	if len(matches) == 0 {
		return fmt.Errorf("yt-dlp produced no output file")
	}
	downloaded := matches[0]
	ext := strings.ToLower(filepath.Ext(downloaded))

	existing := catalog.ReadTags(downloaded)

	title := catalog.CleanYouTubeTitle(existing.Title)
	artist := existing.Artist
	if y.Artist != "" {
		artist = y.Artist
	}
	if artist == "" {
		artist = "Unknown Artist"
	}
	album := existing.Album
	if y.Album != "" {
		album = y.Album
	}
	if album == "" {
		album = "Unknown Album"
	}

	meta := catalog.TrackMeta{
		Title:       title,
		Artist:      artist,
		Album:       album,
		AlbumArtist: artist,
		TrackNumber: existing.TrackNumber,
		Year:        existing.Year,
	}

	switch ext {
	case ".mp3":
		catalog.WriteTagsMP3(downloaded, meta)
	default:
		catalog.WriteTagsFFmpeg(downloaded, meta)
	}

	dest, err := catalog.OrganizeFile(downloaded, cfg.MusicLibrary, meta)
	if err != nil {
		return fmt.Errorf("organize: %w", err)
	}
	ui.Success(fmt.Sprintf("saved: %s", strings.TrimPrefix(dest, cfg.MusicLibrary+"/")))
	return nil
}

// downloadPlaylist enumerates the playlist first, then downloads each track
// individually — mirroring the Spotify flow for consistent UI and full tag control.
func (y YouTube) downloadPlaylist(url string, cfg *config.Config) error {
	var items []ytdlp.PlaylistItem

	if err := ui.RunWithSpinner("Fetching playlist info...", func() error {
		var err error
		items, err = ytdlp.FlatPlaylist(url)
		return err
	}); err != nil || len(items) == 0 {
		// Enumeration failed — fall back to a bulk download with a spinner.
		return y.downloadPlaylistBulk(url, cfg)
	}

	albumName := y.Album
	if albumName == "" {
		albumName = items[0].PlaylistTitle
	}

	format, quality := audioArgs(cfg)
	tp := ui.NewTrackProgress(albumName, len(items))

	tmpDir, err := os.MkdirTemp("", "mangolib-yt-*")
	if err != nil {
		tp.Done()
		return fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	failed := 0
	for i, item := range items {
		// Download to tmpDir named by video ID so we can find it reliably.
		outTmpl := filepath.Join(tmpDir, item.ID+".%(ext)s")
		videoURL := "https://www.youtube.com/watch?v=" + item.ID

		args := []string{
			"-x", "--audio-format", format, "--audio-quality", quality,
			"--add-metadata", "--no-playlist",
			"--output", outTmpl,
			videoURL,
		}

		if err := ytdlp.Run(args...); err != nil {
			ui.Warn(fmt.Sprintf("[%s] failed: %v", item.Title, err))
			failed++
			continue
		}

		matches, _ := filepath.Glob(filepath.Join(tmpDir, item.ID+".*"))
		if len(matches) == 0 {
			ui.Warn(fmt.Sprintf("no output file for: %s", item.Title))
			failed++
			continue
		}
		downloaded := matches[0]
		ext := strings.ToLower(filepath.Ext(downloaded))

		// Read yt-dlp's metadata (artist/uploader) then override with our values.
		existing := catalog.ReadTags(downloaded)

		// Always clean YouTube title noise — "(Official Video)", "Channel | " etc.
		// are never legitimate song titles.
		title := catalog.CleanYouTubeTitle(item.Title)

		artist := existing.Artist
		if y.Artist != "" {
			artist = y.Artist
		}
		if artist == "" {
			artist = "Unknown Artist"
		}

		meta := catalog.TrackMeta{
			Title:       title,
			Artist:      artist,
			Album:       albumName,
			AlbumArtist: artist,
			TrackNumber: i + 1,
			TrackTotal:  len(items),
		}

		switch ext {
		case ".mp3":
			catalog.WriteTagsMP3(downloaded, meta)
		default:
			catalog.WriteTagsFFmpeg(downloaded, meta)
		}

		if _, err := catalog.OrganizeFile(downloaded, cfg.MusicLibrary, meta); err != nil {
			ui.Warn(fmt.Sprintf("organize failed for %s: %v", title, err))
			failed++
			continue
		}

		tp.Track(title)
	}

	tp.Done()

	if failed == len(items) {
		return fmt.Errorf("all %d downloads failed", failed)
	}
	if failed > 0 {
		ui.Warn(fmt.Sprintf("%d/%d tracks failed", failed, len(items)))
	}
	ui.Success(fmt.Sprintf("Downloaded %d/%d tracks", len(items)-failed, len(items)))
	return nil
}

// downloadPlaylistBulk is the fallback when FlatPlaylist enumeration fails.
func (y YouTube) downloadPlaylistBulk(url string, cfg *config.Config) error {
	artistSeg := "%(artist,uploader)s"
	if y.Artist != "" {
		artistSeg = y.Artist
	}
	albumSeg := "%(album,playlist_title,Unknown Album)s"
	if y.Album != "" {
		albumSeg = y.Album
	}
	outTmpl := filepath.Join(cfg.MusicLibrary, artistSeg, albumSeg,
		"%(playlist_index,track_number)02d. %(track,title)s.%(ext)s")

	format, quality := audioArgs(cfg)
	args := []string{
		"-x", "--audio-format", format, "--audio-quality", quality,
		"--add-metadata", "--output", outTmpl, url,
	}

	if err := ui.RunWithSpinner("Downloading playlist...", func() error {
		return ytdlp.Run(args...)
	}); err != nil {
		return fmt.Errorf("yt-dlp: %w", err)
	}
	ui.Success("Download complete")
	return nil
}

func audioArgs(cfg *config.Config) (format, quality string) {
	format = cfg.Download.AudioFormat
	if format == "" {
		format = "mp3"
	}
	quality = cfg.Download.AudioQuality
	if quality == "" {
		quality = "320"
	}
	return format, quality + "k"
}


package download

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mmrmagno/mangolib/internal/catalog"
	"github.com/mmrmagno/mangolib/internal/config"
	"github.com/mmrmagno/mangolib/internal/streamrip"
	"github.com/mmrmagno/mangolib/internal/ui"
)

// Tidal downloads from Tidal by driving the streamrip `rip` CLI, then folds the
// results into the library via catalog.OrganizeFile. streamrip's tags (from the
// Tidal API) are trusted, so mangolib does not re-tag.
type Tidal struct{}

func (t Tidal) Download(url string, cfg *config.Config) error {
	url = normalizeTidalURL(url)
	if err := streamrip.EnsureInstalled(); err != nil {
		return err
	}

	cfgPath := streamrip.ConfigPath()
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return fmt.Errorf("creating streamrip config dir: %w", err)
	}

	if err := streamrip.EnsureConfig(cfgPath); err != nil {
		return err
	}

	authenticated := streamrip.IsAuthenticated(cfgPath)
	if !authenticated {
		ui.Step("Tidal login required — follow streamrip's prompt to authorize this device.")
	}

	tmp, err := os.MkdirTemp("", "mangolib-tidal-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)
	quality := streamrip.QualityArg(cfg.Tidal.Quality)

	// When not authenticated, streamrip needs a clean terminal for the OAuth device
	// code flow (shows a URL + polls interactively). Skip the spinner in that case.
	runDownload := func() error {
		return streamrip.Download(context.Background(), cfgPath, tmp, quality, url, authenticated)
	}
	var dlErr error
	if authenticated {
		dlErr = ui.RunWithSpinner("Downloading from Tidal...", runDownload)
	} else {
		dlErr = runDownload()
	}
	if dlErr != nil {
		return fmt.Errorf("streamrip download: %w", dlErr)
	}

	// Collect downloaded audio files before organizing so we know the total for
	// the progress bar — same pattern as Spotify album / YouTube playlist.
	var audioFiles []string
	_ = filepath.Walk(tmp, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && catalog.IsAudioFile(path) {
			audioFiles = append(audioFiles, path)
		}
		return nil
	})

	if len(audioFiles) == 0 {
		return fmt.Errorf("no audio files were downloaded from Tidal")
	}

	firstMeta := catalog.ReadTags(audioFiles[0])
	albumLabel := firstMeta.Album
	if albumLabel == "" {
		albumLabel = "Tidal"
	}

	var tp *ui.TrackProgress
	if len(audioFiles) == 1 {
		label := firstMeta.Title
		if firstMeta.Artist != "" && firstMeta.Title != "" {
			label = firstMeta.Artist + " — " + firstMeta.Title
		}
		ui.Step(label)
	} else {
		tp = ui.NewTrackProgress(albumLabel, len(audioFiles))
	}

	failed := 0
	for _, path := range audioFiles {
		meta := catalog.ReadTags(path)
		if _, err := catalog.OrganizeFile(path, cfg.MusicLibrary, meta); err != nil {
			ui.Warn(fmt.Sprintf("organize failed: %s: %v", filepath.Base(path), err))
			failed++
			continue
		}
		if tp != nil {
			tp.Track(meta.Title)
		}
	}
	tp.Done()

	imported := len(audioFiles) - failed
	if imported == 0 {
		return fmt.Errorf("no audio files could be organized from Tidal download")
	}
	if failed > 0 {
		ui.Warn(fmt.Sprintf("%d file(s) failed to organize", failed))
	}
	ui.Success(fmt.Sprintf("Downloaded %d track(s) from Tidal", imported))
	return nil
}

// tidalURLRe matches the resource type and id in any Tidal URL form, ignoring
// subdomains (listen.), "browse/" prefixes, trailing path segments, and queries.
var tidalURLRe = regexp.MustCompile(`(?i)(track|album|playlist|artist|video|mix)/([A-Za-z0-9-]+)`)

// normalizeTidalURL rebuilds a canonical https://tidal.com/<type>/<id> URL.
// streamrip's URL parser mis-handles trailing segments (e.g. ".../track/123/u"
// is read as id "u"), so we strip everything but the resource type and id.
// Returns the input unchanged if no Tidal resource is recognized.
func normalizeTidalURL(raw string) string {
	m := tidalURLRe.FindStringSubmatch(raw)
	if m == nil {
		return raw
	}
	return fmt.Sprintf("https://tidal.com/%s/%s", strings.ToLower(m[1]), m[2])
}

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

	imported, failed := 0, 0
	walkErr := filepath.Walk(tmp, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !catalog.IsAudioFile(path) {
			return nil
		}
		meta := catalog.ReadTags(path)
		dest, err := catalog.OrganizeFile(path, cfg.MusicLibrary, meta)
		if err != nil {
			ui.Warn(fmt.Sprintf("organize failed: %s: %v", filepath.Base(path), err))
			failed++
			return nil
		}
		ui.Success("imported: " + strings.TrimPrefix(dest, cfg.MusicLibrary+"/"))
		imported++
		return nil
	})
	if walkErr != nil {
		return walkErr
	}

	if imported == 0 {
		return fmt.Errorf("no audio files were downloaded from Tidal")
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

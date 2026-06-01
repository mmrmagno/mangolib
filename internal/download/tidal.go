package download

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

	if !streamrip.IsAuthenticated(cfgPath) {
		ui.Step("Tidal login required — follow streamrip's prompt to authorize this device.")
	}

	tmp, err := os.MkdirTemp("", "mangolib-tidal-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	quality := streamrip.QualityArg(cfg.Tidal.Quality)
	ui.Step("Downloading from Tidal...")
	if err := streamrip.Download(context.Background(), cfgPath, tmp, quality, url); err != nil {
		return fmt.Errorf("streamrip download: %w", err)
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

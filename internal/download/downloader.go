package download

import (
	"fmt"
	"strings"

	"github.com/mmrmagno/mangolib/internal/config"
)

// Service constants matched by URL pattern.
const (
	ServiceSpotify  = "spotify"
	ServiceYouTube  = "youtube"
	ServiceTidal    = "tidal"
)

// Downloader is implemented by each service-specific package.
type Downloader interface {
	Download(url string, cfg *config.Config) error
}

// DetectService infers the streaming service from a URL.
func DetectService(url string) (string, error) {
	switch {
	case strings.Contains(url, "spotify.com"):
		return ServiceSpotify, nil
	case strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be"):
		return ServiceYouTube, nil
	case strings.Contains(url, "tidal.com"):
		return ServiceTidal, nil
	default:
		return "", fmt.Errorf("cannot detect service from URL %q — use --spotify, --youtube, or --tidal", url)
	}
}

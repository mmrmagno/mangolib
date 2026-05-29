package download

import (
	"fmt"

	"github.com/mmrmagno/mangolib/internal/config"
	"github.com/mmrmagno/mangolib/internal/ui"
)

// Tidal downloader — full implementation planned for v1.0.0.
// Requires PKCE OAuth2 flow against the unofficial Tidal API.
type Tidal struct{}

func (t Tidal) Download(url string, cfg *config.Config) error {
	ui.Warn("Tidal download is not yet implemented — coming in v1.0.0.")
	return fmt.Errorf("tidal download not yet implemented")
}

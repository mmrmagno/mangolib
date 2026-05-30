package covers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const userAgent = "mangolib/0.1.0 (https://github.com/mmrmagno/mangolib)"

// Fetch returns album art bytes for the given artist+album.
// Tries iTunes Search API first, then MusicBrainz / Cover Art Archive.
// Returns nil, nil, nil if no art is found anywhere.
func Fetch(artist, album string) ([]byte, string, error) {
	if data, mime, err := fetchITunes(artist, album); err == nil && data != nil {
		return data, mime, nil
	}
	if data, mime, err := fetchMusicBrainz(artist, album); err == nil && data != nil {
		return data, mime, nil
	}
	return nil, "", nil
}

// --- iTunes Search API ---

func fetchITunes(artist, album string) ([]byte, string, error) {
	query := url.QueryEscape(artist + " " + album)
	apiURL := fmt.Sprintf("https://itunes.apple.com/search?term=%s&entity=album&limit=5", query)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	var result struct {
		Results []struct {
			ArtworkUrl100 string `json:"artworkUrl100"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", err
	}

	for _, r := range result.Results {
		if r.ArtworkUrl100 == "" {
			continue
		}
		// Request the largest available size from Apple's CDN; ffmpeg will downscale.
		imgURL := strings.Replace(r.ArtworkUrl100, "100x100", "3000x3000", 1)
		if data, mime, err := fetchImage(imgURL); err == nil && data != nil {
			return data, mime, nil
		}
	}
	return nil, "", fmt.Errorf("no iTunes results")
}

// --- MusicBrainz + Cover Art Archive ---

func fetchMusicBrainz(artist, album string) ([]byte, string, error) {
	// Search release-groups (canonical per-album entity, not per-edition).
	// MusicBrainz requires rate-limiting to 1 req/sec for anonymous access.
	time.Sleep(1 * time.Second)

	query := url.QueryEscape(fmt.Sprintf(`artist:"%s" AND releasegroup:"%s"`, artist, album))
	mbURL := fmt.Sprintf("https://musicbrainz.org/ws/2/release-group/?query=%s&fmt=json&limit=5", query)

	req, _ := http.NewRequest("GET", mbURL, nil)
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	var mbResult struct {
		ReleaseGroups []struct {
			ID string `json:"id"`
		} `json:"release-groups"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&mbResult); err != nil || len(mbResult.ReleaseGroups) == 0 {
		return nil, "", fmt.Errorf("no MusicBrainz results")
	}

	for _, rg := range mbResult.ReleaseGroups {
		imgURL := fmt.Sprintf("https://coverartarchive.org/release-group/%s/front", rg.ID)
		if data, mime, err := fetchImage(imgURL); err == nil && data != nil {
			return data, mime, nil
		}
	}
	return nil, "", fmt.Errorf("no cover art in Cover Art Archive")
}

// --- helpers ---

func fetchImage(imgURL string) ([]byte, string, error) {
	req, _ := http.NewRequest("GET", imgURL, nil)
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	mime := resp.Header.Get("Content-Type")
	if mime == "" {
		mime = "image/jpeg"
	}
	return data, mime, nil
}

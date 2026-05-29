package covers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Fetch returns album art bytes for the given artist+album.
// Tries iTunes Search API first (more reliable), then Cover Art Archive.
// Returns nil, nil if no art is found (not an error).
func Fetch(artist, album string, size int) ([]byte, string, error) {
	if data, mime, err := fetchITunes(artist, album, size); err == nil && data != nil {
		return data, mime, nil
	}
	if data, mime, err := fetchCoverArtArchive(artist, album); err == nil && data != nil {
		return data, mime, nil
	}
	return nil, "", nil
}

// --- iTunes Search API ---

func fetchITunes(artist, album string, size int) ([]byte, string, error) {
	query := url.QueryEscape(artist + " " + album)
	apiURL := fmt.Sprintf("https://itunes.apple.com/search?term=%s&entity=album&limit=5", query)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	var result struct {
		Results []struct {
			ArtistName  string `json:"artistName"`
			CollectionName string `json:"collectionName"`
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
		// Scale the artwork URL to requested size.
		imgURL := strings.Replace(r.ArtworkUrl100, "100x100", fmt.Sprintf("%dx%d", size, size), 1)
		data, mime, err := fetchImage(imgURL)
		if err == nil && data != nil {
			return data, mime, nil
		}
	}
	return nil, "", nil
}

// --- Cover Art Archive (MusicBrainz) ---

func fetchCoverArtArchive(artist, album string) ([]byte, string, error) {
	query := url.QueryEscape(fmt.Sprintf("artist:%s release:%s", artist, album))
	mbURL := fmt.Sprintf("https://musicbrainz.org/ws/2/release/?query=%s&fmt=json&limit=3", query)

	req, _ := http.NewRequest("GET", mbURL, nil)
	req.Header.Set("User-Agent", "mangolib/1.0 (https://github.com/mmrmagno/mangolib)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	var mbResult struct {
		Releases []struct {
			ID string `json:"id"`
		} `json:"releases"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&mbResult); err != nil || len(mbResult.Releases) == 0 {
		return nil, "", fmt.Errorf("no MusicBrainz results")
	}

	for _, rel := range mbResult.Releases {
		imgURL := fmt.Sprintf("https://coverartarchive.org/release/%s/front", rel.ID)
		data, mime, err := fetchImage(imgURL)
		if err == nil && data != nil {
			return data, mime, nil
		}
	}
	return nil, "", nil
}

// --- helpers ---

func fetchImage(imgURL string) ([]byte, string, error) {
	resp, err := http.Get(imgURL)
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

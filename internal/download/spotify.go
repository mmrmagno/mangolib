package download

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mmrmagno/mangolib/internal/config"
	"github.com/mmrmagno/mangolib/internal/catalog"
	"github.com/mmrmagno/mangolib/internal/ui"
	"github.com/mmrmagno/mangolib/internal/ytdlp"
)

type Spotify struct{}

type spotifyTrack struct {
	Name   string `json:"name"`
	Artists []struct {
		Name string `json:"name"`
	} `json:"artists"`
	Album struct {
		Name        string `json:"name"`
		ReleaseDate string `json:"release_date"`
	} `json:"album"`
	TrackNumber int `json:"track_number"`
}

func (s Spotify) Download(rawURL string, cfg *config.Config) error {
	if cfg.Spotify.ClientID == "" || cfg.Spotify.ClientSecret == "" {
		return fmt.Errorf("Spotify credentials not set — add client_id and client_secret to mangolib.toml\n" +
			"Get them at https://developer.spotify.com/dashboard (free account, no website needed)")
	}

	if err := ytdlp.EnsureInstalled(); err != nil {
		return err
	}

	token, err := spotifyToken(cfg.Spotify.ClientID, cfg.Spotify.ClientSecret)
	if err != nil {
		return fmt.Errorf("Spotify auth: %w", err)
	}

	tracks, err := resolveTracks(rawURL, token)
	if err != nil {
		return fmt.Errorf("resolving Spotify URL: %w", err)
	}

	// Single track: spinner. Multiple tracks: progress bar.
	albumLabel := "Downloading"
	if len(tracks) > 0 && tracks[0].Album.Name != "" {
		albumLabel = tracks[0].Album.Name
	}
	var tp *ui.TrackProgress
	if len(tracks) == 1 {
		artist := ""
		if len(tracks[0].Artists) > 0 {
			artist = tracks[0].Artists[0].Name
		}
		ui.Step(fmt.Sprintf("%s — %s", artist, tracks[0].Name))
	} else {
		tp = ui.NewTrackProgress(albumLabel, len(tracks))
	}

	tmpDir, err := os.MkdirTemp("", "mangolib-spotify-*")
	if err != nil {
		tp.Done()
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	format := cfg.Download.AudioFormat
	if format == "" {
		format = "mp3"
	}
	quality := cfg.Download.AudioQuality
	if quality == "" {
		quality = "320"
	}

	failed := 0
	for i, t := range tracks {
		_ = i
		artist := ""
		if len(t.Artists) > 0 {
			artist = t.Artists[0].Name
		}

		// yt-dlp output template — the %% escapes the yt-dlp %(ext)s placeholder.
		outTmpl := filepath.Join(tmpDir, fmt.Sprintf("%02d. %s - %s.%%(ext)s",
			t.TrackNumber, sanitize(artist), sanitize(t.Name)))

		args := []string{
			"--no-playlist",
			"-x",
			"--audio-format", format,
			"--audio-quality", quality + "k",
			"--embed-thumbnail",
			"--output", outTmpl,
			fmt.Sprintf("ytsearch1:%s %s audio", artist, t.Name),
		}

		if err := ytdlp.Run(args...); err != nil {
			ui.Warn(fmt.Sprintf("[%s] download failed: %v", t.Name, err))
			failed++
			continue
		}

		// Find the file yt-dlp actually created (extension may vary).
		pattern := filepath.Join(tmpDir, fmt.Sprintf("%02d. %s - %s.*",
			t.TrackNumber, sanitize(artist), sanitize(t.Name)))
		matches, _ := filepath.Glob(pattern)
		if len(matches) == 0 {
			ui.Warn(fmt.Sprintf("no output file found for %q", t.Name))
			failed++
			continue
		}
		downloaded := matches[0]

		// Build metadata from what Spotify told us — authoritative, not yt-dlp guesses.
		year := ""
		if len(t.Album.ReleaseDate) >= 4 {
			year = t.Album.ReleaseDate[:4]
		}
		meta := catalog.TrackMeta{
			Title:       t.Name,
			Artist:      artist,
			Album:       t.Album.Name,
			AlbumArtist: artist,
			Year:        year,
			TrackNumber: t.TrackNumber,
			TrackTotal:  len(tracks),
		}

		// Write Spotify metadata into the file, overriding YouTube's.
		switch strings.ToLower(filepath.Ext(downloaded)) {
		case ".mp3":
			if err := catalog.WriteTagsMP3(downloaded, meta); err != nil {
				ui.Warn(fmt.Sprintf("tagging failed for %q: %v", t.Name, err))
			}
		default:
			if err := catalog.WriteTagsFFmpeg(downloaded, meta); err != nil {
				ui.Warn(fmt.Sprintf("tagging failed for %q: %v", t.Name, err))
			}
		}

		// Move into MusicLibrary/Artist/Album/NN. Title.ext
		if _, err := catalog.OrganizeFile(downloaded, cfg.MusicLibrary, meta); err != nil {
			ui.Warn(fmt.Sprintf("organize failed for %q: %v", t.Name, err))
			failed++
			continue
		}
		tp.Track(t.Name)
	}

	tp.Done()

	if failed == len(tracks) {
		return fmt.Errorf("all %d downloads failed", failed)
	}
	if failed > 0 {
		ui.Warn(fmt.Sprintf("%d/%d tracks failed", failed, len(tracks)))
	}
	ui.Success(fmt.Sprintf("Downloaded %d/%d tracks", len(tracks)-failed, len(tracks)))
	return nil
}

// --- Spotify API helpers ---

func spotifyToken(clientID, clientSecret string) (string, error) {
	creds := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	req, _ := http.NewRequest("POST", "https://accounts.spotify.com/api/token",
		strings.NewReader("grant_type=client_credentials"))
	req.Header.Set("Authorization", "Basic "+creds)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()

	var tok struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(r.Body).Decode(&tok); err != nil {
		return "", err
	}
	if tok.Error != "" {
		return "", fmt.Errorf("Spotify returned error: %s", tok.Error)
	}
	return tok.AccessToken, nil
}

var spotifyIDRe = regexp.MustCompile(`/(track|album|playlist)/([A-Za-z0-9]+)`)

func resolveTracks(rawURL, token string) ([]spotifyTrack, error) {
	m := spotifyIDRe.FindStringSubmatch(rawURL)
	if m == nil {
		return nil, fmt.Errorf("cannot parse Spotify URL: %s", rawURL)
	}
	kind, id := m[1], m[2]

	switch kind {
	case "track":
		t, err := fetchTrack(id, token)
		if err != nil {
			return nil, err
		}
		return []spotifyTrack{*t}, nil

	case "album":
		return fetchAlbumTracks(id, token)

	case "playlist":
		return fetchPlaylistTracks(id, token)
	}
	return nil, fmt.Errorf("unsupported Spotify URL type: %s", kind)
}

func spotifyGet(path, token string, out any) error {
	req, _ := http.NewRequest("GET", "https://api.spotify.com/v1/"+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Spotify API %s → HTTP %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func fetchTrack(id, token string) (*spotifyTrack, error) {
	var t spotifyTrack
	return &t, spotifyGet("tracks/"+id, token, &t)
}

func fetchAlbumTracks(id, token string) ([]spotifyTrack, error) {
	var album struct {
		Name        string `json:"name"`
		ReleaseDate string `json:"release_date"`
		Artists     []struct{ Name string `json:"name"` } `json:"artists"`
		Tracks      struct {
			Items []spotifyTrack `json:"items"`
		} `json:"tracks"`
	}
	if err := spotifyGet("albums/"+id, token, &album); err != nil {
		return nil, err
	}
	// Enrich each track with album metadata (simple tracks endpoint lacks it)
	for i := range album.Tracks.Items {
		album.Tracks.Items[i].Album.Name = album.Name
		album.Tracks.Items[i].Album.ReleaseDate = album.ReleaseDate
		if len(album.Tracks.Items[i].Artists) == 0 {
			album.Tracks.Items[i].Artists = album.Artists
		}
	}
	return album.Tracks.Items, nil
}

func fetchPlaylistTracks(id, token string) ([]spotifyTrack, error) {
	var result []spotifyTrack
	path := fmt.Sprintf("playlists/%s/tracks?limit=100", id)
	for path != "" {
		var page struct {
			Items []struct {
				Track spotifyTrack `json:"track"`
			} `json:"items"`
			Next string `json:"next"`
		}
		if err := spotifyGet(path, token, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			result = append(result, item.Track)
		}
		// next is a full URL; strip the base for our helper
		if page.Next != "" {
			path = strings.TrimPrefix(page.Next, "https://api.spotify.com/v1/")
		} else {
			path = ""
		}
	}
	return result, nil
}

var unsafeChars = regexp.MustCompile(`[<>:"/\\|?*]`)

func sanitize(s string) string {
	return strings.TrimSpace(unsafeChars.ReplaceAllString(s, "_"))
}

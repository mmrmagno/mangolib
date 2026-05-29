package ytdlp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mmrmagno/mangolib/internal/ui"
)

// BinPath returns the path where mangolib stores its own yt-dlp binary.
func BinPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin", "yt-dlp")
}

// Locate finds yt-dlp: first at BinPath(), then anywhere on PATH.
// Returns the resolved path or an error if not found.
func Locate() (string, error) {
	managed := BinPath()
	if _, err := os.Stat(managed); err == nil {
		return managed, nil
	}
	if path, err := exec.LookPath("yt-dlp"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("yt-dlp not found — run: mangolib update")
}

// EnsureInstalled installs yt-dlp if it is not already present.
func EnsureInstalled() error {
	if _, err := Locate(); err == nil {
		return nil
	}
	ui.Step("yt-dlp not found — downloading latest release...")
	return download()
}

// Update installs the latest yt-dlp release unconditionally.
func Update() error {
	ui.Step("Updating yt-dlp...")
	return download()
}

// Run executes yt-dlp with the given arguments, streaming output to the terminal.
func Run(args ...string) error {
	bin, err := Locate()
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// Version returns the installed yt-dlp version string.
func Version() (string, error) {
	bin, err := Locate()
	if err != nil {
		return "", err
	}
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// --- internal ---

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func assetName() string {
	switch runtime.GOOS {
	case "darwin":
		return "yt-dlp_macos"
	case "linux":
		if runtime.GOARCH == "arm64" {
			return "yt-dlp_linux_aarch64"
		}
		return "yt-dlp_linux"
	default:
		return "yt-dlp"
	}
}

func download() error {
	rel, err := latestRelease()
	if err != nil {
		return fmt.Errorf("fetching yt-dlp release info: %w", err)
	}

	want := assetName()
	var url string
	for _, a := range rel.Assets {
		if a.Name == want {
			url = a.BrowserDownloadURL
			break
		}
	}
	if url == "" {
		return fmt.Errorf("no yt-dlp asset %q in release %s", want, rel.TagName)
	}

	dest := BinPath()
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	ui.Step(fmt.Sprintf("Downloading yt-dlp %s → %s", rel.TagName, dest))
	if err := downloadFile(url, dest); err != nil {
		return fmt.Errorf("downloading yt-dlp: %w", err)
	}
	if err := os.Chmod(dest, 0755); err != nil {
		return err
	}
	ui.Success("yt-dlp installed at " + dest)
	return nil
}

func latestRelease() (*ghRelease, error) {
	resp, err := http.Get("https://api.github.com/repos/yt-dlp/yt-dlp/releases/latest")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

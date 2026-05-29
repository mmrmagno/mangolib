package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mmrmagno/mangolib/internal/config"
	"github.com/mmrmagno/mangolib/internal/ui"
)

var audioExts = map[string]bool{
	".mp3": true, ".m4a": true, ".flac": true,
	".ogg": true, ".opus": true, ".aac": true,
}

// Import moves audio files from srcDir into the music library.
func Import(cfg *config.Config, srcDir string) error {
	if err := os.MkdirAll(cfg.MusicLibrary, 0755); err != nil {
		return fmt.Errorf("creating music library dir: %w", err)
	}
	return importNative(cfg, srcDir)
}

// ScanAndTag walks the library and re-organizes any file not already in the
// correct Artist/Album/NN. Title.ext location. Used by `mangolib init`.
func ScanAndTag(cfg *config.Config) error {
	ui.Step("Scanning library...")
	moved, skipped, failed := 0, 0, 0

	err := filepath.Walk(cfg.MusicLibrary, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !audioExts[ext] {
			return nil
		}

		meta := ReadTags(path)
		artist := sanitizePath(meta.Artist)
		if artist == "" {
			artist = "Unknown Artist"
		}
		album := sanitizePath(meta.Album)
		if album == "" {
			album = "Unknown Album"
		}
		title := sanitizePath(meta.Title)
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(path), ext)
		}
		var filename string
		if meta.TrackNumber > 0 {
			filename = fmt.Sprintf("%02d. %s%s", meta.TrackNumber, title, ext)
		} else {
			filename = title + ext
		}
		expected := filepath.Join(cfg.MusicLibrary, artist, album, filename)
		if path == expected {
			skipped++
			return nil
		}

		dest, err := OrganizeFile(path, cfg.MusicLibrary, meta)
		if err != nil {
			ui.Warn(fmt.Sprintf("failed to move %s: %v", filepath.Base(path), err))
			failed++
			return nil
		}
		ui.Success(fmt.Sprintf("%s → %s", filepath.Base(path), strings.TrimPrefix(dest, cfg.MusicLibrary+"/")))
		moved++
		return nil
	})
	if err != nil {
		return err
	}

	_ = RemoveEmptyDirs(cfg.MusicLibrary)
	ui.Success(fmt.Sprintf("done — %d moved, %d already correct, %d failed", moved, skipped, failed))
	return nil
}

// ListTracks prints all tracks in the library as Artist — Album — Title.
func ListTracks(cfg *config.Config) error {
	return walkAndPrint(cfg.MusicLibrary)
}

// --- internal ---

func importNative(cfg *config.Config, srcDir string) error {
	ui.Step("Importing files into library...")
	imported := 0

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !audioExts[ext] {
			return nil
		}

		meta := ReadTags(path)

		artist := sanitizePath(meta.Artist)
		if artist == "" {
			artist = "Unknown Artist"
		}
		album := sanitizePath(meta.Album)
		if album == "" {
			album = "Unknown Album"
		}
		title := sanitizePath(meta.Title)
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(path), ext)
		}
		var filename string
		if meta.TrackNumber > 0 {
			filename = fmt.Sprintf("%02d. %s%s", meta.TrackNumber, title, ext)
		} else {
			filename = title + ext
		}
		dest := filepath.Join(cfg.MusicLibrary, artist, album, filename)

		if _, err := os.Stat(dest); err == nil {
			if cfg.Library.DuplicateAction == "skip" {
				ui.Step(fmt.Sprintf("skip (exists): %s", filename))
				return nil
			}
		}

		destPath, err := OrganizeFile(path, cfg.MusicLibrary, meta)
		if err != nil {
			ui.Warn(fmt.Sprintf("failed to import %s: %v", filepath.Base(path), err))
			return nil
		}
		ui.Success(fmt.Sprintf("imported: %s", strings.TrimPrefix(destPath, cfg.MusicLibrary+"/")))
		imported++
		return nil
	})

	if err != nil {
		return err
	}
	ui.Success(fmt.Sprintf("%d file(s) imported", imported))
	return nil
}

func walkAndPrint(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !audioExts[ext] {
			return nil
		}
		m := ReadTags(path)
		artist := m.Artist
		if artist == "" {
			artist = "Unknown Artist"
		}
		album := m.Album
		if album == "" {
			album = "Unknown Album"
		}
		title := m.Title
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(path), ext)
		}
		fmt.Printf("%s — %s — %s\n", artist, album, title)
		return nil
	})
}

// RemoveEmptyDirs removes all empty subdirectories under root (bottom-up).
func RemoveEmptyDirs(root string) error {
	return removeEmptyDirs(root)
}

func removeEmptyDirs(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		child := filepath.Join(root, e.Name())
		if err := removeEmptyDirs(child); err != nil {
			return err
		}
		children, err := os.ReadDir(child)
		if err == nil && len(children) == 0 {
			os.Remove(child)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	buf := make([]byte, 32*1024)
	for {
		n, err := in.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
		}
		if err != nil {
			break
		}
	}
	return nil
}

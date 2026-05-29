package catalog

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bogem/id3v2/v2"
	"github.com/dhowden/tag"
)

// TrackMeta holds the metadata we care about for organizing and tagging.
type TrackMeta struct {
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	Year        string
	TrackNumber int
	TrackTotal  int
	CoverArt    []byte
	CoverMime   string
}

// ReadTags extracts embedded metadata from an audio file.
// Returns a best-effort result; missing fields are empty strings/zeros.
func ReadTags(path string) TrackMeta {
	f, err := os.Open(path)
	if err != nil {
		return TrackMeta{}
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return TrackMeta{}
	}

	track, _ := m.Track()
	meta := TrackMeta{
		Title:  m.Title(),
		Artist: m.Artist(),
		Album:  m.Album(),
		Year:   fmt.Sprintf("%d", m.Year()),
	}
	if meta.Year == "0" {
		meta.Year = ""
	}
	meta.TrackNumber = track

	if p := m.Picture(); p != nil {
		meta.CoverArt = p.Data
		meta.CoverMime = p.MIMEType
	}
	return meta
}

// HasEmbeddedCover returns true if the file already has album art embedded.
func HasEmbeddedCover(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	m, err := tag.ReadFrom(f)
	if err != nil {
		return false
	}
	return m.Picture() != nil
}

// OrganizeFile moves src into libraryRoot/Artist/Album/NN. Title.ext.
// Returns the destination path. Uses embedded tags; falls back to filename.
func OrganizeFile(src, libraryRoot string, meta TrackMeta) (string, error) {
	ext := strings.ToLower(filepath.Ext(src))

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
		title = strings.TrimSuffix(filepath.Base(src), ext)
	}

	var filename string
	if meta.TrackNumber > 0 {
		filename = fmt.Sprintf("%02d. %s%s", meta.TrackNumber, title, ext)
	} else {
		filename = title + ext
	}

	destDir := filepath.Join(libraryRoot, artist, album)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}

	dest := filepath.Join(destDir, filename)
	if _, err := os.Stat(dest); err == nil {
		return dest, nil // already exists
	}

	if err := os.Rename(src, dest); err != nil {
		if err2 := copyFile(src, dest); err2 != nil {
			return "", err2
		}
		os.Remove(src)
	}
	return dest, nil
}

// WriteTagsMP3 writes metadata into an MP3 file using ID3v2.
func WriteTagsMP3(path string, meta TrackMeta) error {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return err
	}
	defer tag.Close()

	if meta.Title != "" {
		tag.SetTitle(meta.Title)
	}
	if meta.Artist != "" {
		tag.SetArtist(meta.Artist)
	}
	if meta.Album != "" {
		tag.SetAlbum(meta.Album)
	}
	if meta.Year != "" {
		tag.SetYear(meta.Year)
	}
	if meta.TrackNumber > 0 {
		trackStr := fmt.Sprintf("%d", meta.TrackNumber)
		if meta.TrackTotal > 0 {
			trackStr = fmt.Sprintf("%d/%d", meta.TrackNumber, meta.TrackTotal)
		}
		tag.AddTextFrame(tag.CommonID("Track number/Position in set"),
			id3v2.EncodingUTF8, trackStr)
	}
	if len(meta.CoverArt) > 0 {
		mime := meta.CoverMime
		if mime == "" {
			mime = "image/jpeg"
		}
		pic := id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    mime,
			PictureType: id3v2.PTFrontCover,
			Description: "Cover",
			Picture:     meta.CoverArt,
		}
		tag.DeleteFrames(tag.CommonID("Attached picture"))
		tag.AddAttachedPicture(pic)
	}

	return tag.Save()
}

// WriteTagsFFmpeg writes metadata into an M4A/FLAC file via ffmpeg.
// If meta.CoverArt is set, it is embedded as attached art.
func WriteTagsFFmpeg(path string, meta TrackMeta) error {
	tmp := path + ".tmp"

	var args []string
	if len(meta.CoverArt) > 0 {
		// Write cover art to a temp file so ffmpeg can read it as a second input.
		coverTmp, err := os.CreateTemp("", "mangolib-cover-*")
		if err != nil {
			return fmt.Errorf("creating cover temp file: %w", err)
		}
		coverPath := coverTmp.Name()
		defer os.Remove(coverPath)
		if _, err := coverTmp.Write(meta.CoverArt); err != nil {
			coverTmp.Close()
			return fmt.Errorf("writing cover temp file: %w", err)
		}
		coverTmp.Close()

		args = []string{"-y", "-i", path, "-i", coverPath,
			"-map", "0", "-map", "1",
			"-disposition:v:0", "attached_pic",
		}
	} else {
		args = []string{"-y", "-i", path}
	}

	if meta.Title != "" {
		args = append(args, "-metadata", "title="+meta.Title)
	}
	if meta.Artist != "" {
		args = append(args, "-metadata", "artist="+meta.Artist)
	}
	if meta.Album != "" {
		args = append(args, "-metadata", "album="+meta.Album)
	}
	if meta.Year != "" {
		args = append(args, "-metadata", "date="+meta.Year)
	}
	if meta.TrackNumber > 0 {
		track := fmt.Sprintf("%d", meta.TrackNumber)
		if meta.TrackTotal > 0 {
			track = fmt.Sprintf("%d/%d", meta.TrackNumber, meta.TrackTotal)
		}
		args = append(args, "-metadata", "track="+track)
	}
	args = append(args, "-codec", "copy", tmp)

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

var unsafePathChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

func sanitizePath(s string) string {
	return SanitizePathPublic(s)
}

// SanitizePathPublic is the exported version for use in other packages.
func SanitizePathPublic(s string) string {
	s = unsafePathChars.ReplaceAllString(s, "_")
	s = strings.TrimSpace(s)
	s = strings.Trim(s, ".")
	return s
}

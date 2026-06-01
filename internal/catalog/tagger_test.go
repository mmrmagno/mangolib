package catalog

import "testing"

func TestIsAudioFile(t *testing.T) {
	yes := []string{"a.mp3", "/x/y/b.FLAC", "song.m4a", "t.opus"}
	no := []string{"cover.jpg", "notes.txt", "noext", "a.mp3.part"}
	for _, p := range yes {
		if !IsAudioFile(p) {
			t.Errorf("IsAudioFile(%q) = false, want true", p)
		}
	}
	for _, p := range no {
		if IsAudioFile(p) {
			t.Errorf("IsAudioFile(%q) = true, want false", p)
		}
	}
}

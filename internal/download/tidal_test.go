package download

import "testing"

func TestNormalizeTidalURL(t *testing.T) {
	cases := map[string]string{
		"https://tidal.com/track/139189447/u":       "https://tidal.com/track/139189447",
		"https://tidal.com/track/139189447":         "https://tidal.com/track/139189447",
		"https://tidal.com/browse/track/139189447":  "https://tidal.com/track/139189447",
		"https://listen.tidal.com/album/12345":      "https://tidal.com/album/12345",
		"https://tidal.com/album/12345?foo=bar":     "https://tidal.com/album/12345",
		"https://tidal.com/playlist/abcd-12ef-3456": "https://tidal.com/playlist/abcd-12ef-3456",
		"https://tidal.com/artist/987/u":            "https://tidal.com/artist/987",
		"not a tidal resource at all":               "not a tidal resource at all",
	}
	for in, want := range cases {
		if got := normalizeTidalURL(in); got != want {
			t.Errorf("normalizeTidalURL(%q) = %q, want %q", in, got, want)
		}
	}
}

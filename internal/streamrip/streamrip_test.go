package streamrip

import (
	"reflect"
	"strings"
	"testing"
)

func TestQualityArg(t *testing.T) {
	cases := map[string]string{
		"LOW":              "0",
		"HIGH":             "1",
		"LOSSLESS":         "2",
		"HI_RES":           "3",
		"HI_RES_LOSSLESS":  "3",
		"hi_res_lossless ": "3",
		"":                 "3",
		"garbage":          "3",
	}
	for in, want := range cases {
		if got := QualityArg(in); got != want {
			t.Errorf("QualityArg(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildRipArgs(t *testing.T) {
	got := buildRipArgs("/cfg/streamrip.toml", "/tmp/out", "3", "https://tidal.com/album/123")
	want := []string{
		"--config-path", "/cfg/streamrip.toml",
		"--folder", "/tmp/out",
		"--quality", "3",
		"url", "https://tidal.com/album/123",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildRipArgs() = %v, want %v", got, want)
	}
}

func TestDisableVideos(t *testing.T) {
	in := []byte("[tidal]\nquality = 3\ndownload_videos = true\naccess_token = \"tok123\"\n\n[downloads]\nfolder = \"/x\"\n")
	out, err := disableVideos(in)
	if err != nil {
		t.Fatalf("disableVideos: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "download_videos = false") {
		t.Errorf("expected download_videos = false, got:\n%s", got)
	}
	// must preserve the existing access token (do not clobber auth)
	if !strings.Contains(got, "tok123") {
		t.Errorf("expected access_token preserved, got:\n%s", got)
	}
	// re-running is idempotent
	out2, err := disableVideos(out)
	if err != nil || !strings.Contains(string(out2), "download_videos = false") {
		t.Errorf("not idempotent: err=%v out=%s", err, out2)
	}
}

func TestIsAuthenticatedFromTOML(t *testing.T) {
	withToken := []byte("[tidal]\naccess_token = \"abc123\"\nrefresh_token = \"r\"\n")
	if !isAuthenticatedFromTOML(withToken) {
		t.Error("expected authenticated when access_token is set")
	}
	empty := []byte("[tidal]\naccess_token = \"\"\n")
	if isAuthenticatedFromTOML(empty) {
		t.Error("expected NOT authenticated when access_token is empty")
	}
	missing := []byte("[downloads]\nfolder = \"/x\"\n")
	if isAuthenticatedFromTOML(missing) {
		t.Error("expected NOT authenticated when [tidal] is absent")
	}
	if isAuthenticatedFromTOML([]byte("this is not valid toml = = =")) {
		t.Error("expected NOT authenticated on parse error")
	}
}

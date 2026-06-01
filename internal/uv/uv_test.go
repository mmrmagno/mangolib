package uv

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestManagedPathEndsWithLocalBinUv(t *testing.T) {
	got := ManagedPath()
	want := filepath.Join(".local", "bin", "uv")
	if !strings.HasSuffix(got, want) {
		t.Fatalf("ManagedPath() = %q, want suffix %q", got, want)
	}
}

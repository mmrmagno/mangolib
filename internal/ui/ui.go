package ui

import (
	"fmt"
	"os"
)

const banner = `
                                    _ _ _
  _ __ ___   __ _ _ __   __ _  ___ | (_) |__
 | '_ ` + "`" + ` _ \ / _` + "`" + ` | '_ \ / _` + "`" + ` |/ _ \| | | '_ \
 | | | | | | (_| | | | | (_| | (_) | | | |_) |
 |_| |_| |_|\__,_|_| |_|\__, |\___/|_|_|_.__/
                         |___/
 Your music library manager, downloader & iPod sync
 ---------------------------------------------------`

func Banner() {
	fmt.Println(banner)
}

func Step(msg string) {
	fmt.Printf("→ %s\n", msg)
}

func Success(msg string) {
	fmt.Printf("✓ %s\n", msg)
}

func Warn(msg string) {
	fmt.Fprintf(os.Stderr, "⚠ %s\n", msg)
}

func Fatal(msg string) {
	fmt.Fprintf(os.Stderr, "✗ %s\n", msg)
	os.Exit(1)
}

func Fatalf(format string, args ...any) {
	Fatal(fmt.Sprintf(format, args...))
}

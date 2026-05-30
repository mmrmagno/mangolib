<p align="center">
  <img src="assets/logo.svg" alt="mangolib" width="380"/>
</p>

<p align="center">
  A single-binary music library manager for people who actually own their music.
  <br/>
  Download from Spotify &amp; YouTube, organize your collection, sync to your iPod.
</p>

<p align="center">
  <a href="https://github.com/mmrmagno/mangolib/releases/latest">
    <img src="https://img.shields.io/github/v/release/mmrmagno/mangolib?color=FF8C00&label=release" alt="Latest Release"/>
  </a>
  <a href="https://github.com/mmrmagno/mangolib/actions/workflows/release.yml">
    <img src="https://github.com/mmrmagno/mangolib/actions/workflows/release.yml/badge.svg" alt="Build"/>
  </a>
  <a href="https://aur.archlinux.org/packages/mangolib-bin">
    <img src="https://img.shields.io/aur/version/mangolib-bin?color=1793d1&label=AUR" alt="AUR"/>
  </a>
  <img src="https://img.shields.io/badge/license-AGPL--3.0-blue" alt="License"/>
</p>

---

> **Disclaimer:** mangolib is intended for managing music you already legally own. Downloading copyrighted content without authorization may violate the terms of service of the platforms involved and applicable law in your jurisdiction. This tool does not circumvent DRM or access gated content; it uses public APIs and publicly accessible YouTube streams. The author is not responsible for any misuse. Use responsibly and respect artists.

---

## Features

- **Spotify:** resolves tracks, albums, and playlists via the Spotify Web API, searches YouTube for each track, downloads and tags with authoritative Spotify metadata
- **YouTube:** direct video or playlist download with progress bars; titles automatically cleaned of channel prefixes and video noise
- **Native tagging:** writes ID3v2 tags (MP3) and ffmpeg metadata (M4A / FLAC) directly, no external taggers
- **Cover art:** fetches high-res art from iTunes and MusicBrainz, embeds in tags and writes `cover.jpg` for Rockbox
- **Library organization:** `Artist / Album / NN. Title.ext` structure enforced with `reorganize`
- **iPod sync:** rsyncs your library to a Rockbox iPod; bidirectional (PC to iPod or iPod to PC)
- **Self-managing:** downloads and updates `yt-dlp` automatically; no Python, no Docker

---

## Requirements

| Tool | How to get it |
|---|---|
| `ffmpeg` | `pacman -S ffmpeg` · `apt install ffmpeg` · `brew install ffmpeg` |
| `yt-dlp` | Auto-downloaded by mangolib on first use |

---

## Installation

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/mmrmagno/mangolib/main/install.sh | bash
```

Installs the binary to `~/.local/bin/mangolib` and creates a default config at `~/.config/mangolib/mangolib.toml`.

### Arch Linux

```bash
yay -S mangolib-bin
```

### Build from source

```bash
git clone https://github.com/mmrmagno/mangolib
cd mangolib
go build -o ~/.local/bin/mangolib ./cmd/mangolib
```

Requires Go 1.22+.

---

## Setup

Edit `~/.config/mangolib/mangolib.toml` (created automatically on first run):

```toml
music_library = "~/Music/mangolib"
ipod_mount    = "/mnt/MANGOPOD/Music"   # set to the Music subfolder, not the mount root

[spotify]
client_id     = "your_client_id"
client_secret = "your_client_secret"

[download]
audio_format  = "mp3"     # mp3 | m4a | flac
audio_quality = "320"     # kbps

[covers]
size = 500                # cover.jpg output size in px
```

**Getting Spotify credentials:** create a free app at [developer.spotify.com/dashboard](https://developer.spotify.com/dashboard). No website or redirect URIs needed, just copy the client ID and secret.

---

## Usage

```
mangolib [--verbose] <command> [flags]
```

| Command | What it does |
|---|---|
| `download <URL>` | Auto-detect source and download (Spotify or YouTube) |
| `download --spotify <URL>` | Spotify track / album / playlist: download + tag + cover |
| `download --youtube <URL>` | YouTube video or playlist: audio + cover, titles auto-cleaned |
| `download --youtube --album "Name" --artist "Name" <URL>` | Override album and artist for YouTube |
| `get-covers` | Fetch and embed missing album art for every track in the library |
| `get-covers --force` | Re-fetch and overwrite all covers, replacing any bad art |
| `reorganize` | Re-organize files into `Artist/Album/NN. Title.ext` using embedded tags |
| `reorganize --clean` | Strip YouTube title noise from all track titles, then reorganize |
| `import <path>` | Import pre-tagged audio files from a folder into the library |
| `ls` | List all tracks as `Artist / Album / Title` |
| `sync` | Sync library to iPod via rsync |
| `sync --from-ipod` | Pull music from iPod into library and reorganize |
| `sync --dry-run` | Show what would be synced without transferring files |
| `update` | Update yt-dlp to the latest release |
| `init` | Scan and re-organize everything already in the library |

Use `-v` / `--verbose` on any command to show raw output from yt-dlp, ffmpeg, and rsync.

### Examples

```bash
# Download a Spotify album
mangolib download https://open.spotify.com/album/...

# Download a YouTube playlist (titles cleaned automatically)
mangolib download https://youtube.com/playlist?list=...

# Download a single YouTube video with explicit tags
mangolib download --album "Dangerous" --artist "Michael Jackson" https://youtu.be/...

# Fix YouTube title noise on existing library tracks
mangolib reorganize --clean

# Fetch missing covers
mangolib get-covers

# Re-fetch all covers at full quality (replaces bad art)
mangolib get-covers --force

# Sync to iPod
mangolib sync

# Pull music from iPod back to library
mangolib sync --from-ipod

# See exactly what sync would do before running it
mangolib sync --dry-run
```

---

## Roadmap

- [x] Spotify download (track / album / playlist)
- [x] YouTube download with progress bars and auto-cleaned titles
- [x] Native ID3 / ffmpeg tagging
- [x] Cover art: iTunes high-res + MusicBrainz fallback
- [x] Rockbox `cover.jpg` support (resized, square-padded)
- [x] Bidirectional iPod sync (`sync --from-ipod`)
- [x] AUR package (`mangolib-bin`)
- [ ] Tidal download
- [ ] Bubble Tea TUI with progress bars and album art preview

---

## Credits

Inspired by [podlib](https://github.com/mikeshootzz/podlib) by mikeshootzz. Thank you for the original idea.

---

## License

[AGPL-3.0](LICENSE)

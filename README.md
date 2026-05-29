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
- **YouTube:** direct video or playlist download; `--album` and `--artist` flags for clean folder organization
- **Native tagging:** writes ID3v2 tags (MP3) and ffmpeg metadata (M4A / FLAC) directly, no external taggers
- **Cover art:** fetches from iTunes Search API and Cover Art Archive, embeds automatically after every download
- **Library organization:** `Artist / Album / NN. Title.ext` structure enforced with `reorganize`
- **iPod sync:** rsyncs your library to a Rockbox iPod (or any mount point)
- **Self-managing:** downloads and updates `yt-dlp` automatically

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
ipod_mount    = "/mnt/ipod"        # leave empty to skip sync

[spotify]
client_id     = "your_client_id"
client_secret = "your_client_secret"
```

**Getting Spotify credentials:** create a free app at [developer.spotify.com/dashboard](https://developer.spotify.com/dashboard). No website or redirect URIs needed, just copy the client ID and secret.

---

## Usage

```
mangolib <command> [flags]
```

| Command | What it does |
|---|---|
| `download <URL>` | Auto-detect source and download (Spotify or YouTube) |
| `download --spotify <URL>` | Spotify track / album / playlist: download + tag + cover |
| `download --youtube <URL>` | YouTube video or playlist: audio + cover |
| `download --youtube --album "Name" --artist "Name" <URL>` | Override folder metadata for YouTube |
| `get-covers` | Fetch and embed missing album art for every track in the library |
| `reorganize` | Re-organize files into `Artist/Album/NN. Title.ext` using embedded tags |
| `import <path>` | Import audio files from a folder into the library |
| `ls` | List all tracks as `Artist / Album / Title` |
| `sync` | Sync library to iPod via rsync |
| `update` | Update yt-dlp to the latest release |
| `init` | Scan and re-organize everything already in the library |

### Examples

```bash
# Download a Spotify album
mangolib download https://open.spotify.com/album/...

# Download a YouTube playlist
mangolib download https://youtube.com/playlist?list=...

# Download a single YouTube video with proper tags
mangolib download --album "Dangerous" --artist "Michael Jackson" https://youtu.be/...

# Fetch missing covers across the whole library
mangolib get-covers

# Sync to iPod
mangolib sync
```

---

## Roadmap

- [x] Spotify download (track / album / playlist)
- [x] YouTube download with metadata override flags
- [x] Native ID3 / ffmpeg tagging
- [x] Cover art auto-fetch and embed
- [x] AUR package (`mangolib-bin`)
- [ ] Tidal download
- [ ] `--clean` flag to strip YouTube title noise (`(Official Video)`, `(Audio)`, etc.)
- [ ] Bubble Tea TUI with progress bars and album art preview

---

## Credits

Inspired by [podlib](https://github.com/mikeshootzz/podlib) by mikeshootzz. Thank you for the original idea.

---

## License

[AGPL-3.0](LICENSE)

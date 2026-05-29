#!/usr/bin/env bash
set -euo pipefail

REPO="mmrmagno/mangolib"
BIN_DIR="${HOME}/.local/bin"
CONFIG_DIR="${HOME}/.config/mangolib"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
URL="https://github.com/${REPO}/releases/download/${LATEST}/mangolib-${OS}-${ARCH}"

echo "Installing mangolib ${LATEST} (${OS}/${ARCH})..."
mkdir -p "$BIN_DIR"
curl -fsSL "$URL" -o "$BIN_DIR/mangolib"
chmod +x "$BIN_DIR/mangolib"

if ! echo "$PATH" | grep -q "$BIN_DIR"; then
  for rc in "${HOME}/.bashrc" "${HOME}/.zshrc"; do
    [ -f "$rc" ] && echo "export PATH=\"\$PATH:$BIN_DIR\"" >> "$rc" || true
  done
  echo "Added $BIN_DIR to PATH — restart your shell or run: export PATH=\"\$PATH:$BIN_DIR\""
fi

mkdir -p "$CONFIG_DIR"
if [ ! -f "$CONFIG_DIR/mangolib.toml" ]; then
  cat > "$CONFIG_DIR/mangolib.toml" <<'EOF'
music_library = "~/Music/mangolib"
ipod_mount    = ""

[spotify]
client_id     = ""
client_secret = ""

[download]
audio_format  = "mp3"
audio_quality = "320"

[tidal]
quality = "HI_RES_LOSSLESS"

[library]
duplicate_action = "skip"

[covers]
size = 500
EOF
  echo "Created default config at $CONFIG_DIR/mangolib.toml"
  echo "Edit it to set ipod_mount and Spotify credentials."
fi

if ! command -v ffmpeg &>/dev/null; then
  echo "WARNING: ffmpeg not found — required for audio conversion."
  echo "  Ubuntu/Debian: sudo apt install ffmpeg"
  echo "  macOS:         brew install ffmpeg"
fi

echo ""
echo "Done! Run: mangolib --help"

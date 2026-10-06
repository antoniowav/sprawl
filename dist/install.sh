#!/usr/bin/env bash
# Install Sprawl for the current user: binary, icon and launcher entry.
# Undo with: dist/install.sh --uninstall
set -euo pipefail
cd "$(dirname "$0")/.."

bin="$HOME/.local/bin/sprawl"
icon="$HOME/.local/share/icons/hicolor/256x256/apps/sprawl.png"
desktop="$HOME/.local/share/applications/sprawl.desktop"

if [[ "${1:-}" == "--uninstall" ]]; then
  rm -f "$bin" "$icon" "$desktop"
  echo "removed Sprawl (saves in ~/.local/share/sprawl and config in ~/.config/sprawl are kept)"
  exit 0
fi

go build -trimpath -ldflags "-X github.com/antoniowav/sprawl/internal/meta.Version=$(git describe --tags --always 2>/dev/null || echo 0.1.0-dev)" -o sprawl ./cmd/sprawl
install -Dm755 sprawl "$bin"
./sprawl -write-icon "$icon"
mkdir -p "$(dirname "$desktop")"
# Absolute Exec path, so it works even if ~/.local/bin isn't on PATH.
sed "s|^Exec=sprawl|Exec=$bin|" dist/sprawl.desktop > "$desktop"
command -v update-desktop-database >/dev/null && update-desktop-database -q "$(dirname "$desktop")" || true
command -v gtk-update-icon-cache >/dev/null && gtk-update-icon-cache -q "$HOME/.local/share/icons/hicolor" 2>/dev/null || true
echo "installed: $bin"
echo "launcher:  $desktop"

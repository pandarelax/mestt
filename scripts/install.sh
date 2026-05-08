#!/usr/bin/env sh
set -eu

INSTALL_DIR="${MESTT_INSTALL_DIR:-$HOME/.local/bin}"

mkdir -p "$INSTALL_DIR"

go build -o "$INSTALL_DIR/mestt" ./cmd/mestt
go build -tags fyne -o "$INSTALL_DIR/mestt-gui" ./cmd/mestt-gui
go build -o "$INSTALL_DIR/mesttd" ./cmd/mesttd

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) printf 'warning: %s is not in PATH\n' "$INSTALL_DIR" >&2 ;;
esac

"$INSTALL_DIR/mestt" version

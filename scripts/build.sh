#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
ROOT=$(pwd)
export GOPATH="$ROOT/.local/gopath" GOMODCACHE="$ROOT/.local/gomodcache" GOCACHE="$ROOT/.local/gocache" GOBIN="$ROOT/.local/bin" GOTMPDIR="$ROOT/.local/tmp" TMPDIR="$ROOT/.local/tmp" GOTOOLCHAIN=local
mkdir -p "$GOPATH" "$GOMODCACHE" "$GOCACHE" "$GOBIN" "$GOTMPDIR" dist
# Requer GTK/WebKitGTK e toolchain nativo já existentes. Não instala nada.
go build -trimpath -tags "desktop,production${TERMIASHELL_EXTRA_TAGS:+,$TERMIASHELL_EXTRA_TAGS}" -o dist/termiaShell .
cp frontend/termiaShell.png frontend/termiaShell.svg dist/

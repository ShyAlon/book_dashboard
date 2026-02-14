#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DESKTOP_DIR="$ROOT_DIR/desktop"
FRONTEND_DIR="$DESKTOP_DIR/frontend"
APP_BUNDLE="$DESKTOP_DIR/build/bin/desktop.app"

export GOCACHE="${GOCACHE:-/tmp/go-build}"
export GOMODCACHE="${GOMODCACHE:-/tmp/gomodcache}"
export GOPATH="${GOPATH:-/tmp/gopath}"
mkdir -p "$GOCACHE" "$GOMODCACHE" "$GOPATH"

if ! command -v go >/dev/null 2>&1; then
  echo "error: go is not installed or not on PATH" >&2
  exit 1
fi

if ! command -v npm >/dev/null 2>&1; then
  echo "error: npm is not installed or not on PATH" >&2
  exit 1
fi

if ! command -v wails >/dev/null 2>&1; then
  if [[ -x "$GOPATH/bin/wails" ]]; then
    export PATH="$GOPATH/bin:$PATH"
  else
    echo "error: wails CLI is not installed. Install with:" >&2
    echo "  go install github.com/wailsapp/wails/v2/cmd/wails@latest" >&2
    exit 1
  fi
fi

echo "==> Running backend tests"
(cd "$ROOT_DIR" && go test ./...)

echo "==> Building frontend"
(cd "$FRONTEND_DIR" && npm install && npm run build)

echo "==> Packaging desktop app"
(cd "$DESKTOP_DIR" && wails build -clean)

echo "==> Launching app bundle"
open "$APP_BUNDLE"

echo "done: $APP_BUNDLE"

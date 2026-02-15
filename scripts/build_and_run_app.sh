#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DESKTOP_DIR="$ROOT_DIR/desktop"
FRONTEND_DIR="$DESKTOP_DIR/frontend"
APP_BUNDLE="$DESKTOP_DIR/build/bin/desktop.app"
APP_EXECUTABLE="$APP_BUNDLE/Contents/MacOS/desktop"

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

echo "==> Checking for running app instance"
if [[ -x "$APP_EXECUTABLE" ]]; then
  APP_PIDS="$(pgrep -f "$APP_EXECUTABLE" || true)"
  if [[ -n "$APP_PIDS" ]]; then
    echo "found running instance(s): $APP_PIDS"
    # shellcheck disable=SC2086
    kill $APP_PIDS || true

    for _ in {1..20}; do
      STILL_RUNNING="$(pgrep -f "$APP_EXECUTABLE" || true)"
      if [[ -z "$STILL_RUNNING" ]]; then
        break
      fi
      sleep 0.25
    done

    STILL_RUNNING="$(pgrep -f "$APP_EXECUTABLE" || true)"
    if [[ -n "$STILL_RUNNING" ]]; then
      echo "forcing shutdown for PID(s): $STILL_RUNNING"
      # shellcheck disable=SC2086
      kill -9 $STILL_RUNNING || true
    fi
  else
    echo "no running instance found"
  fi
else
  echo "no previous app bundle executable found yet"
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

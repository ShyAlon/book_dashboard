#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DESKTOP_DIR="$ROOT_DIR/desktop"
FRONTEND_DIR="$DESKTOP_DIR/frontend"
PID_FILE="${TMPDIR:-/tmp}/book_dashboard_e2e.pid"

export GOCACHE="${GOCACHE:-/tmp/go-build}"
export GOMODCACHE="${GOMODCACHE:-/tmp/gomodcache}"
export GOPATH="${GOPATH:-/tmp/gopath}"
export GO_TEST_TIMEOUT="${GO_TEST_TIMEOUT:-45m}"
export OLLAMA_URL="${OLLAMA_URL:-http://127.0.0.1:11434}"
export LANGUAGETOOL_URL="${LANGUAGETOOL_URL:-http://127.0.0.1:8010/v2/check}"
mkdir -p "$GOCACHE" "$GOMODCACHE" "$GOPATH"

cleanup_stale_desktop_test_processes() {
  if ! command -v pgrep >/dev/null 2>&1; then
    return
  fi

  stale_pids=()
  while IFS= read -r pid; do
    [[ -z "$pid" ]] && continue
    skip=0
    for existing in "${stale_pids[@]:-}"; do
      if [[ "$existing" == "$pid" ]]; then
        skip=1
        break
      fi
    done
    if [[ "$skip" -eq 0 ]]; then
      stale_pids+=("$pid")
    fi
  done < <(
    {
      pgrep -f '/go-build.*/desktop\.test' || true
      pgrep -f 'go test .*desktop' || true
    }
  )

  if [[ "${#stale_pids[@]}" -eq 0 ]]; then
    return
  fi

  echo "==> [preflight] Killing stale desktop test processes: ${stale_pids[*]}"
  kill "${stale_pids[@]}" 2>/dev/null || true
  sleep 1
  for pid in "${stale_pids[@]}"; do
    if kill -0 "$pid" 2>/dev/null; then
      kill -9 "$pid" 2>/dev/null || true
    fi
  done
}

cleanup_pid_file() {
  if [[ -f "$PID_FILE" ]] && [[ "$(cat "$PID_FILE" 2>/dev/null)" == "$$" ]]; then
    rm -f "$PID_FILE"
  fi
}

if [[ -f "$PID_FILE" ]]; then
  old_pid="$(cat "$PID_FILE" 2>/dev/null || true)"
  if [[ -n "${old_pid:-}" ]] && kill -0 "$old_pid" 2>/dev/null; then
    echo "==> [preflight] Previous e2e run detected (pid=$old_pid); terminating it"
    kill -TERM "$old_pid" 2>/dev/null || true
    sleep 1
    if kill -0 "$old_pid" 2>/dev/null; then
      kill -KILL "$old_pid" 2>/dev/null || true
    fi
  fi
fi
echo "$$" > "$PID_FILE"
trap cleanup_pid_file EXIT

if ! command -v go >/dev/null 2>&1; then
  echo "error: go is not installed or not on PATH" >&2
  exit 1
fi

if ! command -v npm >/dev/null 2>&1; then
  echo "error: npm is not installed or not on PATH" >&2
  exit 1
fi

cleanup_stale_desktop_test_processes

if [[ "${E2E_SAVE_ARTIFACTS:-0}" == "1" ]]; then
  artifacts_dir="$DESKTOP_DIR/test-artifacts/e2e"
  if [[ -d "$artifacts_dir" ]]; then
    echo "==> [preflight] Removing previous artifacts: $artifacts_dir"
    rm -rf "$artifacts_dir"
  fi
fi

echo "==> [1/6] Root module tests"
(cd "$ROOT_DIR" && go test ./...)

echo "==> [2/6] Ingestion integration (books folder)"
(cd "$ROOT_DIR" && go test -v ./internal/ingest -run TestParseBooksFolderPDFAndDOCX)

echo "==> [3/6] Desktop module tests (fast suite)"
(cd "$DESKTOP_DIR" && go test -v -timeout "$GO_TEST_TIMEOUT" ./backend)
(cd "$DESKTOP_DIR" && go test -v -timeout "$GO_TEST_TIMEOUT" . -run TestAnalyzeFilePersistsReport)

echo "==> [4/6] Desktop books analysis integration"
if [[ "${E2E_SAVE_ARTIFACTS:-0}" == "1" ]]; then
  (cd "$DESKTOP_DIR" && MHD_TRACE_PROGRESS=1 KEEP_E2E_ARTIFACTS=1 go test -timeout "$GO_TEST_TIMEOUT" -v . -run 'TestAnalyzeBooksDOCX_RichAnalysis|TestAnalyzeBooksPDF_ChapterizationOnly')
  echo "artifacts saved under: $DESKTOP_DIR/test-artifacts/e2e/"
else
  (cd "$DESKTOP_DIR" && MHD_TRACE_PROGRESS=1 go test -timeout "$GO_TEST_TIMEOUT" -v . -run 'TestAnalyzeBooksDOCX_RichAnalysis|TestAnalyzeBooksPDF_ChapterizationOnly')
fi

echo "==> [5/6] Service lifecycle diagnostics e2e"
(cd "$DESKTOP_DIR" && go test -timeout "$GO_TEST_TIMEOUT" -v . -run TestServiceLifecycleDiagnosticsMissingBinaries_E2E)

echo "==> [6/6] Frontend production build"
(cd "$FRONTEND_DIR" && npm run build)

echo "all e2e checks passed"

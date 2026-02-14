#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR"
E2E_SAVE_ARTIFACTS=1 ./scripts/run_full_e2e_test.sh

echo
echo "E2E artifacts available at:"
echo "  $ROOT_DIR/desktop/test-artifacts/e2e/"

#!/usr/bin/env bash
# build.sh — Build the who3 binary.
#
# Run as your normal (non-root) user from anywhere inside the repo:
#   bash deploy/who3/build.sh
#
# The compiled binary is written to dist/who3 at the repo root.
# Run deploy/who3/install.sh (as root/sudo) afterwards to deploy it.
set -euo pipefail

BINARY_NAME="who3"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIST_DIR="${REPO_ROOT}/dist"

# ---------------------------------------------------------------------------
# Check for Go toolchain
# ---------------------------------------------------------------------------
echo "==> Checking for Go toolchain..."
if ! command -v go &>/dev/null; then
  echo "ERROR: 'go' not found. Install Go 1.21+ and ensure it is on PATH." >&2
  exit 1
fi
echo "    $(go version)"

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
echo "==> Building ${BINARY_NAME}..."
mkdir -p "${DIST_DIR}"
cd "${REPO_ROOT}"
go build -trimpath -ldflags="-s -w" -o "${DIST_DIR}/${BINARY_NAME}" ./cmd/who3/

echo "==> Binary written to ${DIST_DIR}/${BINARY_NAME}"
echo "    Run 'sudo bash deploy/who3/install.sh' to deploy."

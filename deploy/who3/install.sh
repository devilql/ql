#!/usr/bin/env bash
# install.sh — Deploy who3 as a systemd service on Ubuntu.
#
# Build the binary first (as a normal user):
#   bash deploy/who3/build.sh
#
# Then run this script as root:
#   sudo bash deploy/who3/install.sh
#
# Configuration is read from /etc/who3/who3.cfg.
# On first install that file is seeded from deploy/who3/who3.cfg.default.
# Edit /etc/who3/who3.cfg on the server for live settings; it is preserved
# across reinstalls so you never lose your customisations.
set -euo pipefail

BINARY_NAME="who3"
SERVICE_FILE="/etc/systemd/system/${BINARY_NAME}.service"
CFG_DIR="/etc/who3"
CFG_FILE="${CFG_DIR}/who3.cfg"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DEFAULT_CFG="${REPO_ROOT}/deploy/who3/who3.cfg.default"
DIST_BINARY="${REPO_ROOT}/dist/${BINARY_NAME}"

# ---------------------------------------------------------------------------
# Verify the binary has been built
# ---------------------------------------------------------------------------
if [[ ! -f "${DIST_BINARY}" ]]; then
  echo "ERROR: ${DIST_BINARY} not found." >&2
  echo "       Run 'bash deploy/who3/build.sh' first (as a normal user)." >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# Seed config on first install — never overwrite an existing live config
# ---------------------------------------------------------------------------
if [[ ! -f "${CFG_FILE}" ]]; then
  echo "==> No config found at ${CFG_FILE}."
  echo "    Seeding from ${DEFAULT_CFG}..."
  mkdir -p "${CFG_DIR}"
  cp "${DEFAULT_CFG}" "${CFG_FILE}"
  chmod 640 "${CFG_FILE}"
  echo "    Review ${CFG_FILE} and re-run the script when you are ready to deploy."
  exit 0
fi

# ---------------------------------------------------------------------------
# Load configuration
# ---------------------------------------------------------------------------
# shellcheck source=/dev/null
source "${CFG_FILE}"

# Validate required variables supplied by the config
: "${SERVICE_USER:?SERVICE_USER must be set in ${CFG_FILE}}"
: "${INSTALL_DIR:?INSTALL_DIR must be set in ${CFG_FILE}}"
: "${DB_PATH:?DB_PATH must be set in ${CFG_FILE}}"
: "${LISTEN_ADDR:?LISTEN_ADDR must be set in ${CFG_FILE}}"
CORS_ORIGINS="${CORS_ORIGINS:-}"
CORS_ORIGIN_SUFFIXES="${CORS_ORIGIN_SUFFIXES:-}"

echo "==> Configuration loaded from ${CFG_FILE}"
echo "    SERVICE_USER=${SERVICE_USER}  INSTALL_DIR=${INSTALL_DIR}"
echo "    LISTEN_ADDR=${LISTEN_ADDR}  DB_PATH=${DB_PATH}"

# ---------------------------------------------------------------------------
# Stop and remove existing deployment (preserves the database)
# ---------------------------------------------------------------------------
if systemctl list-unit-files --quiet "${BINARY_NAME}.service" 2>/dev/null | grep -q "${BINARY_NAME}.service"; then
  echo "==> Existing deployment detected — stopping and disabling service..."
  systemctl stop  "${BINARY_NAME}.service" || true
  systemctl disable "${BINARY_NAME}.service" || true
fi

# ---------------------------------------------------------------------------
# Create service user (no login shell, no home directory)
# ---------------------------------------------------------------------------
if ! id -u "${SERVICE_USER}" &>/dev/null; then
  echo "==> Creating system user '${SERVICE_USER}'..."
  useradd --system --no-create-home --shell /usr/sbin/nologin "${SERVICE_USER}"
fi

# ---------------------------------------------------------------------------
# Install binary
# ---------------------------------------------------------------------------
echo "==> Installing binary to ${INSTALL_DIR}/${BINARY_NAME}..."
mkdir -p "${INSTALL_DIR}"
install -m 0755 "${DIST_BINARY}" "${INSTALL_DIR}/${BINARY_NAME}"

# Ensure the install directory is owned by the service user so it can write
# the SQLite database and WAL files.
chown -R "${SERVICE_USER}:${SERVICE_USER}" "${INSTALL_DIR}"

# ---------------------------------------------------------------------------
# Write systemd unit file
# ---------------------------------------------------------------------------
echo "==> Writing systemd unit ${SERVICE_FILE}..."
cat > "${SERVICE_FILE}" <<EOF
[Unit]
Description=who3 consecutive game-count tracker
After=network.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_USER}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/${BINARY_NAME} -port ${LISTEN_ADDR} -db ${DB_PATH}
Environment="CORS_ORIGINS=${CORS_ORIGINS}"
Environment="CORS_ORIGIN_SUFFIXES=${CORS_ORIGIN_SUFFIXES}"

# Keep the process alive on failure
Restart=on-failure
RestartSec=5

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=${INSTALL_DIR}

[Install]
WantedBy=multi-user.target
EOF

# ---------------------------------------------------------------------------
# Enable and start
# ---------------------------------------------------------------------------
echo "==> Reloading systemd and starting ${BINARY_NAME}.service..."
systemctl daemon-reload
systemctl enable "${BINARY_NAME}.service"
systemctl start  "${BINARY_NAME}.service"

echo ""
echo "==> Done. Service status:"
systemctl status "${BINARY_NAME}.service" --no-pager

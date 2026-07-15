#!/usr/bin/env bash
# Installs the ampligo-agent binary + systemd service on a Linux host.
# Usage: sudo ./install.sh
set -euo pipefail

BIN_NAME="ampligo-agent"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/ampligo"

if [ "$(id -u)" -ne 0 ]; then
    echo "This script must be run as root (sudo ./install.sh)" >&2
    exit 1
fi

if [ ! -f "./${BIN_NAME}" ]; then
    echo "Binary ./${BIN_NAME} not found. Build it first: go build -o ${BIN_NAME} ." >&2
    exit 1
fi

install -m 755 "./${BIN_NAME}" "${INSTALL_DIR}/${BIN_NAME}"

mkdir -p "${CONFIG_DIR}"
if [ ! -f "${CONFIG_DIR}/agent.yml" ]; then
    cp "./agent.example.yml" "${CONFIG_DIR}/agent.yml"
    chmod 600 "${CONFIG_DIR}/agent.yml"
    echo "Wrote ${CONFIG_DIR}/agent.yml - edit it and set api_key before starting the service."
fi

cp "./ampligo-agent.service" /etc/systemd/system/ampligo-agent.service
systemctl daemon-reload

echo "Installed. Next steps:"
echo "  1. Edit ${CONFIG_DIR}/agent.yml and set api_key"
echo "  2. sudo systemctl enable --now ampligo-agent"

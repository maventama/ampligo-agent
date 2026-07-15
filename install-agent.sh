#!/usr/bin/env bash
# Ampligo agent installer - self-contained, meant for:
#   curl -sSL https://ampligo.niago.id/install-agent.sh | sudo AMPLIGO_API_KEY=amp_xxx bash
#
# Downloads the right prebuilt binary from GitHub Releases, installs it,
# writes /etc/ampligo/agent.yml, and sets it up as a systemd service.
set -euo pipefail

REPO="maventama/ampligo-agent"
BIN_NAME="ampligo-agent"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/ampligo"

API_KEY="${AMPLIGO_API_KEY:-}"
INGEST_URL="${AMPLIGO_INGEST_URL:-https://ampligo.niago.id/api/v1/ingest/usage}"

if [ "$(id -u)" -ne 0 ]; then
    echo "This installer must be run as root, e.g. with sudo." >&2
    exit 1
fi

if [ -z "$API_KEY" ]; then
    echo "Missing API key. Generate one from the Settings tab in Ampligo, then run:" >&2
    echo "  curl -sSL https://ampligo.niago.id/install-agent.sh | sudo AMPLIGO_API_KEY=amp_xxx bash" >&2
    exit 1
fi

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *)
        echo "Unsupported architecture: $arch" >&2
        exit 1
        ;;
esac

if [ "$os" != "linux" ] && [ "$os" != "darwin" ]; then
    echo "Unsupported OS: $os" >&2
    exit 1
fi

if [ "$os" != "linux" ]; then
    echo "Warning: systemd service setup only applies to Linux. On $os, run the binary manually." >&2
fi

tarball="${BIN_NAME}_${os}_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/latest/download/${tarball}"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

echo "Downloading ${url} ..."
curl -fsSL "$url" -o "${tmp_dir}/${tarball}"
tar -xzf "${tmp_dir}/${tarball}" -C "$tmp_dir"

install -m 755 "${tmp_dir}/${BIN_NAME}" "${INSTALL_DIR}/${BIN_NAME}"
echo "Installed ${INSTALL_DIR}/${BIN_NAME}"

mkdir -p "$CONFIG_DIR"
cat > "${CONFIG_DIR}/agent.yml" <<CONFIG
api_key: "${API_KEY}"
ingest_url: "${INGEST_URL}"
interval_seconds: 15
CONFIG
chmod 600 "${CONFIG_DIR}/agent.yml"
echo "Wrote ${CONFIG_DIR}/agent.yml"

if [ "$os" = "linux" ] && command -v systemctl >/dev/null 2>&1; then
    cat > /etc/systemd/system/ampligo-agent.service <<SERVICE
[Unit]
Description=Ampligo monitoring agent
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=${INSTALL_DIR}/${BIN_NAME} --config ${CONFIG_DIR}/agent.yml
Restart=on-failure
RestartSec=5
DynamicUser=yes

[Install]
WantedBy=multi-user.target
SERVICE

    systemctl daemon-reload
    systemctl enable --now ampligo-agent
    echo "ampligo-agent installed and started. Check status: systemctl status ampligo-agent"
else
    echo "Skipped systemd setup. Run manually: ${INSTALL_DIR}/${BIN_NAME} --config ${CONFIG_DIR}/agent.yml"
fi

echo ""
echo "To also monitor MySQL/Postgres, edit ${CONFIG_DIR}/agent.yml and add a 'database' block"
echo "(see https://github.com/${REPO}#optional-mysqlpostgres-monitoring), then:"
echo "  systemctl restart ampligo-agent"

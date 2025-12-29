#!/usr/bin/env bash
set -euo pipefail

APP_USER="${APP_USER:-trakt-sync}"
INSTALL_PREFIX="${INSTALL_PREFIX:-/usr/local}"
SERVICE_PATH="${SERVICE_PATH:-/etc/systemd/system/trakt-sync.service}"
TRAKT_SYNC_INTERVAL="${TRAKT_SYNC_INTERVAL:-6h}"
SKIP_DEPS="${SKIP_DEPS:-0}"

TRAKT_CLIENT_ID="${TRAKT_CLIENT_ID:-}"
TRAKT_CLIENT_SECRET="${TRAKT_CLIENT_SECRET:-}"
TRAKT_USERNAME="${TRAKT_USERNAME:-}"

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${INSTALL_PREFIX}/bin"
BIN_PATH="${BIN_DIR}/trakt-sync"
HAS_SYSTEMD=0

if [ "$(id -u)" -ne 0 ]; then
  echo "Please run as root."
  exit 1
fi

if ! command -v apt-get >/dev/null 2>&1; then
  echo "This installer expects apt-get (Debian/Ubuntu LXC)."
  exit 1
fi

if [ "$SKIP_DEPS" != "1" ]; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get update
  apt-get install -y ca-certificates git curl make golang-go
fi

if ! id -u "$APP_USER" >/dev/null 2>&1; then
  if command -v adduser >/dev/null 2>&1; then
    adduser --system --home "/var/lib/${APP_USER}" --group --disabled-login "$APP_USER"
  else
    if ! getent group "$APP_USER" >/dev/null 2>&1; then
      groupadd --system "$APP_USER"
    fi
    useradd --system --create-home --home-dir "/var/lib/${APP_USER}" --shell /usr/sbin/nologin --gid "$APP_USER" "$APP_USER"
  fi
fi

USER_HOME="$(getent passwd "$APP_USER" | cut -d: -f6)"
if [ -z "$USER_HOME" ]; then
  USER_HOME="/var/lib/${APP_USER}"
fi

if [ ! -f "${REPO_DIR}/go.mod" ]; then
  echo "go.mod not found in ${REPO_DIR}"
  exit 1
fi

if [ ! -f "${REPO_DIR}/config.example.yaml" ]; then
  echo "config.example.yaml not found in ${REPO_DIR}"
  exit 1
fi

install -d "$BIN_DIR"
cd "$REPO_DIR"
CGO_ENABLED=0 go build -trimpath -o "$BIN_PATH" ./cmd/trakt-sync

CONFIG_DIR="${USER_HOME}/.config/trakt-sync"
CONFIG_PATH="${CONFIG_DIR}/config.yaml"
mkdir -p "$CONFIG_DIR"
if [ ! -f "$CONFIG_PATH" ]; then
  cp "${REPO_DIR}/config.example.yaml" "$CONFIG_PATH"
fi

if [ -t 0 ] && [ -t 1 ]; then
  if [ -z "$TRAKT_CLIENT_ID" ]; then
    while [ -z "$TRAKT_CLIENT_ID" ]; do
      if ! read -r -p "Trakt Client ID: " TRAKT_CLIENT_ID; then
        echo "Input aborted."
        exit 1
      fi
    done
  fi

  if [ -z "$TRAKT_CLIENT_SECRET" ]; then
    while [ -z "$TRAKT_CLIENT_SECRET" ]; do
      if ! read -r -s -p "Trakt Client Secret: " TRAKT_CLIENT_SECRET; then
        echo "\nInput aborted."
        exit 1
      fi
      echo
    done
  fi

  if [ -z "$TRAKT_USERNAME" ]; then
    while [ -z "$TRAKT_USERNAME" ]; do
      if ! read -r -p "Trakt Username: " TRAKT_USERNAME; then
        echo "Input aborted."
        exit 1
      fi
    done
  fi
else
  if [ -z "$TRAKT_CLIENT_ID" ] || [ -z "$TRAKT_CLIENT_SECRET" ] || [ -z "$TRAKT_USERNAME" ]; then
    echo "Credentials not provided and no TTY detected; leaving config values unchanged."
  fi
fi

escape_sed() {
  printf '%s' "$1" | sed -e 's/[\\/|&]/\\&/g'
}

if [ -n "$TRAKT_CLIENT_ID" ]; then
  safe_id="$(escape_sed "$TRAKT_CLIENT_ID")"
  sed -i "s|^  client_id:.*|  client_id: \"${safe_id}\"|" "$CONFIG_PATH"
fi
if [ -n "$TRAKT_CLIENT_SECRET" ]; then
  safe_secret="$(escape_sed "$TRAKT_CLIENT_SECRET")"
  sed -i "s|^  client_secret:.*|  client_secret: \"${safe_secret}\"|" "$CONFIG_PATH"
fi
if [ -n "$TRAKT_USERNAME" ]; then
  safe_user="$(escape_sed "$TRAKT_USERNAME")"
  sed -i "s|^  username:.*|  username: \"${safe_user}\"|" "$CONFIG_PATH"
fi

chown -R "$APP_USER":"$APP_USER" "${USER_HOME}/.config"
chmod 600 "$CONFIG_PATH"

if command -v systemctl >/dev/null 2>&1; then
  HAS_SYSTEMD=1
  "$BIN_PATH" install-service --user "$APP_USER" --interval "$TRAKT_SYNC_INTERVAL" --path "$SERVICE_PATH"
  systemctl daemon-reload
fi

cat << EOF_OUTPUT
Installation complete.
Config: ${CONFIG_PATH}
Binary: ${BIN_PATH}
Service file: ${SERVICE_PATH}

Next steps:
1) Edit config and set trakt client_id/client_secret/username:
   nano ${CONFIG_PATH}
2) Authenticate:
   sudo -u ${APP_USER} ${BIN_PATH} auth
EOF_OUTPUT

if [ "$HAS_SYSTEMD" -eq 1 ]; then
  cat << EOF_OUTPUT
3) Start service:
   systemctl enable --now trakt-sync
EOF_OUTPUT
else
  cat << EOF_OUTPUT
3) systemd not detected; run manually:
   sudo -u ${APP_USER} ${BIN_PATH} daemon --interval ${TRAKT_SYNC_INTERVAL}
EOF_OUTPUT
fi

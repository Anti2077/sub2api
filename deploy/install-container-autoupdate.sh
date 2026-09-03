#!/usr/bin/env bash

set -euo pipefail

COMPOSE_DIR=""
COMPOSE_FILE="docker-compose.yml"
COMPOSE_ENV_FILE=".env"

usage() {
  cat <<'EOF'
Install the Sub2API host-side container auto-updater.

Usage:
  sudo ./install-container-autoupdate.sh --compose-dir /absolute/deploy/path [options]

Options:
  --compose-dir PATH   Directory containing the active Compose deployment (required)
  --compose-file FILE  Compose file inside that directory (default: docker-compose.yml)
  --env-file FILE      Compose environment file inside that directory (default: .env)
  -h, --help           Show this help

The installer enables a systemd timer that checks every five minutes. It only
updates the Compose service named "sub2api" from ghcr.io/anti2077/sub2api:custom.
EOF
}

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

validate_relative_file() {
  local name=$1
  local value=$2
  [[ "$value" != /* ]] || fail "$name must be relative to --compose-dir"
  [[ "$value" != *..* ]] || fail "$name must not contain .."
  [[ "$value" =~ ^[A-Za-z0-9_.-]+$ ]] || fail "$name contains unsupported characters"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --compose-dir)
      COMPOSE_DIR=${2:-}
      shift 2
      ;;
    --compose-file)
      COMPOSE_FILE=${2:-}
      shift 2
      ;;
    --env-file)
      COMPOSE_ENV_FILE=${2:-}
      shift 2
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
done

[[ $(id -u) -eq 0 ]] || fail "run this installer as root (for example, with sudo)"
[[ -n "$COMPOSE_DIR" ]] || fail "--compose-dir is required"
[[ "$COMPOSE_DIR" == /* && "$COMPOSE_DIR" != / ]] || fail "--compose-dir must be an absolute directory other than /"
[[ "$COMPOSE_DIR" =~ ^[A-Za-z0-9_./-]+$ ]] || fail "--compose-dir contains unsupported characters"
validate_relative_file --compose-file "$COMPOSE_FILE"
validate_relative_file --env-file "$COMPOSE_ENV_FILE"

command -v docker >/dev/null 2>&1 || fail "docker is not installed"
docker compose version >/dev/null 2>&1 || fail "Docker Compose v2 is not available"
command -v systemctl >/dev/null 2>&1 || fail "systemd is required"
command -v flock >/dev/null 2>&1 || fail "flock is required (normally provided by util-linux)"

COMPOSE_DIR=$(cd -- "$COMPOSE_DIR" && pwd -P)
[[ -f "$COMPOSE_DIR/$COMPOSE_FILE" ]] || fail "missing Compose file: $COMPOSE_DIR/$COMPOSE_FILE"
[[ -f "$COMPOSE_DIR/$COMPOSE_ENV_FILE" ]] || fail "missing Compose environment file: $COMPOSE_DIR/$COMPOSE_ENV_FILE"

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
for source_file in \
  sub2api-container-update.sh \
  sub2api-container-update.service \
  sub2api-container-update.timer \
  docker-compose.auto-update.yml
do
  [[ -f "$SCRIPT_DIR/$source_file" ]] || fail "missing installer source: $SCRIPT_DIR/$source_file"
done

SUB2API_IMAGE=ghcr.io/anti2077/sub2api:custom docker compose \
  --project-directory "$COMPOSE_DIR" \
  --env-file "$COMPOSE_DIR/$COMPOSE_ENV_FILE" \
  -f "$COMPOSE_DIR/$COMPOSE_FILE" \
  -f "$SCRIPT_DIR/docker-compose.auto-update.yml" \
  config --quiet

install -d -m 0755 /usr/local/libexec /etc/sub2api-container-update
install -d -m 0750 /var/lib/sub2api-container-update
install -m 0755 "$SCRIPT_DIR/sub2api-container-update.sh" /usr/local/libexec/sub2api-container-update
install -m 0644 "$SCRIPT_DIR/docker-compose.auto-update.yml" \
  /etc/sub2api-container-update/docker-compose.auto-update.yml
install -m 0644 "$SCRIPT_DIR/sub2api-container-update.service" \
  /etc/systemd/system/sub2api-container-update.service
install -m 0644 "$SCRIPT_DIR/sub2api-container-update.timer" \
  /etc/systemd/system/sub2api-container-update.timer

CONFIG_PATH=/etc/sub2api-container-update.env
if [[ -f "$CONFIG_PATH" ]]; then
  BACKUP_PATH="${CONFIG_PATH}.bak.$(date -u '+%Y%m%dT%H%M%SZ')"
  cp -a "$CONFIG_PATH" "$BACKUP_PATH"
  printf 'Backed up existing configuration to %s\n' "$BACKUP_PATH"
fi

cat > "$CONFIG_PATH" <<EOF
COMPOSE_PROJECT_DIR=$COMPOSE_DIR
COMPOSE_FILE=$COMPOSE_FILE
COMPOSE_ENV_FILE=$COMPOSE_ENV_FILE
COMPOSE_OVERRIDE_FILE=/etc/sub2api-container-update/docker-compose.auto-update.yml
SUB2API_SERVICE=sub2api
SUB2API_IMAGE=ghcr.io/anti2077/sub2api:custom
SUB2API_ROLLBACK_IMAGE=ghcr.io/anti2077/sub2api:rollback-local
HEALTH_TIMEOUT_SECONDS=180
HEALTH_POLL_INTERVAL_SECONDS=3
EOF
chmod 0600 "$CONFIG_PATH"

systemctl daemon-reload
systemctl enable --now sub2api-container-update.timer

cat <<'EOF'
Sub2API container auto-update is installed.

The first check is scheduled shortly. Useful commands:
  sudo systemctl start sub2api-container-update.service
  sudo systemctl status sub2api-container-update.timer
  sudo journalctl -u sub2api-container-update.service -f

The updater never starts a manually stopped Sub2API service. See
deploy/CONTAINER_AUTO_UPDATE.md for rollback, pause, and removal procedures.
EOF

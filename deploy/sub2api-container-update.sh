#!/usr/bin/env bash

set -euo pipefail

log() {
  printf '%s %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$*"
}

fatal() {
  log "ERROR: $*" >&2
  exit 1
}

require_positive_integer() {
  local name=$1
  local value=$2
  [[ "$value" =~ ^[1-9][0-9]*$ ]] || fatal "$name must be a positive integer"
}

resolve_from_project() {
  local path=$1
  if [[ "$path" == /* ]]; then
    printf '%s\n' "$path"
  else
    printf '%s/%s\n' "$COMPOSE_PROJECT_DIR" "$path"
  fi
}

container_health() {
  local container_id=$1
  "$DOCKER_BIN" inspect \
    --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}no-healthcheck{{end}}' \
    "$container_id" 2>/dev/null || printf 'missing\n'
}

wait_for_healthy() {
  local deadline=$((SECONDS + HEALTH_TIMEOUT_SECONDS))
  local container_id
  local status

  while (( SECONDS <= deadline )); do
    container_id=$("${COMPOSE[@]}" ps -q "$SUB2API_SERVICE")
    if [[ -z "$container_id" ]]; then
      status=missing
    else
      status=$(container_health "$container_id")
    fi

    case "$status" in
      healthy)
        return 0
        ;;
      unhealthy | exited | dead | missing | no-healthcheck)
        log "Container health check failed with status: $status"
        return 1
        ;;
      starting)
        ;;
      *)
        log "Waiting for container health; current status: $status"
        ;;
    esac

    sleep "$HEALTH_POLL_INTERVAL_SECONDS"
  done

  log "Container did not become healthy within ${HEALTH_TIMEOUT_SECONDS}s"
  return 1
}

record_failed_image() {
  local image_id=$1
  local temporary_file="${FAILED_IMAGE_FILE}.tmp"
  printf '%s\n' "$image_id" > "$temporary_file"
  chmod 0600 "$temporary_file"
  mv -f "$temporary_file" "$FAILED_IMAGE_FILE"
}

rollback_update() {
  local reason=$1
  log "Update failed: $reason"
  log "Restoring previous image $CURRENT_IMAGE_ID"

  record_failed_image "$TARGET_IMAGE_ID"
  "$DOCKER_BIN" image tag "$CURRENT_IMAGE_ID" "$SUB2API_IMAGE"

  if ! "${COMPOSE[@]}" up -d --no-deps --force-recreate "$SUB2API_SERVICE"; then
    fatal "rollback container recreation failed; previous image is available as $SUB2API_ROLLBACK_IMAGE"
  fi
  if ! wait_for_healthy; then
    fatal "rollback container did not become healthy; inspect service logs immediately"
  fi

  fatal "rolled back unhealthy image $TARGET_IMAGE_ID; it will not be retried until the remote image changes"
}

: "${COMPOSE_PROJECT_DIR:?COMPOSE_PROJECT_DIR is required}"

COMPOSE_FILE=${COMPOSE_FILE:-docker-compose.yml}
COMPOSE_OVERRIDE_FILE=${COMPOSE_OVERRIDE_FILE:-/etc/sub2api-container-update/docker-compose.auto-update.yml}
COMPOSE_ENV_FILE=${COMPOSE_ENV_FILE:-.env}
SUB2API_SERVICE=${SUB2API_SERVICE:-sub2api}
SUB2API_IMAGE=${SUB2API_IMAGE:-ghcr.io/anti2077/sub2api:custom}
SUB2API_ROLLBACK_IMAGE=${SUB2API_ROLLBACK_IMAGE:-ghcr.io/anti2077/sub2api:rollback-local}
HEALTH_TIMEOUT_SECONDS=${HEALTH_TIMEOUT_SECONDS:-180}
HEALTH_POLL_INTERVAL_SECONDS=${HEALTH_POLL_INTERVAL_SECONDS:-3}
STATE_DIR=${STATE_DIR:-/var/lib/sub2api-container-update}
LOCK_FILE=${LOCK_FILE:-/run/lock/sub2api-container-update.lock}
DOCKER_BIN=${DOCKER_BIN:-docker}
FLOCK_BIN=${FLOCK_BIN:-flock}

[[ "$COMPOSE_PROJECT_DIR" == /* && "$COMPOSE_PROJECT_DIR" != / ]] || \
  fatal "COMPOSE_PROJECT_DIR must be an absolute directory other than /"
[[ "$STATE_DIR" == /* && "$STATE_DIR" != / ]] || fatal "STATE_DIR must be an absolute directory other than /"
[[ "$LOCK_FILE" == /* && "$LOCK_FILE" != / ]] || fatal "LOCK_FILE must be an absolute file path"
[[ "$SUB2API_SERVICE" =~ ^[A-Za-z0-9_.-]+$ ]] || fatal "SUB2API_SERVICE contains invalid characters"
[[ "$SUB2API_IMAGE" == ghcr.io/anti2077/sub2api:* ]] || \
  fatal "SUB2API_IMAGE must use ghcr.io/anti2077/sub2api"
[[ "$SUB2API_ROLLBACK_IMAGE" == ghcr.io/anti2077/sub2api:* ]] || \
  fatal "SUB2API_ROLLBACK_IMAGE must use ghcr.io/anti2077/sub2api"
require_positive_integer HEALTH_TIMEOUT_SECONDS "$HEALTH_TIMEOUT_SECONDS"
require_positive_integer HEALTH_POLL_INTERVAL_SECONDS "$HEALTH_POLL_INTERVAL_SECONDS"

if [[ "$DOCKER_BIN" == */* ]]; then
  [[ -x "$DOCKER_BIN" ]] || fatal "Docker executable not found: $DOCKER_BIN"
else
  command -v "$DOCKER_BIN" >/dev/null 2>&1 || fatal "Docker executable not found: $DOCKER_BIN"
fi
if [[ "$FLOCK_BIN" == */* ]]; then
  [[ -x "$FLOCK_BIN" ]] || fatal "flock executable not found: $FLOCK_BIN"
else
  command -v "$FLOCK_BIN" >/dev/null 2>&1 || fatal "flock is required"
fi

COMPOSE_PATH=$(resolve_from_project "$COMPOSE_FILE")
COMPOSE_OVERRIDE_PATH=$(resolve_from_project "$COMPOSE_OVERRIDE_FILE")
COMPOSE_ENV_PATH=$(resolve_from_project "$COMPOSE_ENV_FILE")

[[ -d "$COMPOSE_PROJECT_DIR" ]] || fatal "Compose project directory does not exist: $COMPOSE_PROJECT_DIR"
[[ -f "$COMPOSE_PATH" ]] || fatal "Compose file does not exist: $COMPOSE_PATH"
[[ -f "$COMPOSE_OVERRIDE_PATH" ]] || fatal "Compose override does not exist: $COMPOSE_OVERRIDE_PATH"
[[ -f "$COMPOSE_ENV_PATH" ]] || fatal "Compose environment file does not exist: $COMPOSE_ENV_PATH"

mkdir -p "$STATE_DIR" "$(dirname "$LOCK_FILE")"
chmod 0750 "$STATE_DIR"
FAILED_IMAGE_FILE="$STATE_DIR/failed-image-id"

exec 9>"$LOCK_FILE"
if ! "$FLOCK_BIN" -n 9; then
  log "Another container update is already running; skipping"
  exit 0
fi

export SUB2API_IMAGE
COMPOSE=(
  "$DOCKER_BIN" compose
  --project-directory "$COMPOSE_PROJECT_DIR"
  --env-file "$COMPOSE_ENV_PATH"
  -f "$COMPOSE_PATH"
  -f "$COMPOSE_OVERRIDE_PATH"
)

CONTAINER_ID=$("${COMPOSE[@]}" ps -q "$SUB2API_SERVICE")
[[ -n "$CONTAINER_ID" ]] || fatal "$SUB2API_SERVICE is not running; automatic updates will not start stopped services"

CURRENT_IMAGE_ID=$("$DOCKER_BIN" inspect --format '{{.Image}}' "$CONTAINER_ID")
[[ -n "$CURRENT_IMAGE_ID" ]] || fatal "could not resolve the running image ID"

log "Checking $SUB2API_IMAGE for updates"
"$DOCKER_BIN" pull "$SUB2API_IMAGE"
TARGET_IMAGE_ID=$("$DOCKER_BIN" image inspect --format '{{.Id}}' "$SUB2API_IMAGE")
[[ -n "$TARGET_IMAGE_ID" ]] || fatal "could not resolve the pulled image ID"

if [[ "$CURRENT_IMAGE_ID" == "$TARGET_IMAGE_ID" ]]; then
  rm -f "$FAILED_IMAGE_FILE"
  log "No update available; running image is current"
  exit 0
fi

if [[ -f "$FAILED_IMAGE_FILE" ]] && [[ "$(<"$FAILED_IMAGE_FILE")" == "$TARGET_IMAGE_ID" ]]; then
  log "Skipping previously failed image $TARGET_IMAGE_ID; waiting for a newer remote image"
  exit 0
fi

log "Saving the current image as $SUB2API_ROLLBACK_IMAGE"
"$DOCKER_BIN" image tag "$CURRENT_IMAGE_ID" "$SUB2API_ROLLBACK_IMAGE"

log "Recreating only the $SUB2API_SERVICE service"
if ! "${COMPOSE[@]}" up -d --no-deps --force-recreate "$SUB2API_SERVICE"; then
  rollback_update "Docker Compose could not recreate the service"
fi
if ! wait_for_healthy; then
  rollback_update "the replacement container did not become healthy"
fi

rm -f "$FAILED_IMAGE_FILE"
log "Updated $SUB2API_SERVICE successfully: $CURRENT_IMAGE_ID -> $TARGET_IMAGE_ID"

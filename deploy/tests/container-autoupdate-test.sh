#!/usr/bin/env bash

set -euo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
updater="$repo_root/deploy/sub2api-container-update.sh"

fail() {
  printf 'container auto-update test failed: %s\n' "$1" >&2
  exit 1
}

assert_contains() {
  local file=$1
  local expected=$2
  grep -Fq -- "$expected" "$file" || fail "$file does not contain: $expected"
}

assert_not_contains() {
  local file=$1
  local unexpected=$2
  if grep -Fq -- "$unexpected" "$file"; then
    fail "$file unexpectedly contains: $unexpected"
  fi
}

make_fixture() {
  local fixture=$1
  mkdir -p "$fixture/project" "$fixture/state" "$fixture/bin" "$fixture/lock"
  : > "$fixture/project/.env"
  : > "$fixture/project/docker-compose.yml"
  : > "$fixture/project/docker-compose.auto-update.yml"
  : > "$fixture/docker.log"
  printf 'container-id\n' > "$fixture/container_id"
  printf 'healthy\n' > "$fixture/health"
  printf '0\n' > "$fixture/up_count"

  cat > "$fixture/bin/flock" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF

  cat > "$fixture/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

printf '%s\n' "$*" >> "$FAKE_DOCKER_LOG"

if [[ ${1:-} == compose ]]; then
  for argument in "$@"; do
    case "$argument" in
      ps)
        cat "$FAKE_ROOT/container_id"
        exit 0
        ;;
      up)
        count=$(<"$FAKE_ROOT/up_count")
        count=$((count + 1))
        printf '%s\n' "$count" > "$FAKE_ROOT/up_count"
        cp "$FAKE_ROOT/target_image" "$FAKE_ROOT/current_image"
        if [[ ${FAKE_UPDATE_RESULT:-healthy} == unhealthy && $count -eq 1 ]]; then
          printf 'unhealthy\n' > "$FAKE_ROOT/health"
        else
          printf 'healthy\n' > "$FAKE_ROOT/health"
        fi
        exit 0
        ;;
    esac
  done
fi

case "${1:-} ${2:-}" in
  "pull "*)
    cp "$FAKE_ROOT/remote_image" "$FAKE_ROOT/target_image"
    ;;
  "inspect --format")
    if [[ ${3:-} == *State.Health* ]]; then
      cat "$FAKE_ROOT/health"
    else
      cat "$FAKE_ROOT/current_image"
    fi
    ;;
  "image inspect")
    cat "$FAKE_ROOT/target_image"
    ;;
  "image tag")
    if [[ ${4:-} == "$SUB2API_IMAGE" ]]; then
      printf '%s\n' "${3:-}" > "$FAKE_ROOT/target_image"
    fi
    ;;
  *)
    ;;
esac
EOF

  chmod +x "$fixture/bin/docker" "$fixture/bin/flock"
}

run_updater() {
  local fixture=$1
  FAKE_ROOT="$fixture" \
  FAKE_DOCKER_LOG="$fixture/docker.log" \
  DOCKER_BIN="$fixture/bin/docker" \
  FLOCK_BIN="$fixture/bin/flock" \
  COMPOSE_PROJECT_DIR="$fixture/project" \
  COMPOSE_OVERRIDE_FILE=docker-compose.auto-update.yml \
  STATE_DIR="$fixture/state" \
  LOCK_FILE="$fixture/lock/update.lock" \
  HEALTH_TIMEOUT_SECONDS=1 \
  HEALTH_POLL_INTERVAL_SECONDS=1 \
  "$updater"
}

test_no_update() {
  local fixture=$1
  make_fixture "$fixture"
  printf 'sha256:current\n' > "$fixture/current_image"
  printf 'sha256:current\n' > "$fixture/remote_image"
  printf 'sha256:current\n' > "$fixture/target_image"

  run_updater "$fixture"

  assert_contains "$fixture/docker.log" "pull ghcr.io/anti2077/sub2api:custom"
  assert_not_contains "$fixture/docker.log" "up -d --no-deps --force-recreate sub2api"
}

test_stopped_service_is_not_started() {
  local fixture=$1
  make_fixture "$fixture"
  : > "$fixture/container_id"
  printf 'sha256:old\n' > "$fixture/current_image"
  printf 'sha256:new\n' > "$fixture/remote_image"
  printf 'sha256:new\n' > "$fixture/target_image"

  if run_updater "$fixture" >/dev/null 2>&1; then
    fail "stopped service unexpectedly triggered an update"
  fi

  assert_not_contains "$fixture/docker.log" "pull ghcr.io/anti2077/sub2api:custom"
  assert_not_contains "$fixture/docker.log" "up -d --no-deps --force-recreate sub2api"
}

test_successful_update() {
  local fixture=$1
  make_fixture "$fixture"
  printf 'sha256:old\n' > "$fixture/current_image"
  printf 'sha256:new\n' > "$fixture/remote_image"
  printf 'sha256:new\n' > "$fixture/target_image"

  run_updater "$fixture"

  [[ $(<"$fixture/current_image") == sha256:new ]] || fail "successful update did not use the new image"
  assert_contains "$fixture/docker.log" "image tag sha256:old ghcr.io/anti2077/sub2api:rollback-local"
  assert_contains "$fixture/docker.log" "up -d --no-deps --force-recreate sub2api"
  [[ ! -f "$fixture/state/failed-image-id" ]] || fail "successful image was marked as failed"
}

test_failed_update_rolls_back_and_is_quarantined() {
  local fixture=$1
  make_fixture "$fixture"
  printf 'sha256:old\n' > "$fixture/current_image"
  printf 'sha256:new\n' > "$fixture/remote_image"
  printf 'sha256:new\n' > "$fixture/target_image"

  if FAKE_UPDATE_RESULT=unhealthy run_updater "$fixture"; then
    fail "unhealthy update unexpectedly succeeded"
  fi

  [[ $(<"$fixture/current_image") == sha256:old ]] || fail "rollback did not restore the old image"
  [[ $(<"$fixture/state/failed-image-id") == sha256:new ]] || fail "failed image was not quarantined"
  [[ $(<"$fixture/up_count") == 2 ]] || fail "rollback should recreate the service twice"

  : > "$fixture/docker.log"
  FAKE_UPDATE_RESULT=healthy run_updater "$fixture"
  assert_not_contains "$fixture/docker.log" "up -d --no-deps --force-recreate sub2api"
}

test_rejects_unapproved_registry() {
  local fixture=$1
  make_fixture "$fixture"
  printf 'sha256:old\n' > "$fixture/current_image"
  printf 'sha256:new\n' > "$fixture/remote_image"
  printf 'sha256:new\n' > "$fixture/target_image"

  if SUB2API_IMAGE=docker.io/example/unapproved:latest run_updater "$fixture" >/dev/null 2>&1; then
    fail "unapproved registry image was accepted"
  fi
  [[ ! -s "$fixture/docker.log" ]] || fail "Docker was called for an unapproved registry"
}

tmp_root=$(mktemp -d)
trap 'rm -rf "$tmp_root"' EXIT

test_no_update "$tmp_root/no-update"
test_stopped_service_is_not_started "$tmp_root/stopped"
test_successful_update "$tmp_root/success"
test_failed_update_rolls_back_and_is_quarantined "$tmp_root/rollback"
test_rejects_unapproved_registry "$tmp_root/registry"

printf 'container auto-update tests passed\n'

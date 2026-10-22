#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

fail() {
  printf 'docker TLS trust test failed: %s\n' "$1" >&2
  exit 1
}

assert_line() {
  grep -Fqx "$2" "$1" || fail "$1 is missing: $2"
}

# eMudhra incident: Go and Node must reject the compromised intermediate
# keys, and OS clients need Alpine's updated CA bundle. A floating tag or
# cached apk layer alone does not guarantee the security update is installed.
for file in Dockerfile deploy/Dockerfile; do
  assert_line "$file" 'ARG NODE_IMAGE=node:24.22.1-alpine'
  assert_line "$file" 'ARG GOLANG_IMAGE=golang:1.27.3-alpine'
done
assert_line backend/Dockerfile 'FROM golang:1.27.3-alpine'
assert_line identity-engine/Dockerfile 'FROM node:24.22.1-alpine AS build'
assert_line identity-engine/Dockerfile 'FROM node:24.22.1-alpine'
for file in Dockerfile Dockerfile.goreleaser deploy/Dockerfile backend/Dockerfile identity-engine/Dockerfile; do
  grep -Fq "'ca-certificates>=20261020-r0'" "$file" || fail "$file allows the old CA bundle"
done
for file in .github/workflows/backend-ci.yml .github/workflows/security-scan.yml .github/workflows/release.yml; do
  grep -Fq "node-version: '24.22.1'" "$file" || fail "$file uses an unpatched Node.js version"
done

printf 'docker TLS trust test passed\n'

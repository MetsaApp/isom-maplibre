#!/usr/bin/env bash
# Runs locally what GitHub Actions CI ran on pull requests (CI is off to save
# paid minutes): the test job of .github/workflows/release-please.yml. The
# pre-push hook runs it. There is no slow suite, so there is no --full.
set -euo pipefail
cd "$(dirname "$0")/.."

case "${1:-}" in
  "") ;;
  *) echo "usage: scripts/check.sh" >&2; exit 2 ;;
esac

step() { printf '\n==> %s\n' "$*"; }
need() { command -v "$1" >/dev/null 2>&1 || { echo "check: $1 is missing; $2" >&2; exit 1; }; }

step tools
need go "install Go from https://go.dev/dl/ (go.mod names the version)"
need node "install Node.js 22 or later from https://nodejs.org (CI used 22)"
need npm "npm comes with Node.js from https://nodejs.org"

step "go vet"
go vet ./...
step "go test"
go test ./...
step "generated files match isom.yaml"
go generate ./...
if ! git diff --exit-code -- src/ || [ -n "$(git ls-files --others --exclude-standard -- src/)" ]; then
  git ls-files --others --exclude-standard -- src/
  echo "check: go generate changed src/; commit the result" >&2
  exit 1
fi
step "npm ci"
npm ci --ignore-scripts
step "npm run build"
npm run build
step "npm test"
npm test

printf '\ncheck: passed\n'

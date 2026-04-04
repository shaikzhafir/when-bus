#!/usr/bin/env bash
# Hot-reload the dev server when Go or template files change (requires entr).
# Usage: ./reload.sh   or   make run   (make run-plain = no hot reload)

set -euo pipefail
cd "$(dirname "$0")"

if ! command -v entr >/dev/null 2>&1; then
	echo "entr is required for hot reload. Install: brew install entr" >&2
	exit 1
fi

set -a
[ -f .env ] && . ./.env
set +a

find . \
	-type f \
	\( -name '*.go' -o -name '*.html' -o -name 'openapi.yaml' \) \
	! -path './.git/*' \
	! -path './bin/*' \
	! -name '*_test.go' \
	-print | entr -r go run ./cmd/server

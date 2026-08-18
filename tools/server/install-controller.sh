#!/usr/bin/env bash
set -euo pipefail
ROOT="${FABRIC_ROOT:-/opt/zorin-fabric}"
DEPLOY="$ROOT/deploy/controller"

if [[ $EUID -ne 0 ]]; then echo "Run as root (or sudo)." >&2; exit 1; fi
command -v docker >/dev/null || { echo "Docker is required" >&2; exit 1; }
docker compose version >/dev/null
[[ -d "$ROOT/.git" ]] || { echo "$ROOT is not a cloned zorin-fabric repository" >&2; exit 1; }

for f in fabric-auth-key.txt remnawave-api-token.txt; do
  [[ -f "$DEPLOY/secrets/$f" ]] || { echo "Missing secret: $DEPLOY/secrets/$f" >&2; exit 1; }
  chmod 600 "$DEPLOY/secrets/$f"
done

cd "$DEPLOY"
docker compose up -d --build

echo
docker compose ps

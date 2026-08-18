#!/usr/bin/env bash
set -euo pipefail

ROOT="${FABRIC_ROOT:-/opt/zorin-fabric}"
VCONNECT_ROOT="${VCONNECT_ROOT:-/opt/vconnect}"
DEPLOY="$ROOT/deploy/ru-edge"

if [[ $EUID -ne 0 ]]; then
  echo "Run as root (or sudo)." >&2
  exit 1
fi

for cmd in docker git; do
  command -v "$cmd" >/dev/null || { echo "Missing command: $cmd" >&2; exit 1; }
done

docker compose version >/dev/null

[[ -d "$ROOT/.git" ]] || { echo "$ROOT is not a cloned zorin-fabric repository" >&2; exit 1; }
[[ -d "$VCONNECT_ROOT/.git" ]] || { echo "$VCONNECT_ROOT is not a cloned vconnect repository" >&2; exit 1; }
[[ -f "$DEPLOY/.env" ]] || { echo "Missing $DEPLOY/.env (copy .env.example and edit it)" >&2; exit 1; }

for f in fabric-auth-key.txt remnawave-api-token.txt vault-key.txt cookies-yandex.json cookies-vk.json edge-heartbeat-token.txt; do
  [[ -f "$DEPLOY/secrets/$f" ]] || { echo "Missing secret: $DEPLOY/secrets/$f" >&2; exit 1; }
  chmod 600 "$DEPLOY/secrets/$f"
done

export VCONNECT_SOURCE="$VCONNECT_ROOT"
cd "$DEPLOY"
docker compose -f docker-compose.yml -f docker-compose.build.yml build --pull
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d

echo
docker compose ps

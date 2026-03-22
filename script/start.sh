#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
DEPLOY_DIR="$ROOT_DIR/deploy"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker not found"
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "docker compose not available"
  exit 1
fi

cd "$DEPLOY_DIR"

# 加载前端镜像
if [ -f "front-web/images/ran-feed-front-latest.tar" ]; then
  echo "Loading frontend image..."
  docker load -i front-web/images/ran-feed-front-latest.tar
fi

docker compose --env-file .env up -d --build

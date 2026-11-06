#!/usr/bin/env bash
# 在远端部署目录执行：构建、重启、等待健康检查，失败时恢复旧镜像。
set -Eeuo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-deploy/docker-compose.local.yml}"
SERVICE="${SERVICE:-sub2api}"
IMAGE="${IMAGE:-sub2api:local}"
HEALTH_TIMEOUT_SECONDS="${HEALTH_TIMEOUT_SECONDS:-180}"

if [[ ! -f "$COMPOSE_FILE" ]]; then
  echo "部署失败：找不到 $COMPOSE_FILE" >&2
  exit 1
fi
if [[ ! -f "deploy/.env" ]]; then
  echo "部署失败：找不到 deploy/.env，拒绝使用未配置环境变量的生产部署" >&2
  exit 1
fi

compose=(docker compose -f "$COMPOSE_FILE")
image_repository="${IMAGE%:*}"
rollback_image="${image_repository}:rollback-$(date -u +%Y%m%d%H%M%S)"
rollback_ready=0
service_updated=0

wait_for_health() {
  local timeout_seconds="${1:-$HEALTH_TIMEOUT_SECONDS}"
  local deadline=$((SECONDS + timeout_seconds))
  local container_id health_status

  while (( SECONDS < deadline )); do
    container_id="$("${compose[@]}" ps -q "$SERVICE" 2>/dev/null || true)"
    if [[ -n "$container_id" ]]; then
      health_status="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container_id" 2>/dev/null || true)"
      case "$health_status" in
        healthy)
          return 0
          ;;
        unhealthy|dead|exited)
          echo "健康检查失败：容器状态为 $health_status" >&2
          return 1
          ;;
      esac
    fi
    sleep 2
  done

  echo "健康检查超时：${timeout_seconds}s 内未达到 healthy" >&2
  return 1
}

rollback() {
  local exit_code=$?
  trap - ERR
  echo "发布失败，开始恢复旧版本。" >&2
  if (( service_updated == 1 && rollback_ready == 1 )); then
    docker tag "$rollback_image" "$IMAGE" || true
    "${compose[@]}" up -d --no-deps --force-recreate "$SERVICE" || true
    wait_for_health 60 || true
  fi
  "${compose[@]}" logs --tail=200 "$SERVICE" || true
  exit "$exit_code"
}

trap rollback ERR

"${compose[@]}" config --quiet
old_image_id="$(docker image inspect --format '{{.Id}}' "$IMAGE" 2>/dev/null || true)"
if [[ -n "$old_image_id" ]]; then
  docker tag "$IMAGE" "$rollback_image"
  rollback_ready=1
fi

echo "开始构建 $SERVICE..."
"${compose[@]}" build "$SERVICE"
echo "启动 $SERVICE..."
"${compose[@]}" up -d --no-deps "$SERVICE"
service_updated=1
echo "等待健康检查..."
wait_for_health

trap - ERR
echo "发布成功：$SERVICE 已通过健康检查。"
if (( rollback_ready == 1 )); then
  echo "旧镜像保留为：$rollback_image"
fi

#!/usr/bin/env bash
# 在本地仓库执行：同步源码到远端，再调用远端的健康发布脚本。
set -Eeuo pipefail

usage() {
  echo "用法：$0 user@host [remote_dir] [ssh_key]"
  echo "示例：$0 root@107.175.50.44 /opt/sub2api ~/.ssh/id_ed25519"
}

if [[ $# -lt 1 || $# -gt 3 ]]; then
  usage >&2
  exit 2
fi

REMOTE_HOST="$1"
REMOTE_DIR="${2:-/opt/sub2api}"
SSH_KEY="${3:-$HOME/.ssh/id_ed25519}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

if [[ ! -f "$SSH_KEY" ]]; then
  echo "找不到 SSH 私钥：$SSH_KEY" >&2
  exit 1
fi
if [[ ! "$REMOTE_DIR" =~ ^/[A-Za-z0-9._/-]+$ ]]; then
  echo "远端目录格式不安全：$REMOTE_DIR" >&2
  exit 1
fi
command -v rsync >/dev/null || { echo "本机缺少 rsync" >&2; exit 1; }
command -v ssh >/dev/null || { echo "本机缺少 ssh" >&2; exit 1; }

ssh_options=(-i "$SSH_KEY" -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -o ConnectTimeout=15)
rsync_options=(-az --delete-delay --itemize-changes -e "ssh ${ssh_options[*]}")
rsync_options+=(
  --exclude=.git/
  --exclude=node_modules/
  --exclude=frontend/node_modules/
  --exclude=backend/vendor/
  --exclude=deploy/.env
  --exclude=deploy/data/
  --exclude=deploy/postgres_data/
  --exclude=deploy/redis_data/
  --exclude=backend/server
)

echo "同步源码到 ${REMOTE_HOST}:${REMOTE_DIR}..."
rsync "${rsync_options[@]}" "$REPO_DIR/" "${REMOTE_HOST}:${REMOTE_DIR}/"

printf -v remote_command 'cd %q && bash %q' "$REMOTE_DIR" "deploy/update-local.sh"
echo "远端开始构建并执行健康检查..."
ssh "${ssh_options[@]}" "$REMOTE_HOST" "$remote_command"

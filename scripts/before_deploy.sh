#!/usr/bin/env bash
set -euo pipefail

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] [before_deploy] $*"
}

# 应用发布目录，CodePipeline 会把产物复制到这里。
APP_DIR="/srv/project/turtlepoll"
# 应用运行用户和用户组。
APP_USER="ubuntu"
APP_GROUP="ubuntu"
# 预留运行时数据目录，避免后续上传/索引等数据散落到临时目录。
DATA_DIR="/srv/project/turtlepoll/data"
# 与 bbs-go.example.yaml 中的日志路径保持一致。
LOG_DIR="/data/logs"

log "start"
log "whoami=$(whoami), pwd=$(pwd)"
log "APP_DIR=$APP_DIR, DATA_DIR=$DATA_DIR, LOG_DIR=$LOG_DIR"

# 确保部署、数据、日志目录都存在。
mkdir -p "$APP_DIR" "$DATA_DIR" "$LOG_DIR"
log "ensured deployment directories"

# 目录由 root 创建，但应用后续应由 ubuntu 用户读写。
chown -R "$APP_USER:$APP_GROUP" "$APP_DIR" "$DATA_DIR" "$LOG_DIR"
log "changed ownership to $APP_USER:$APP_GROUP"

# 如果 systemd 中已经存在 bbs-go 服务，先停止旧进程；首次部署没有服务时忽略错误。
if command -v systemctl >/dev/null 2>&1; then
  log "stopping existing bbs-go systemd service if present"
  systemctl stop bbs-go || true
else
  log "systemctl not found, skip service stop"
fi

log "done"

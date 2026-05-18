#!/usr/bin/env bash
set -euo pipefail

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] [after_deploy] $*"
}

# 应用发布目录，需与 deployspec.yml 的 destination 保持一致。
APP_DIR="/srv/project/turtlepoll"
# 应用运行用户和用户组。
APP_USER="ubuntu"
APP_GROUP="ubuntu"
# systemd 服务中直接启动流水线生成的 Linux 二进制。
APP_BIN="$APP_DIR/bbs-go-linux"
# systemd 服务文件路径。
SERVICE_FILE="/etc/systemd/system/bbs-go.service"

log "start"
log "whoami=$(whoami), pwd=$(pwd)"
log "APP_DIR=$APP_DIR, APP_BIN=$APP_BIN, SERVICE_FILE=$SERVICE_FILE"

cd "$APP_DIR"
log "changed directory to $(pwd)"

log "top-level files in $APP_DIR:"
find "$APP_DIR" -maxdepth 2 -mindepth 1 -printf "%M %u:%g %s %p\n" | sort | head -120 || true

log "searching for bbs-go-linux under $APP_DIR:"
find "$APP_DIR" -maxdepth 5 -name "bbs-go-linux" -printf "%M %u:%g %s %p\n" | sort || true

# 流水线产物必须包含 make buildlinux 生成的 bbs-go-linux。
if [ ! -f "$APP_BIN" ]; then
  log "binary not found at expected path"
  echo "missing binary: $APP_BIN. Please build bbs-go-linux before EC2 deploy." >&2
  exit 1
fi
log "found binary at $APP_BIN"

# 确保服务二进制具有可执行权限。
if [ ! -x "$APP_BIN" ]; then
  log "binary is not executable, chmod +x $APP_BIN"
  chmod +x "$APP_BIN"
else
  log "binary is already executable"
fi

# 只在首次部署缺少配置时创建示例配置，避免覆盖 EC2 上已有的生产配置。
if [ ! -f "$APP_DIR/bbs-go.yaml" ] && [ -f "$APP_DIR/bbs-go.example.yaml" ]; then
  log "bbs-go.yaml missing, copy from bbs-go.example.yaml"
  cp "$APP_DIR/bbs-go.example.yaml" "$APP_DIR/bbs-go.yaml"
elif [ -f "$APP_DIR/bbs-go.yaml" ]; then
  log "bbs-go.yaml exists, keep current production config"
else
  log "bbs-go.yaml and bbs-go.example.yaml are both missing"
fi

# CodePipeline/SSM 通常以 root 写入文件，这里统一交还给应用用户。
chown -R "$APP_USER:$APP_GROUP" "$APP_DIR" /data/logs
log "changed ownership to $APP_USER:$APP_GROUP"

# 当前部署方式依赖 systemd 管理 Go 服务进程。
if ! command -v systemctl >/dev/null 2>&1; then
  echo "systemd is required on the EC2 instance." >&2
  exit 1
fi
log "systemctl found at $(command -v systemctl)"

# 创建或更新 systemd 服务，确保机器重启后服务能自动恢复。
log "writing systemd service file $SERVICE_FILE"
cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=bbs-go service
After=network.target

[Service]
Type=simple
User=$APP_USER
Group=$APP_GROUP
WorkingDirectory=$APP_DIR
ExecStart=$APP_BIN
Environment=BBSGO_ENV=prod
Restart=always
RestartSec=5
StandardOutput=append:/data/logs/bbs-go.stdout.log
StandardError=append:/data/logs/bbs-go.stderr.log

[Install]
WantedBy=multi-user.target
EOF

# 重新加载 systemd 配置，设置开机自启，并重启服务。
log "systemd service file content:"
sed -n '1,120p' "$SERVICE_FILE"

log "running systemctl daemon-reload"
systemctl daemon-reload
log "running systemctl enable bbs-go"
systemctl enable bbs-go
log "running systemctl restart bbs-go"
systemctl restart bbs-go

log "systemctl status bbs-go:"
systemctl --no-pager --full status bbs-go || true

log "recent bbs-go journal:"
journalctl -u bbs-go --no-pager -n 80 || true

log "done"

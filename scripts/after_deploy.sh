#!/usr/bin/env bash
set -euo pipefail

# 应用发布目录，需与 deployspec.yml 的 destination 保持一致。
APP_DIR="/srv/project/turtlepoll"
# 应用运行用户和用户组。
APP_USER="ubuntu"
APP_GROUP="ubuntu"
# systemd 服务中直接启动流水线生成的 Linux 二进制。
APP_BIN="$APP_DIR/bbs-go-linux"
# systemd 服务文件路径。
SERVICE_FILE="/etc/systemd/system/bbs-go.service"

cd "$APP_DIR"

# 流水线产物必须包含 make buildlinux 生成的 bbs-go-linux。
if [ ! -f "$APP_BIN" ]; then
  echo "missing binary: $APP_BIN. Please build bbs-go-linux before EC2 deploy." >&2
  exit 1
fi

# 确保服务二进制具有可执行权限。
if [ ! -x "$APP_BIN" ]; then
  chmod +x "$APP_BIN"
fi

# 只在首次部署缺少配置时创建示例配置，避免覆盖 EC2 上已有的生产配置。
if [ ! -f "$APP_DIR/bbs-go.yaml" ] && [ -f "$APP_DIR/bbs-go.example.yaml" ]; then
  cp "$APP_DIR/bbs-go.example.yaml" "$APP_DIR/bbs-go.yaml"
fi

# CodePipeline/SSM 通常以 root 写入文件，这里统一交还给应用用户。
chown -R "$APP_USER:$APP_GROUP" "$APP_DIR" /data/logs

# 当前部署方式依赖 systemd 管理 Go 服务进程。
if ! command -v systemctl >/dev/null 2>&1; then
  echo "systemd is required on the EC2 instance." >&2
  exit 1
fi

# 创建或更新 systemd 服务，确保机器重启后服务能自动恢复。
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
systemctl daemon-reload
systemctl enable bbs-go
systemctl restart bbs-go

# 概述
此文档介绍在 部署服务器时，需要安装的软件。

# 应用服务器
## 安装 Nginx

```bash
sudo apt update
sudo apt install -y nginx
```

测试
```bash
curl http://localhost
```

## 安装 go
```bash
# 使用官网地址（海外服务器速度更快）
wget -c https://go.dev/dl/go1.21.5.linux-amd64.tar.gz -O /tmp/go1.21.5.linux-amd64.tar.gz
```

```bash
# 删除旧版本（如果有）
sudo rm -rf /usr/local/go

# 解压到 /usr/local
sudo tar -C /usr/local -xzf /tmp/go1.21.5.linux-amd64.tar.gz
```

```bash
# 添加环境变量到 ~/.bashrc
cat >> ~/.bashrc << 'EOF'

# Go Environment
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
EOF

# 使配置生效
source ~/.bashrc
```
```bash
go version
```

## 安装 make
```bash
sudo apt install make -y
make --version
```

## 数据库客户端
```bash
sudo apt install -y postgresql-client
```

# 数据库服务器
## 安装PG
```bash
sudo apt install -y postgresql postgresql-contrib
```
## 创建用户 数据库
```bash
sudo -u postgres psql <<EOF
-- 创建用户
CREATE USER appuser WITH PASSWORD 'root';

-- 创建数据库
CREATE DATABASE turtlepoll OWNER appuser;

-- 连接到数据库并授予完整权限
\c turtlepoll
GRANT ALL ON SCHEMA public TO appuser;
GRANT ALL PRIVILEGES ON DATABASE turtlepoll TO appuser;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO appuser;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO appuser;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO appuser;

-- 确保将来创建的表也有权限
GRANT CREATE ON SCHEMA public TO appuser;
EOF
```

## 开通监听
```bash
# 修改 postgresql.conf（自动去除注释并修改）
sudo sed -i 's/^#listen_addresses = .*/listen_addresses = '\''*'\''/' /etc/postgresql/*/main/postgresql.conf

# 如果没有匹配到，直接追加
sudo grep -q "listen_addresses = '*'" /etc/postgresql/*/main/postgresql.conf || sudo sh -c "echo \"listen_addresses = '*'\" >> /etc/postgresql/*/main/postgresql.conf"

# 添加 pg_hba 规则
sudo sh -c "echo 'host    all    all    0.0.0.0/0    md5' >> /etc/postgresql/*/main/pg_hba.conf"

# 重启
sudo systemctl restart postgresql
```

# web证书安装
## 安装certbot
```bash
sudo apt install certbot python3-certbot-nginx -y
```
## 一键申请
```bash
sudo certbot --nginx -d www.turtkepoll.com
```
## 查看nginx状态
```bash
sudo nginx -t
```

## 重启nginx
```bash
sudo systemctl reload nginx
```

## 测试自动续期：运行下面的命令，模拟证书续期过程，检查配置是否正确
```bash
sudo certbot renew --dry-run
```
如果看到 The dry run was successful，就代表自动续期已经配置好了，证书到期前会自动更新
## 确认定时器
```bash
sudo systemctl status certbot.timer
```

# aws Amplify 兼容

```bash
#!/bin/bash

# 写入正确的配置
sudo tee /etc/nginx/sites-available/www.turtkepoll.com > /dev/null << 'EOF'
server {
    listen 80;
    listen [::]:80;
    server_name www.turtkepoll.com turtkepoll.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name www.turtkepoll.com turtkepoll.com;

    ssl_certificate /etc/letsencrypt/live/www.turtkepoll.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/www.turtkepoll.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    resolver 8.8.8.8 8.8.4.4 valid=300s;
    resolver_timeout 5s;

    # /api/ 路径转发到 bbs-go
    location /api/ {
        proxy_pass http://127.0.0.1:8082;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 其他路径转发到 AWS Amplify
    location / {
        set $amplify_backend "main.d3t0t4yyv8sthr.amplifyapp.com";
        proxy_pass https://$amplify_backend;
        proxy_set_header Host $amplify_backend;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_ssl_protocols TLSv1.2 TLSv1.3;
        proxy_ssl_server_name on;
        proxy_buffering off;
        proxy_http_version 1.1;
    }
}
EOF

# 测试配置
sudo nginx -t

# 重载 Nginx
sudo systemctl reload nginx

# 测试 API
echo "测试 API 代理:"
curl -k https://www.turtkepoll.com/api/user/current
```

# 无域名兼容 aws Amplify

```bash
#!/bin/bash

# 设置 test 环境参数
SERVER_IP="52.220.26.101"
AMPLIFY_TEST_URL="test.d3t0t4yyv8sthr.amplifyapp.com"
BACKEND_PORT="8082"

# 1. 安装 Nginx（如果未安装）
if ! command -v nginx &> /dev/null; then
    sudo apt update
    sudo apt install nginx -y
fi

# 2. 创建自签名 SSL 证书（用于 IP 访问）
sudo mkdir -p /etc/nginx/ssl
sudo openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout /etc/nginx/ssl/selfsigned.key \
    -out /etc/nginx/ssl/selfsigned.crt \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=Test/CN=${SERVER_IP}"

# 3. 创建 Nginx 配置文件
sudo tee /etc/nginx/sites-available/test-ip-proxy > /dev/null << 'EOF'
# HTTP 自动跳转到 HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name _;
    return 301 https://$host$request_uri;
}

# HTTPS 服务器 - 代理前端 Amplify
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name _;

    # 自签名证书
    ssl_certificate /etc/nginx/ssl/selfsigned.crt;
    ssl_certificate_key /etc/nginx/ssl/selfsigned.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # DNS 解析器
    resolver 8.8.8.8 8.8.4.4 valid=300s;
    resolver_timeout 5s;

    # ========== 前端 Amplify test 分支代理 ==========
    location / {
        set $amplify_backend "test.d3t0t4yyv8sthr.amplifyapp.com";
        proxy_pass https://$amplify_backend;
        
        # 关键：Host 头设置为 Amplify 应用的地址
        proxy_set_header Host $amplify_backend;
        
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # SSL 和代理优化
        proxy_ssl_protocols TLSv1.2 TLSv1.3;
        proxy_ssl_server_name on;
        proxy_buffering off;
        proxy_http_version 1.1;
    }
}
EOF

# 4. 禁用默认配置并启用新配置
sudo rm -f /etc/nginx/sites-enabled/default
sudo ln -sf /etc/nginx/sites-available/test-ip-proxy /etc/nginx/sites-enabled/

# 5. 测试配置语法
sudo nginx -t

# 6. 重载 Nginx
if [ $? -eq 0 ]; then
    sudo systemctl reload nginx
    echo ""
    echo "=========================================="
    echo "✅ Nginx 配置成功！"
    echo "=========================================="
    echo "前端 Amplify: https://${SERVER_IP}"
    echo "后端 API 地址: http://${SERVER_IP}:${BACKEND_PORT}"
    echo "=========================================="
    echo ""
    echo "⚠️  注意事项："
    echo "1. 使用的是自签名证书，浏览器会显示安全警告"
    echo "2. 点击'高级' -> '继续访问' 即可"
    echo "3. 确保后端服务已在 ${BACKEND_PORT} 端口运行"
else
    echo "❌ 配置有误，请检查"
fi

# 7. 检查后端服务状态
echo ""
echo "检查后端服务 (${BACKEND_PORT} 端口):"
sudo ss -tlnp | grep ${BACKEND_PORT} || echo "⚠️  后端服务未启动，请启动你的后端服务"

# 8. 测试本地代理
echo ""
echo "测试前端代理:"
curl -k -I https://${SERVER_IP} 2>/dev/null | head -3
```

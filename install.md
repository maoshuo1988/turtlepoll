# 安装 Nginx

```bash
sudo apt update
sudo apt install -y nginx
```

测试
```bash
curl http://localhost
```

# 安装 go
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